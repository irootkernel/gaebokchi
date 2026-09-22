package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// Terminal connection errors. A connection dies at most once with one of
// these; every pending call then fails immediately.
var (
	// ErrEOF reports the child closing stdout before a response arrived.
	ErrEOF = errors.New("rpc: stream closed before response")
	// ErrServerRequest reports a server-initiated request. It was
	// rejected with a dialect-correct error, and no host capability was
	// executed.
	ErrServerRequest = errors.New("rpc: server-initiated request rejected")
	// ErrMalformedMessage reports a structurally invalid frame.
	ErrMalformedMessage = errors.New("rpc: malformed message")
	// ErrConnDead reports using a connection after a terminal error.
	ErrConnDead = errors.New("rpc: connection is dead")
	// ErrBusy reports a second concurrent call on a connection that
	// allows one outstanding request.
	ErrBusy = errors.New("rpc: another request is outstanding")
)

// RPCError is a well-formed remote error delivered as the response to a
// client request.
type RPCError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc: remote error %d: %s", e.Code, e.Message)
}

// Response is one correlated reply: exactly one of Result or Err is set.
type Response struct {
	Result json.RawMessage
	Err    *RPCError
}

// RemoteError returns the remote error when the response carries one.
func (r *Response) RemoteError() *RPCError { return r.Err }

type pendingReply struct {
	resp Response
	// at records when the reply was routed off the wire. A response
	// first established after the caller's deadline must not become a
	// clean result, while one established in time survives any later
	// cleanup delay.
	at time.Time
}

// Conn is a correlated stdio JSON connection using one dialect codec.
// It allows a single outstanding client request and ignores unrelated
// well-formed notifications and unknown response ids.
type Conn struct {
	codec Codec
	w     io.Writer
	wmu   sync.Mutex
	fr    *FrameReader

	abort func() // tears the transport down to unblock readers/writers

	mu       sync.Mutex
	nextID   int64
	pending  map[int64]chan pendingReply
	busy     bool
	dead     error
	deadOnce sync.Once
	deadCh   chan struct{}
}

// NewConn starts the read loop for a connection over the given pipes.
// abort is invoked whenever a call is cancelled or the connection dies;
// it must unblock any stuck read or write (e.g. by closing the child).
func NewConn(w io.Writer, r io.Reader, codec Codec, abort func()) *Conn {
	if abort == nil {
		abort = func() {}
	}
	c := &Conn{
		codec:  codec,
		w:      w,
		fr:     NewFrameReader(r),
		abort:  abort,
		pending: make(map[int64]chan pendingReply),
		deadCh: make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// Close stops the connection from accepting new work. Pending calls are
// failed and the transport is torn down via the abort hook exactly once;
// Close blocks for at most the abort hook's bounded cleanup.
func (c *Conn) Close() {
	c.die(ErrConnDead)
}

func (c *Conn) die(err error) {
	c.deadOnce.Do(func() {
		c.mu.Lock()
		c.dead = err
		c.mu.Unlock()
		close(c.deadCh)
		// Terminal failure tears the owned transport down so blocked
		// readers and writers are unblocked. The hook must be idempotent
		// and must not call back into this Conn.
		c.abort()
	})
}

func (c *Conn) deadError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dead == nil {
		return ErrConnDead
	}
	return c.dead
}

func (c *Conn) readLoop() {
	for {
		line, err := c.fr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.die(ErrEOF)
			} else {
				c.die(fmt.Errorf("%w: %v", ErrMalformedMessage, err))
			}
			return
		}
		if err := c.handleLine(line); err != nil {
			c.die(err)
			return
		}
	}
}

// incoming preserves presence of the keys that matter for classification.
type incoming struct {
	keys   map[string]json.RawMessage
	id     json.RawMessage
	method string
}

func parseIncoming(line []byte) (*incoming, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		return nil, fmt.Errorf("%w: not a JSON object", ErrMalformedMessage)
	}
	// Top-level arrays/strings/numbers fail Unmarshal into a map, so
	// reaching here means the frame was an object (batches rejected).
	in := &incoming{keys: obj}
	if m, ok := obj["method"]; ok {
		if err := json.Unmarshal(m, &in.method); err != nil {
			return nil, fmt.Errorf("%w: method is not a string", ErrMalformedMessage)
		}
	}
	in.id = obj["id"]
	return in, nil
}

func (c *Conn) handleLine(line []byte) error {
	in, err := parseIncoming(line)
	if err != nil {
		return err
	}
	hasMethod := in.keys["method"] != nil
	hasID := in.keys["id"] != nil

	switch {
	case hasMethod && hasID:
		// Server-initiated request: never execute any capability.
		// Reject once with the exact received id, then abort. The
		// rejection write is bounded so a full pipe cannot wedge the
		// read loop.
		c.sendRejection(in.id)
		return fmt.Errorf("%w: method %q", ErrServerRequest, in.method)
	case hasMethod:
		// Well-formed unrelated notification: ignore.
		return nil
	case hasID:
		return c.deliverResponse(in)
	default:
		return fmt.Errorf("%w: object has neither method nor id", ErrMalformedMessage)
	}
}

func (c *Conn) deliverResponse(in *incoming) error {
	if err := c.codec.ValidateResponseEnvelope(in.keys); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedMessage, err)
	}
	rawResult, resultPresent := in.keys["result"]
	rawErr, errorPresent := in.keys["error"]
	if resultPresent && errorPresent {
		return fmt.Errorf("%w: response carries both result and error", ErrMalformedMessage)
	}
	if !resultPresent && !errorPresent {
		return fmt.Errorf("%w: response has neither result nor error", ErrMalformedMessage)
	}
	if errorPresent {
		// An error member must be exactly one structurally valid
		// alternative: error:null is a malformed response, never a
		// success with a nil result.
		if string(rawErr) == "null" {
			return fmt.Errorf("%w: response carries error:null without a result", ErrMalformedMessage)
		}
		var we wireErrorIn
		if err := json.Unmarshal(rawErr, &we); err != nil {
			return fmt.Errorf("%w: error member is malformed", ErrMalformedMessage)
		}
		if we.Code == nil || we.Message == nil {
			return fmt.Errorf("%w: error member lacks its required typed fields", ErrMalformedMessage)
		}
		return c.route(in.id, Response{Err: &RPCError{Code: *we.Code, Message: *we.Message, Data: we.Data}})
	}
	// result may be a JSON null; the adapter decides if that is valid
	// for the operation's schema.
	return c.route(in.id, Response{Result: rawResult})
}

// route matches a response id against pending calls. Unknown or
// duplicate ids never satisfy a pending request.
func (c *Conn) route(idRaw json.RawMessage, resp Response) error {
	var id int64
	if err := json.Unmarshal(idRaw, &id); err != nil {
		return nil // string/odd ids cannot match our numeric requests
	}
	c.mu.Lock()
	ch, ok := c.pending[id]
	if ok {
		delete(c.pending, id)
	}
	c.mu.Unlock()
	if !ok {
		return nil
	}
	ch <- pendingReply{resp: resp, at: time.Now()}
	return nil
}

// Call sends one request and waits for its correlated response, context
// cancellation, or connection death. One outstanding request per
// connection is enforced. A response counts as established at the moment
// it is routed off the wire: one routed after the context deadline fails
// the call even if it wins the wake-up race, while one routed in time is
// returned regardless of how long later processing takes.
func (c *Conn) Call(ctx context.Context, method string, params any) (*Response, error) {
	c.mu.Lock()
	if c.dead != nil {
		c.mu.Unlock()
		return nil, c.dead
	}
	if c.busy {
		c.mu.Unlock()
		return nil, ErrBusy
	}
	c.busy = true
	c.nextID++
	id := c.nextID
	ch := make(chan pendingReply, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	unregister := func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.busy = false
		c.mu.Unlock()
	}
	rejectLate := func() (*Response, error) {
		// The response was first established after the deadline.
		unregister()
		c.abort()
		return nil, ctx.Err()
	}

	payload, err := c.codec.EncodeRequest(id, method, params)
	if err != nil {
		unregister()
		return nil, err
	}

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- c.writeLine(payload)
	}()

	deadline, hasDeadline := ctx.Deadline()

	// Wait for the write, the correlated reply, cancellation, or
	// connection death. When the connection dies after the reply was
	// routed (e.g. a server that closes stdout right after answering),
	// the buffered channel still holds the real reply, so check it
	// before reporting the transport error.
	for {
		select {
		case reply := <-ch:
			if hasDeadline && reply.at.After(deadline) {
				return rejectLate()
			}
			c.mu.Lock()
			c.busy = false
			c.mu.Unlock()
			return &reply.resp, nil
		case werr := <-writeDone:
			if werr != nil {
				unregister()
				c.die(fmt.Errorf("rpc: write failed: %v", werr))
				return nil, c.deadError()
			}
		case <-ctx.Done():
			unregister()
			c.abort() // unblocks the writer and any reader
			return nil, ctx.Err()
		case <-c.deadCh:
			select {
			case reply := <-ch:
				if hasDeadline && reply.at.After(deadline) {
					return rejectLate()
				}
				c.mu.Lock()
				c.busy = false
				c.mu.Unlock()
				return &reply.resp, nil
			default:
				unregister()
				return nil, c.deadError()
			}
		}
	}
}

// Notify sends a notification the dialect codec renders without an id.
// A blocked write is cancellable: on context termination the transport is
// aborted so an owned writer can never wedge the caller past its budget.
func (c *Conn) Notify(ctx context.Context, method string, params any) error {
	payload, err := c.codec.EncodeNotification(method, params)
	if err != nil {
		return err
	}
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- c.writeLine(payload)
	}()
	select {
	case err := <-writeDone:
		return err
	case <-ctx.Done():
		c.abort()
		return ctx.Err()
	case <-c.deadCh:
		return c.deadError()
	}
}

// rejectionWriteBudget bounds the attempt to deliver a dialect-correct
// rejection for a server-initiated request.
const rejectionWriteBudget = 500 * time.Millisecond

func (c *Conn) sendRejection(id json.RawMessage) {
	resp, err := c.codec.EncodeErrorResponse(id, MethodNotFoundCode, "gaebokchi is a read-only client and does not accept server requests")
	if err != nil {
		return
	}
	done := make(chan struct{})
	go func() {
		_ = c.writeLine(resp)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(rejectionWriteBudget):
		// The writer stays blocked on a full pipe. The die() that
		// follows this rejection in the read loop aborts the transport,
		// which unblocks and releases the abandoned writer; it cannot
		// hold the write lock indefinitely.
	}
}

func (c *Conn) writeLine(payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	buf := make([]byte, 0, len(payload)+1)
	buf = append(buf, payload...)
	buf = append(buf, '\n')
	_, err := c.w.Write(buf)
	return err
}

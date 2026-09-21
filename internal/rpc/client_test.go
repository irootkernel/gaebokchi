package rpc

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gaebokchi/internal/testutil"
)

// pipePair gives the client a writer/reader pair plus the server side.
type pipePair struct {
	clientW *io.PipeWriter
	clientR *io.PipeReader
	serverW *io.PipeWriter
	serverR *io.PipeReader
	abort   *abortLatch
}

type abortLatch struct {
	mu sync.Mutex
	n  int
	f  func()
}

func (a *abortLatch) fire() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n++
	if a.f != nil {
		a.f()
	}
}

func (a *abortLatch) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.n
}

func newPipePair(t *testing.T) *pipePair {
	t.Helper()
	cr, sw := io.Pipe() // server writes -> client reads
	sr, cw := io.Pipe() // client writes -> server reads
	latch := &abortLatch{}
	pp := &pipePair{clientW: cw, clientR: cr, serverW: sw, serverR: sr, abort: latch}
	latch.f = func() {
		cw.CloseWithError(errors.New("aborted"))
		cr.CloseWithError(errors.New("aborted"))
	}
	return pp
}

func (pp *pipePair) conn(codec Codec) *Conn {
	return NewConn(pp.clientW, pp.clientR, codec, pp.abort.fire)
}

func TestFakeHelperProcess(t *testing.T) {
	if os.Getenv("GAEBOKCHI_FAKE") != "1" {
		return
	}
	if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestCallCorrelationWithNotificationsAndUnknownIDs(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx := context.Background()

	go func() {
		// Server: read the request, then send an unrelated notification,
		// a response with an unknown id, and finally the real reply.
		line, err := testutilReadLine(pp.serverR)
		if err != nil {
			return
		}
		if !strings.Contains(line, `"method":"account/read"`) || !strings.Contains(line, `"id":1`) {
			pp.serverW.CloseWithError(errors.New("unexpected request: " + line))
			return
		}
		io.WriteString(pp.serverW, "{\"method\":\"remoteControl/status/changed\",\"params\":{}}\n")
		io.WriteString(pp.serverW, "{\"id\":99,\"result\":{}}\n")
		io.WriteString(pp.serverW, "{\"id\":1,\"result\":{\"account\":{\"type\":\"chatgpt\"}}}\n")
		pp.serverW.Close()
	}()

	resp, err := c.Call(ctx, "account/read", map[string]any{"refreshToken": false})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"account":{"type":"chatgpt"}}` {
		t.Fatalf("result = %s", resp.Result)
	}
	if resp.Err != nil {
		t.Fatalf("unexpected remote error: %v", resp.Err)
	}
}

func TestCallRemoteError(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"auth required"}}`+"\n")
		pp.serverW.Close()
	}()
	resp, err := c.Call(context.Background(), "authenticate", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	rerr := resp.RemoteError()
	if rerr == nil || rerr.Code != -32000 || rerr.Message != "auth required" {
		t.Fatalf("remote error = %+v", rerr)
	}
}

func TestEOFailsPendingCall(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		pp.serverW.Close() // EOF before response
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrEOF) {
		t.Fatalf("err = %v, want ErrEOF", err)
	}
}

func TestMalformedFrameFailsCall(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, "this is not json\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if err == nil || !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage", err)
	}
}

func TestBatchTopLevelRejected(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, "[{\"id\":1,\"result\":{}}]\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage", err)
	}
}

func TestBothResultAndErrorRejected(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"id":1,"result":{},"error":{"code":1,"message":"x"}}`+"\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v", err)
	}
}

func TestNullResultPreserved(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"id":1,"result":null}`+"\n")
		pp.serverW.Close()
	}()
	resp, err := c.Call(context.Background(), "account/read", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != "null" {
		t.Fatalf("null result not preserved: %q", resp.Result)
	}
}

func TestServerInitiatedRequestRejectedAndAborts(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	received := make(chan string, 1)
	go func() {
		// Consume the client's own request first.
		testutilReadLine(pp.serverR)
		// Then deliver the server-initiated request.
		io.WriteString(pp.serverW, `{"jsonrpc":"2.0","id":42,"method":"session/prompt","params":{"prompt":"run this"}}`+"\n")
		line, _ := testutilReadLine(pp.serverR)
		received <- line
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrServerRequest) {
		t.Fatalf("err = %v, want ErrServerRequest", err)
	}
	select {
	case line := <-received:
		if !strings.Contains(line, `"id":42`) || !strings.Contains(line, `-32601`) {
			t.Fatalf("rejection = %q", line)
		}
		if !strings.Contains(line, `"jsonrpc":"2.0"`) {
			t.Fatalf("acp rejection must carry jsonrpc: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no rejection sent")
	}
}

func TestCodexRejectionOmitsJSONRPC(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	received := make(chan string, 1)
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"id":7,"method":"tool/approve"}`+"\n")
		line, _ := testutilReadLine(pp.serverR)
		received <- line
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrServerRequest) {
		t.Fatalf("err = %v, want ErrServerRequest", err)
	}
	select {
	case line := <-received:
		if strings.Contains(line, "jsonrpc") {
			t.Fatalf("codex rejection must omit jsonrpc: %q", line)
		}
		if !strings.Contains(line, `"id":7`) {
			t.Fatalf("rejection must echo id: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no rejection sent")
	}
}

func TestContextCancellationAbortsTransport(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go func() {
		// Server reads the request but never answers.
		testutilReadLine(pp.serverR)
		time.Sleep(5 * time.Second)
	}()
	start := time.Now()
	_, err := c.Call(ctx, "initialize", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}
	if pp.abort.count() == 0 {
		t.Fatal("abort hook not fired")
	}
}

func TestBlockedWriteUnblockedByCancel(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	// Server never reads: the client's write fills the pipe buffer.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := c.Call(ctx, "initialize", map[string]any{"filler": strings.Repeat("x", 1<<20)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
}

func TestSingleOutstandingRequest(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Server side: consume the request, then deliver the reply.
	requestSeen := make(chan string, 1)
	go func() {
		line, _ := testutilReadLine(pp.serverR)
		requestSeen <- line
		io.WriteString(pp.serverW, `{"id":1,"result":{}}`+"\n")
	}()

	firstDone := make(chan error, 1)
	go func() {
		_, err := c.Call(ctx, "account/read", nil)
		firstDone <- err
	}()
	<-requestSeen

	if _, err := c.Call(ctx, "initialize", nil); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first call never completed")
	}
}

// testutilReadLine reads one line from r (test-local helper).
func testutilReadLine(r io.Reader) (string, error) {
	var buf []byte
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, one[0])
		}
		if err != nil {
			return string(buf), err
		}
	}
}

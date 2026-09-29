package rpc

import (
	"context"
	"errors"
	ginkgo "github.com/onsi/ginkgo/v2"
	"io"
	"strings"
	"sync"
	"time"
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
	mu    sync.Mutex
	n     int
	f     func()
	fired chan struct{}
}

func (a *abortLatch) fire() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n++
	if a.f != nil {
		a.f()
	}
	if a.n == 1 {
		close(a.fired)
	}
}

func (a *abortLatch) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.n
}

func newPipePair(t testT) *pipePair {
	t.Helper()
	cr, sw := io.Pipe() // server writes -> client reads
	sr, cw := io.Pipe() // client writes -> server reads
	latch := &abortLatch{fired: make(chan struct{})}
	pp := &pipePair{clientW: cw, clientR: cr, serverW: sw, serverR: sr, abort: latch}
	latch.f = func() {
		cw.CloseWithError(errors.New("aborted"))
		cr.CloseWithError(errors.New("aborted"))
	}
	return pp
}

func readRequest(t testT, r io.Reader) {
	t.Helper()
	if _, err := testutilReadLine(r); err != nil {
		t.Errorf("fake server could not read request: %v", err)
	}
}

func writeReply(t testT, w io.Writer, reply string) {
	t.Helper()
	if _, err := io.WriteString(w, reply); err != nil {
		t.Errorf("fake server could not write reply: %v", err)
	}
}

func (pp *pipePair) conn(codec Codec) *Conn {
	return NewConn(pp.clientW, pp.clientR, codec, pp.abort.fire)
}

var _ = ginkgo.It("TestCallCorrelationWithNotificationsAndUnknownIDs", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx := context.Background()

	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
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
		writeReply(t, pp.serverW, "{\"method\":\"remoteControl/status/changed\",\"params\":{}}\n")
		writeReply(t, pp.serverW, "{\"id\":99,\"result\":{}}\n")
		writeReply(t, pp.serverW, "{\"id\":1,\"result\":{\"account\":{\"type\":\"chatgpt\"}}}\n")
		_ = pp.serverW.Close()
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
})

var _ = ginkgo.It("TestCallRemoteError", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"auth required"}}`+"\n")
		_ = pp.serverW.Close()
	}()
	resp, err := c.Call(context.Background(), "authenticate", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	rerr := resp.RemoteError()
	if rerr == nil || rerr.Code != -32000 || rerr.Message != "auth required" {
		t.Fatalf("remote error = %+v", rerr)
	}
})

var _ = ginkgo.It("TestEOFailsPendingCall", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		_ = pp.serverW.Close() // EOF before response
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrEOF) {
		t.Fatalf("err = %v, want ErrEOF", err)
	}
})

var _ = ginkgo.It("TestMalformedFrameFailsCall", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, "this is not json\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if err == nil || !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage", err)
	}
})

var _ = ginkgo.It("TestBatchTopLevelRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, "[{\"id\":1,\"result\":{}}]\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage", err)
	}
})

var _ = ginkgo.It("TestBothResultAndErrorRejected", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"id":1,"result":{},"error":{"code":1,"message":"x"}}`+"\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v", err)
	}
})

var _ = ginkgo.It("TestNullResultPreserved", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"id":1,"result":null}`+"\n")
		_ = pp.serverW.Close()
	}()
	resp, err := c.Call(context.Background(), "account/read", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != "null" {
		t.Fatalf("null result not preserved: %q", resp.Result)
	}
})

var _ = ginkgo.It("TestServerInitiatedRequestRejectedAndAborts", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	received := make(chan string, 1)
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		// Consume the client's own request first.
		readRequest(t, pp.serverR)
		// Then deliver the server-initiated request.
		writeReply(t, pp.serverW, `{"jsonrpc":"2.0","id":42,"method":"session/prompt","params":{"prompt":"run this"}}`+"\n")
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
})

var _ = ginkgo.It("TestCodexRejectionOmitsJSONRPC", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	received := make(chan string, 1)
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"id":7,"method":"tool/approve"}`+"\n")
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
})

var _ = ginkgo.It("TestContextCancellationAbortsTransport", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		// Server reads the request but never answers.
		readRequest(t, pp.serverR)
		time.Sleep(5 * time.Second)
	}()
	start := time.Now()
	_, err := c.Call(ctx, "initialize", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}
	if pp.abort.count() == 0 {
		t.Fatal("abort hook not fired")
	}
})

var _ = ginkgo.It("TestBlockedWriteUnblockedByCancel", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	// Server never reads: the client's write fills the pipe buffer.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := c.Call(ctx, "initialize", map[string]any{"filler": strings.Repeat("x", 1<<20)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
})

var _ = ginkgo.It("TestSingleOutstandingRequest", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Server side: consume the request and hold the reply until the
	// busy assertion has run, so the first call is deterministically
	// still outstanding.
	requestSeen := make(chan struct{}, 1)
	release := make(chan struct{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		requestSeen <- struct{}{}
		<-release
		writeReply(t, pp.serverW, `{"id":1,"result":{}}`+"\n")
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
	close(release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first call never completed")
	}
})

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

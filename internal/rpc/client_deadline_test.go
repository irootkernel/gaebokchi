package rpc

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// TestLateRoutedReplyNeverSatisfiesCall pins the establishment rule: a
// response whose bytes are routed only after the context deadline must
// not become a clean result, whichever way the wake-up race resolves.
// This prevents the regression where a late write could win the select
// race against an already-expired context.
func TestLateRoutedReplyNeverSatisfiesCall(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go func() {
		testutilReadLine(pp.serverR)
		time.Sleep(300 * time.Millisecond) // reply crosses the deadline
		io.WriteString(pp.serverW, `{"id":1,"result":{}}`+"\n")
		pp.serverW.Close()
	}()
	start := time.Now()
	resp, err := c.Call(ctx, "initialize", nil)
	if err == nil {
		t.Fatalf("late reply accepted as success: %+v", resp)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("call took %v", elapsed)
	}
}

// TestTimelyReplyReturnedWhileConnectionLingers pins the other half of
// the rule: a reply routed before the deadline is returned even though
// the peer lingers afterwards (the slow-cleanup shape).
func TestTimelyReplyReturnedWhileConnectionLingers(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"id":1,"result":{"ok":true}}`+"\n")
		time.Sleep(400 * time.Millisecond) // linger after answering
		pp.serverW.Close()
	}()
	resp, err := c.Call(ctx, "initialize", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"ok":true}` {
		t.Fatalf("result = %s", resp.Result)
	}
}

// TestNotifyBlockedWriteCancellable covers the audit finding that a
// context-less notification write could wedge a whole query: a server
// that stops reading must not hold Notify past the caller's deadline.
func TestNotifyBlockedWriteCancellable(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	// The server never reads its stdin: the notification write blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := c.Notify(ctx, "initialized", map[string]any{"filler": strings.Repeat("x", 1<<20)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("notify took %v", elapsed)
	}
	if pp.abort.count() == 0 {
		t.Fatal("abort hook not fired")
	}
}

// TestBlockedRejectionWriteReleasedByAbort covers the audit's abandoned
// writer: a callback rejection that cannot be delivered must not wedge
// the read loop, and the stuck writer must be released by abort rather
// than holding the write lock forever.
func TestBlockedRejectionWriteReleasedByAbort(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(ACPCodec{})
	go func() {
		// Consume the client's request, then stop reading so the
		// rejection write blocks on the full pipe.
		testutilReadLine(pp.serverR)
		io.WriteString(pp.serverW, `{"jsonrpc":"2.0","id":42,"method":"session/prompt","params":{"prompt":"run this"}}`+"\n")
		// Deliberately never read again.
	}()
	start := time.Now()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrServerRequest) {
		t.Fatalf("err = %v, want ErrServerRequest", err)
	}
	// rejectionWriteBudget plus bounded teardown must dominate.
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("call took %v; blocked rejection wedged the read loop", elapsed)
	}
	if pp.abort.count() == 0 {
		t.Fatal("abort hook not fired; abandoned writer keeps the write lock")
	}
	// The abandoned writer must have been released: any later write
	// terminates quickly instead of queueing behind a stuck lock holder.
	done := make(chan error, 1)
	go func() { done <- c.Notify(context.Background(), "x", nil) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("write lock never released after abort")
	}
}

// TestCloseUnblocksTransportOnTerminalEOF pins die's abort ownership: a
// terminal connection failure must tear the transport down so owned
// readers and writers cannot stay blocked, and repeated Close stays a
// safe no-op.
func TestCloseUnblocksTransportOnTerminalEOF(t *testing.T) {
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		testutilReadLine(pp.serverR)
		pp.serverW.Close() // EOF
	}()
	if _, err := c.Call(context.Background(), "initialize", nil); !errors.Is(err, ErrEOF) {
		t.Fatalf("err = %v, want ErrEOF", err)
	}
	if pp.abort.count() == 0 {
		t.Fatal("abort hook not fired on terminal failure")
	}
	c.Close()
	c.Close() // repeated closure must stay safe
}

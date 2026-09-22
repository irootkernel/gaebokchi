package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"gaebokchi/internal/testutil"
)

// These tests drive the real process pipes through the scripted fake
// child, verifying the transport against actual subprocess I/O.

func TestFakeChildFragmentedReplyOverPipes(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: "{\"id\":1,\"res"},
		{SleepMs: 60},
		{Emit: "ult\":{\"ok\":true}}\n"},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), CodexCodec{}, func() { _ = child.Close() })
	resp, err := c.Call(context.Background(), "initialize", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"ok":true}` {
		t.Fatalf("result = %s", resp.Result)
	}
	child.CloseStdin()
	<-child.Done()
	child.Close()
}

func TestFakeChildStderrFloodDoesNotBreakProtocol(t *testing.T) {
	steps := []testutil.Step{
		{EmitStderr: "warning: flood line\n"},
		{Wait: 1},
		{Reply: []testutil.ReplySpec{{Slot: 0, JSONRPC: true, Result: map[string]any{"v": 1}}}},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), ACPCodec{}, func() { _ = child.Close() })
	resp, err := c.Call(context.Background(), "initialize", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"v":1}` {
		t.Fatalf("result = %s", resp.Result)
	}
	if len(child.StderrKept()) == 0 {
		t.Fatal("stderr retention empty despite flood")
	}
	child.Close()
}

func TestFakeChildBannerIsProtocolError(t *testing.T) {
	steps := []testutil.Step{
		{Emit: "Grok CLI v1.0 ready.\n"},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), ACPCodec{}, func() { _ = child.Close() })
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage for banner", err)
	}
	child.Close()
}

func TestFakeChildOversizeFrameKillsConnection(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{Oversize: 5<<20 + 128},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), CodexCodec{}, func() { _ = child.Close() })
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage wrapping ErrFrameTooLarge", err)
	}
	child.Close()
}

func TestFakeChildHungServerTimesOut(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{SleepMs: 30000},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), ACPCodec{}, func() { _ = child.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Call(ctx, "_x.ai/billing", struct{}{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("timeout unblocking took %v", elapsed)
	}
	if err := child.Close(); err != nil {
		t.Fatalf("close hung child: %v", err)
	}
}

func TestFakeChildCRLFAndDuplicateID(t *testing.T) {
	steps := []testutil.Step{
		{Wait: 1},
		{Emit: "{\"id\":1,\"result\":{\"dup\":true}}\r\n"},
		{Emit: "{\"id\":1,\"result\":{\"dup\":true}}\r\n"},
	}
	child := testutil.StartFake(t, steps)
	c := NewConn(child.Stdin(), child.Stdout(), CodexCodec{}, func() { _ = child.Close() })
	resp, err := c.Call(context.Background(), "account/read", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"dup":true}` {
		t.Fatalf("result = %s", resp.Result)
	}
	// The duplicate response id must be ignored, not delivered twice.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = c.Call(ctx, "account/rateLimits/read", nil)
	if err == nil {
		t.Fatal("second call should not be satisfied by a duplicate id")
	}
	child.Close()
}

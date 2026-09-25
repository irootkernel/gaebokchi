package rpc

import (
	"context"
	"errors"
	ginkgo "github.com/onsi/ginkgo/v2"
)

// TestErrorNullWithoutResultIsMalformed pins the audit's primary
// protocol finding: a response carrying error:null and no result is
// malformed, never a successful response with a nil result.
var _ = ginkgo.It("TestErrorNullWithoutResultIsMalformed", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"id":1,"error":null}`+"\n")
	}()
	_, err := c.Call(context.Background(), "initialize", nil)
	if !errors.Is(err, ErrMalformedMessage) {
		t.Fatalf("err = %v, want ErrMalformedMessage", err)
	}
})

// TestIncompleteErrorObjectIsMalformed requires the error member to be
// structurally complete and correctly typed before it can satisfy a
// pending call.
var _ = ginkgo.Describe("TestIncompleteErrorObjectIsMalformed", func() {
	cases := []string{
		`{"id":1,"error":{"code":-32000}}`,               // message missing
		`{"id":1,"error":{"message":"x"}}`,               // code missing
		`{"id":1,"error":{"code":"oops","message":"x"}}`, // mistyped code
		`{"id":1,"error":{"code":-32000,"message":42}}`,  // mistyped message
		`{"id":1,"error":"not an object"}`,               // wrong member type
	}
	for _, reply := range cases {
		ginkgo.It(reply, func() {
			t := testT{ginkgo.GinkgoT()}
			pp := newPipePair(t)
			c := pp.conn(CodexCodec{})
			go func() {
				defer ginkgo.GinkgoRecover()
				defer func() { _ = pp.serverW.Close() }()
				readRequest(t, pp.serverR)
				writeReply(t, pp.serverW, reply+"\n")
			}()
			_, err := c.Call(context.Background(), "initialize", nil)
			if !errors.Is(err, ErrMalformedMessage) {
				t.Fatalf("err = %v, want ErrMalformedMessage", err)
			}
		})
	}
})

// TestACPResponseRequiresJSONRPC2 pins the envelope rule: an inbound
// ACP response must carry exactly "jsonrpc":"2.0" — a missing marker or
// an incompatible one such as "1.0" is a protocol mismatch, not a
// success.
var _ = ginkgo.Describe("TestACPResponseRequiresJSONRPC2", func() {
	cases := []string{
		`{"id":1,"result":{}}`,
		`{"jsonrpc":"1.0","id":1,"result":{}}`,
		`{"jsonrpc":2.0,"id":1,"result":{}}`,
	}
	for _, reply := range cases {
		ginkgo.It(reply, func() {
			t := testT{ginkgo.GinkgoT()}
			pp := newPipePair(t)
			c := pp.conn(ACPCodec{})
			go func() {
				defer ginkgo.GinkgoRecover()
				defer func() { _ = pp.serverW.Close() }()
				readRequest(t, pp.serverR)
				writeReply(t, pp.serverW, reply+"\n")
			}()
			_, err := c.Call(context.Background(), "initialize", nil)
			if !errors.Is(err, ErrMalformedMessage) {
				t.Fatalf("err = %v, want ErrMalformedMessage", err)
			}
		})
	}
})

// TestCodexToleratesAdditiveJSONRPCMember pins dialect separation: the
// Codex dialect imposes no marker, so an additive jsonrpc member on an
// otherwise valid Codex response stays tolerated (forward
// compatibility), while the same member is mandatory on ACP.
var _ = ginkgo.It("TestCodexToleratesAdditiveJSONRPCMember", func() {
	t := testT{ginkgo.GinkgoT()}
	pp := newPipePair(t)
	c := pp.conn(CodexCodec{})
	go func() {
		defer ginkgo.GinkgoRecover()
		defer func() { _ = pp.serverW.Close() }()
		readRequest(t, pp.serverR)
		writeReply(t, pp.serverW, `{"jsonrpc":"2.0","id":1,"result":{"ok":true},"extra":"additive"}`+"\n")
		_ = pp.serverW.Close()
	}()
	resp, err := c.Call(context.Background(), "initialize", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if string(resp.Result) != `{"ok":true}` {
		t.Fatalf("result = %s", resp.Result)
	}
})

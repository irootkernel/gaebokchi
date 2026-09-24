// Package testutil provides the offline fake-child harness used by rpc
// and provider tests. Fakes are scripted subprocesses built from the test
// binary itself; no provider installation or network is involved.
package testutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gaebokchi/internal/process"
)

// Step is one scripted action of a fake provider child.
type Step struct {
	// Emit writes raw bytes to stdout verbatim (may contain newlines).
	Emit string `json:"emit,omitempty"`
	// EmitB64 writes base64-decoded bytes, for payloads that are not
	// valid UTF-8 or awkward to embed.
	EmitB64 string `json:"emit_b64,omitempty"`
	// Oversize writes n bytes of 'a' followed by a newline: a frame over
	// the 4 MiB ceiling.
	Oversize int `json:"oversize,omitempty"`
	// Wait reads n request lines into numbered slots for Reply.
	Wait int `json:"wait,omitempty"`
	// Reply responds to captured slots.
	Reply []ReplySpec `json:"reply,omitempty"`
	// SleepMs pauses the script.
	SleepMs int `json:"sleep_ms,omitempty"`
	// Capture reads n lines and reports them on stderr as
	// "FAKE-GOT:<line>" so tests can assert what the client sent.
	Capture int `json:"capture,omitempty"`
	// EmitStderr writes raw bytes to stderr.
	EmitStderr string `json:"emit_stderr,omitempty"`
	// ExitCode terminates the child immediately after this step.
	ExitCode *int `json:"exit_code,omitempty"`
}

// ReplySpec describes one response to a request captured by Wait.
type ReplySpec struct {
	// Slot selects which captured request to answer (0-based).
	Slot int `json:"slot"`
	// ID overrides the echoed id; may be a number or a string.
	ID any `json:"id,omitempty"`
	// Result is marshalled as the result member.
	Result any `json:"result,omitempty"`
	// RawResult is used verbatim when Result is unset (e.g. "null").
	RawResult string `json:"raw_result,omitempty"`
	// ErrCode/ErrMsg produce an error member when ErrCode is non-nil.
	ErrCode *int   `json:"err_code,omitempty"`
	ErrMsg  string `json:"err_msg,omitempty"`
	// JSONRPC includes "jsonrpc":"2.0" in the response (ACP dialect).
	JSONRPC bool `json:"jsonrpc,omitempty"`
	// OmitMembers emits an id-only object (neither result nor error).
	OmitMembers bool `json:"omit_members,omitempty"`
}

// StartFake launches the current test binary as a scripted fake child.
// The calling package's test binary must define a TestFakeHelperProcess
// that delegates to RunFakeScript (see fake_helper_test.go in this
// package for the exact shape to copy).
func StartFake(t *testing.T, steps []Step) *process.Child {
	t.Helper()
	script, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal fake script: %v", err)
	}
	c, err := process.Start(process.Config{
		Bin:  os.Args[0],
		Args: []string{"-test.run=TestFakeHelperProcess", "--", "fake"},
		Env:  append(os.Environ(), "GAEBOKCHI_FAKE=1", "GAEBOKCHI_FAKE_STEPS="+string(script)),
	})
	if err != nil {
		t.Fatalf("start fake child: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// RunFakeScript executes the scripted steps against the process stdio;
// it is the body of TestFakeHelperProcess.
func RunFakeScript(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	var steps []Step
	if err := json.Unmarshal([]byte(os.Getenv("GAEBOKCHI_FAKE_STEPS")), &steps); err != nil {
		return err
	}

	slots := make([]json.Number, 0, 8)
	readRequest := func() (json.Number, error) {
		for {
			line, err := readLine(stdin)
			if err != nil {
				return "", err
			}
			// Every consumed line is reported so tests can assert the
			// outgoing transcript.
			if _, err := fmt.Fprintf(stderr, "FAKE-GOT:%s\n", line); err != nil {
				return "", err
			}
			var req struct {
				ID json.Number `json:"id"`
			}
			_ = json.Unmarshal(line, &req)
			if req.ID == "" {
				// Notification without an id: not a request slot.
				continue
			}
			slots = append(slots, req.ID)
			return req.ID, nil
		}
	}

	for _, st := range steps {
		switch {
		case st.Emit != "":
			if _, err := stdout.Write([]byte(st.Emit)); err != nil {
				return err
			}
		case st.EmitB64 != "":
			raw, err := base64.StdEncoding.DecodeString(st.EmitB64)
			if err != nil {
				return err
			}
			if _, err := stdout.Write(raw); err != nil {
				return err
			}
		case st.Oversize > 0:
			chunk := strings.Repeat("a", 65536)
			written := 0
			for written < st.Oversize {
				n := st.Oversize - written
				if n > len(chunk) {
					n = len(chunk)
				}
				if _, err := stdout.Write([]byte(chunk[:n])); err != nil {
					return err
				}
				written += n
			}
			if _, err := fmt.Fprintln(stdout); err != nil {
				return err
			}
		case st.Wait > 0:
			for i := 0; i < st.Wait; i++ {
				if _, err := readRequest(); err != nil {
					return err
				}
			}
		case len(st.Reply) > 0:
			for _, r := range st.Reply {
				line, err := buildReply(r, slots)
				if err != nil {
					return err
				}
				if _, err := stdout.Write([]byte(line + "\n")); err != nil {
					return err
				}
			}
		case st.SleepMs > 0:
			time.Sleep(time.Duration(st.SleepMs) * time.Millisecond)
		case st.Capture > 0:
			for i := 0; i < st.Capture; i++ {
				line, err := readLine(stdin)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(stderr, "FAKE-GOT:%s\n", line); err != nil {
					return err
				}
			}
		case st.EmitStderr != "":
			if _, err := stderr.Write([]byte(st.EmitStderr)); err != nil {
				return err
			}
		case st.ExitCode != nil:
			os.Exit(*st.ExitCode)
		}
	}
	return nil
}

func buildReply(r ReplySpec, slots []json.Number) (string, error) {
	if r.Slot >= len(slots) {
		return "", fmt.Errorf("reply slot %d beyond captured %d", r.Slot, len(slots))
	}
	var b strings.Builder
	b.WriteByte('{')
	if r.JSONRPC {
		b.WriteString(`"jsonrpc":"2.0",`)
	}
	id := r.ID
	if id == nil {
		id = slots[r.Slot]
	}
	idJSON, err := json.Marshal(id)
	if err != nil {
		return "", err
	}
	b.WriteString(`"id":`)
	b.Write(idJSON)
	switch {
	case r.OmitMembers:
	case r.ErrCode != nil:
		b.WriteString(`,"error":{"code":`)
		b.WriteString(strconv.Itoa(*r.ErrCode))
		b.WriteString(`,"message":`)
		msg, _ := json.Marshal(r.ErrMsg)
		b.Write(msg)
		b.WriteString(`}`)
	default:
		b.WriteString(`,"result":`)
		if r.RawResult != "" {
			b.WriteString(r.RawResult)
		} else if r.Result != nil {
			res, err := json.Marshal(r.Result)
			if err != nil {
				return "", err
			}
			b.Write(res)
		} else {
			b.WriteString("null")
		}
	}
	b.WriteString("}")
	return b.String(), nil
}

// readLine reads one newline-terminated line with a 1 MiB ceiling.
func readLine(r io.Reader) ([]byte, error) {
	var buf []byte
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				return trimCR(buf), nil
			}
			buf = append(buf, one[0])
			if len(buf) > 1<<20 {
				return nil, fmt.Errorf("fake: request line too long")
			}
		}
		if err != nil {
			if len(buf) == 0 {
				return nil, err
			}
			return trimCR(buf), nil
		}
	}
}

func trimCR(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\r' {
		return b[:len(b)-1]
	}
	return b
}

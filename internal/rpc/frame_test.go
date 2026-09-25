package rpc

import (
	"bytes"
	"fmt"
	ginkgo "github.com/onsi/ginkgo/v2"
	"io"
	"strings"
)

var _ = ginkgo.It("TestFrameReaderBasics", func() {
	t := testT{ginkgo.GinkgoT()}
	var failures []string
	fail := func(format string, args ...any) { failures = append(failures, fmt.Sprintf(format, args...)) }
	input := "\n\n{\"a\":1}\r\n{\"b\":2}\n\n\n{\"c\":3}\r\n"
	fr := NewFrameReader(strings.NewReader(input))
	var got []string
	for {
		line, err := fr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		got = append(got, string(line))
	}
	want := []string{`{"a":1}`, `{"b":2}`, `{"c":3}`}
	if len(got) != len(want) {
		t.Fatalf("frames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			fail("frame %d = %q, want %q", i, got[i], want[i])
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%s", strings.Join(failures, "\n"))
	}
})

var _ = ginkgo.It("TestFrameReaderFragmentedAndCoalesced", func() {
	t := testT{ginkgo.GinkgoT()}
	pr := newPieceReader([][]byte{
		[]byte("{\"id\":1,\"resu"),
		[]byte("lt\":{}}\n{\"id\":2,\"re"),
		[]byte("sult\":null}\r\n"),
	})
	fr := NewFrameReader(pr)
	first, err := fr.Next()
	if err != nil || string(first) != `{"id":1,"result":{}}` {
		t.Fatalf("first = %q, %v", first, err)
	}
	second, err := fr.Next()
	if err != nil || string(second) != `{"id":2,"result":null}` {
		t.Fatalf("second = %q, %v", second, err)
	}
	if _, err := fr.Next(); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
})

var _ = ginkgo.It("TestFrameReaderRejectsOversize", func() {
	t := testT{ginkgo.GinkgoT()}
	big := make([]byte, MaxFrameBytes+16)
	for i := range big {
		big[i] = 'a'
	}
	big = append(big, '\n')
	fr := NewFrameReader(bytes.NewReader(big))
	if _, err := fr.Next(); err != ErrFrameTooLarge {
		t.Fatalf("err = %v, want ErrFrameTooLarge", err)
	}
})

var _ = ginkgo.It("TestFrameReaderRejectsBoundarySize", func() {
	t := testT{ginkgo.GinkgoT()}
	// Exactly MaxFrameBytes of payload plus newline is over the ceiling
	// (delimiter accounting included).
	line := bytes.Repeat([]byte("a"), MaxFrameBytes)
	line = append(line, '\n')
	fr := NewFrameReader(bytes.NewReader(line))
	if _, err := fr.Next(); err != ErrFrameTooLarge {
		t.Fatalf("err = %v, want ErrFrameTooLarge", err)
	}
	// One byte under the ceiling is accepted.
	line = bytes.Repeat([]byte("a"), MaxFrameBytes-2)
	line = append(line, '\n')
	fr = NewFrameReader(bytes.NewReader(line))
	got, err := fr.Next()
	if err != nil || len(got) != MaxFrameBytes-2 {
		t.Fatalf("boundary frame rejected: len=%d err=%v", len(got), err)
	}
})

var _ = ginkgo.It("TestFrameReaderRejectsInvalidUTF8", func() {
	t := testT{ginkgo.GinkgoT()}
	bad := []byte("{\"x\":\"\xff\xfe\"}\n")
	fr := NewFrameReader(bytes.NewReader(bad))
	if _, err := fr.Next(); err != ErrInvalidUTF8 {
		t.Fatalf("err = %v, want ErrInvalidUTF8", err)
	}
})

var _ = ginkgo.It("TestFrameReaderRejectsMissingFinalNewline", func() {
	t := testT{ginkgo.GinkgoT()}
	fr := NewFrameReader(strings.NewReader("{\"id\":1}"))
	if _, err := fr.Next(); err != ErrTruncated {
		t.Fatalf("err = %v, want ErrTruncated", err)
	}
})

// pieceReader serves reads from fixed chunks to simulate fragmentation.
type pieceReader struct {
	chunks [][]byte
	idx    int
}

func newPieceReader(chunks [][]byte) *pieceReader { return &pieceReader{chunks: chunks} }

func (p *pieceReader) Read(b []byte) (int, error) {
	if p.idx >= len(p.chunks) {
		return 0, io.EOF
	}
	n := copy(b, p.chunks[p.idx])
	if n == len(p.chunks[p.idx]) {
		p.idx++
	} else {
		p.chunks[p.idx] = p.chunks[p.idx][n:]
	}
	return n, nil
}

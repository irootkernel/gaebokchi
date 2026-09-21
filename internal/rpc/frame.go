// Package rpc provides the shared stdio JSON transport used by provider
// adapters: bounded newline framing, request-ID correlation, dialect
// codecs, and terminal transport errors. It deliberately knows nothing
// about quota or billing semantics.
package rpc

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// MaxFrameBytes is the 4 MiB ceiling for one frame, including the
// delimiter.
const MaxFrameBytes = 4 << 20

// Framing and decoding errors. They are terminal for a connection.
var (
	ErrFrameTooLarge = errors.New("rpc: frame exceeds 4 MiB ceiling")
	ErrInvalidUTF8   = errors.New("rpc: frame is not valid UTF-8")
	ErrTruncated     = errors.New("rpc: final frame missing newline")
	ErrEmptyFrame    = errors.New("rpc: empty frame")
)

// FrameReader reads newline-delimited frames. It accepts LF and CRLF,
// partial reads, and multiple messages arriving in one read; empty
// transport lines are skipped; oversized or non-UTF-8 frames are
// rejected.
type FrameReader struct {
	br    *bufio.Reader
	buf   []byte
}

// NewFrameReader wraps r.
func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{br: bufio.NewReaderSize(r, 64*1024)}
}

// Next returns the next non-empty frame without its delimiter. It returns
// io.EOF at clean stream end and a terminal framing error otherwise.
func (fr *FrameReader) Next() ([]byte, error) {
	for {
		line, err := fr.readLine()
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			continue // only empty transport lines may be skipped
		}
		if !utf8.Valid(line) {
			return nil, ErrInvalidUTF8
		}
		return line, nil
	}
}

func (fr *FrameReader) readLine() ([]byte, error) {
	fr.buf = fr.buf[:0]
	for {
		chunk, err := fr.br.ReadSlice('\n')
		fr.buf = append(fr.buf, chunk...)
		// The ceiling includes delimiter accounting.
		if len(fr.buf) > MaxFrameBytes {
			return nil, ErrFrameTooLarge
		}
		if err == nil {
			line := bytes.TrimSuffix(fr.buf, []byte("\n"))
			line = bytes.TrimSuffix(line, []byte("\r"))
			return line, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(fr.buf) == 0 {
				return nil, io.EOF
			}
			// Data without a final newline is malformed framing.
			if strings.TrimSpace(string(fr.buf)) == "" {
				return nil, io.EOF
			}
			return nil, ErrTruncated
		}
		return nil, err
	}
}

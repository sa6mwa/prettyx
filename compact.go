package prettyx

import (
	"bytes"
	"errors"
	"io"

	"pkt.systems/jpact"
)

// CompactTo streams compacted JSON to the provided writer. It supports multiple
// JSON documents in the input stream, emitting one compacted document per line.
// When opts.Unwrap is true, JSON-looking strings are decoded recursively before
// compaction.
func CompactTo(w io.Writer, r io.Reader, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions
	}
	if opts.Unwrap {
		depth := MaxNestedJSONDepth
		if depth <= 0 {
			depth = 1
		}
		return compactWithUnwrap(w, r, opts, depth)
	}
	return compactRaw(w, r)
}

// CompactToBuffer compacts JSON into memory. It preserves the one-document-per-line
// behavior of CompactTo.
func CompactToBuffer(r io.Reader, opts *Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := CompactTo(&buf, r, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var newlineBytes = []byte{'\n'}

func writeNewline(w io.Writer) error {
	if bw, ok := w.(io.ByteWriter); ok {
		return bw.WriteByte('\n')
	}
	_, err := w.Write(newlineBytes)
	return err
}

func compactRaw(w io.Writer, r io.Reader) error {
	vr := acquireValueReader(r)
	defer releaseValueReader(vr)

	for {
		if err := vr.Start(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := jpact.CompactWriter(w, vr, 0); err != nil {
			return err
		}
		if err := writeNewline(w); err != nil {
			return err
		}
		vr.Reset()
	}
}

func compactWithUnwrap(w io.Writer, r io.Reader, _ *Options, _ int) error {
	vr := acquireValueReader(r)
	defer releaseValueReader(vr)
	depth := MaxNestedJSONDepth
	if depth <= 0 {
		depth = 1
	}

	for {
		if err := vr.Start(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		ur := acquireUnwrapReader(vr, depth)
		if err := jpact.CompactWriter(w, ur, 0); err != nil {
			releaseUnwrapReader(ur)
			return err
		}
		releaseUnwrapReader(ur)

		if err := writeNewline(w); err != nil {
			return err
		}
		vr.Reset()
	}
}

type valueReader struct {
	scanner scanner

	started bool
	done    bool
	mode    valueMode
	depth   int
	inStr   bool
	escape  bool
	pending byte
	hasPend bool
}

type valueMode int

const (
	modeScalar valueMode = iota
	modeString
	modeStruct
)

func (v *valueReader) Reset() {
	v.started = false
	v.done = false
	v.mode = modeScalar
	v.depth = 0
	v.inStr = false
	v.escape = false
	v.hasPend = false
	v.pending = 0
}

func (v *valueReader) Start() error {
	if v.started {
		return nil
	}
	b, err := v.scanner.readNonSpace()
	if err != nil {
		return err
	}
	v.started = true
	v.pending = b
	v.hasPend = true
	switch b {
	case '{', '[':
		v.mode = modeStruct
		v.depth = 1
	case '"':
		v.mode = modeString
		v.inStr = true
	default:
		v.mode = modeScalar
	}
	return nil
}

func (v *valueReader) Read(p []byte) (int, error) {
	if v.done {
		return 0, io.EOF
	}
	if !v.started {
		if err := v.Start(); err != nil {
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			return 0, err
		}
	}
	if len(p) == 0 {
		return 0, nil
	}

	n := 0
	for n < len(p) {
		b, err := v.nextByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}
			return n, err
		}
		p[n] = b
		n++
	}
	return n, nil
}

func (v *valueReader) nextByte() (byte, error) {
	if v.done {
		return 0, io.EOF
	}
	if v.hasPend {
		v.hasPend = false
		return v.pending, nil
	}

	switch v.mode {
	case modeString:
		b, err := v.scanner.readByte()
		if err != nil {
			return 0, err
		}
		if v.escape {
			v.escape = false
			return b, nil
		}
		if b == '\\' {
			v.escape = true
			return b, nil
		}
		if b == '"' {
			v.done = true
			return b, nil
		}
		return b, nil
	case modeStruct:
		b, err := v.scanner.readByte()
		if err != nil {
			return 0, err
		}
		if v.inStr {
			if v.escape {
				v.escape = false
				return b, nil
			}
			if b == '\\' {
				v.escape = true
				return b, nil
			}
			if b == '"' {
				v.inStr = false
				return b, nil
			}
			return b, nil
		}
		switch b {
		case '"':
			v.inStr = true
		case '{', '[':
			v.depth++
		case '}', ']':
			v.depth--
			if v.depth == 0 {
				v.done = true
			}
		}
		return b, nil
	default:
		b, err := v.scanner.peekByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				v.done = true
				return 0, io.EOF
			}
			return 0, err
		}
		if isTerminator(b) {
			v.done = true
			return 0, io.EOF
		}
		b, _ = v.scanner.readByte()
		return b, nil
	}
}

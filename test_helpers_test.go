package prettyx

import (
	"bytes"
	"errors"
	"io"
)

type noStringWriter struct {
	buf bytes.Buffer
}

func (w *noStringWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *noStringWriter) String() string {
	return w.buf.String()
}

type stringWriter struct {
	buf bytes.Buffer
}

func (w *stringWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *stringWriter) WriteString(s string) (int, error) {
	return w.buf.WriteString(s)
}

func (w *stringWriter) String() string {
	return w.buf.String()
}

type byteWriter struct {
	buf bytes.Buffer
}

func (w *byteWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *byteWriter) WriteByte(b byte) error {
	return w.buf.WriteByte(b)
}

func (w *byteWriter) String() string {
	return w.buf.String()
}

type fdWriter struct{}

func (fdWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (fdWriter) Fd() uintptr {
	return 0
}

type zeroReader struct {
	called bool
}

func (r *zeroReader) Read(_ []byte) (int, error) {
	if r.called {
		return 0, io.EOF
	}
	r.called = true
	return 0, nil
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read err")
}

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write err")
}

type errStringWriter struct{}

func (errStringWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write err")
}

func (errStringWriter) WriteString(_ string) (int, error) {
	return 0, errors.New("write string err")
}

type errByteWriter struct{}

func (errByteWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write err")
}

func (errByteWriter) WriteByte(_ byte) error {
	return errors.New("write byte err")
}

type newlineFailWriter struct {
	buf bytes.Buffer
}

func (w *newlineFailWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *newlineFailWriter) WriteByte(_ byte) error {
	return errors.New("newline err")
}

type failAfterStringWriter struct {
	count int
	fail  int
	buf   bytes.Buffer
}

func (w *failAfterStringWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *failAfterStringWriter) WriteString(s string) (int, error) {
	w.count++
	if w.count > w.fail {
		return 0, errors.New("write string err")
	}
	return w.buf.WriteString(s)
}

type failAfterByteStringWriter struct {
	count int
	fail  int
	buf   bytes.Buffer
}

func (w *failAfterByteStringWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *failAfterByteStringWriter) WriteString(s string) (int, error) {
	w.count++
	if w.count > w.fail {
		return 0, errors.New("write string err")
	}
	return w.buf.WriteString(s)
}

func (w *failAfterByteStringWriter) WriteByte(b byte) error {
	return w.buf.WriteByte(b)
}

type byteFailStringWriter struct {
	buf bytes.Buffer
}

func (w *byteFailStringWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *byteFailStringWriter) WriteString(s string) (int, error) {
	return w.buf.WriteString(s)
}

func (w *byteFailStringWriter) WriteByte(_ byte) error {
	return errors.New("write byte err")
}

type failSecondNewlineWriter struct {
	newlines int
	buf      bytes.Buffer
}

func (w *failSecondNewlineWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *failSecondNewlineWriter) WriteString(s string) (int, error) {
	return w.buf.WriteString(s)
}

func (w *failSecondNewlineWriter) WriteByte(b byte) error {
	if b == '\n' {
		w.newlines++
		if w.newlines > 1 {
			return errors.New("newline err")
		}
	}
	return w.buf.WriteByte(b)
}

type errAfterReader struct {
	data []byte
	err  error
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	if r.err == nil {
		r.err = errors.New("read err")
	}
	return 0, r.err
}

type failAfterByteWriter struct {
	count int
	fail  int
	buf   bytes.Buffer
}

func (w *failAfterByteWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *failAfterByteWriter) WriteByte(b byte) error {
	w.count++
	if w.count > w.fail {
		return errors.New("write byte err")
	}
	return w.buf.WriteByte(b)
}

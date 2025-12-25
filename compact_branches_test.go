package prettyx

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCompactToBuffer_ErrorPath(t *testing.T) {
	if _, err := CompactToBuffer(strings.NewReader("{\"a\":"), DefaultOptions); err == nil {
		t.Fatalf("expected CompactToBuffer error")
	}
}

func TestCompact_EmptyInput(t *testing.T) {
	var buf bytes.Buffer
	if err := CompactTo(&buf, strings.NewReader("   "), DefaultOptions); err != nil {
		t.Fatalf("expected empty input to succeed: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output for empty input, got %q", buf.String())
	}

	if err := CompactTo(&buf, strings.NewReader("{\"a\":1}"), nil); err != nil {
		t.Fatalf("expected CompactTo with nil opts to succeed: %v", err)
	}

	if err := CompactTo(io.Discard, errReader{}, DefaultOptions); err == nil {
		t.Fatalf("expected error reader to fail")
	}

	nw := &newlineFailWriter{}
	if err := CompactTo(nw, strings.NewReader("{\"a\":1}"), DefaultOptions); err == nil {
		t.Fatalf("expected newline write failure")
	}
}

func TestCompact_Unwrap_InvalidInput(t *testing.T) {
	opts := *DefaultOptions
	opts.Unwrap = true
	if err := CompactTo(io.Discard, strings.NewReader("{\"a\":1,}"), &opts); err == nil {
		t.Fatalf("expected compact unwrap error")
	}

	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 0
	if err := CompactTo(io.Discard, strings.NewReader("{\"a\":\"{\\\"b\\\":1}\"}"), &opts); err != nil {
		t.Fatalf("expected compact unwrap with depth 0 to succeed: %v", err)
	}

	if err := CompactTo(io.Discard, strings.NewReader(""), &opts); err != nil {
		t.Fatalf("expected unwrap empty input to succeed: %v", err)
	}

	if err := CompactTo(io.Discard, errReader{}, &opts); err == nil {
		t.Fatalf("expected unwrap start error")
	}

	nw := &newlineFailWriter{}
	if err := CompactTo(nw, strings.NewReader("{\"a\":\"{\\\"b\\\":1}\"}"), &opts); err == nil {
		t.Fatalf("expected unwrap newline write failure")
	}
}

func TestCompact_UnwrapWriterError(t *testing.T) {
	opts := *DefaultOptions
	opts.Unwrap = true
	if err := CompactTo(errWriter{}, strings.NewReader("{\"a\":1}"), &opts); err == nil {
		t.Fatalf("expected unwrap writer error")
	}
}

func TestValueReaderBranches(t *testing.T) {
	var s scanner
	s.Reset(strings.NewReader(""))
	v := valueReader{scanner: s}
	if err := v.Start(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF start on empty, got %v", err)
	}

	s.Reset(errReader{})
	v = valueReader{scanner: s}
	if err := v.Start(); err == nil {
		t.Fatalf("expected error reader in Start")
	}

	s.Reset(strings.NewReader("\"a\\\"b\""))
	v = valueReader{scanner: s}
	if err := v.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := v.Start(); err != nil {
		t.Fatalf("Start idempotent failed: %v", err)
	}
	if n, err := v.Read(nil); err != nil || n != 0 {
		t.Fatalf("expected zero read, got n=%d err=%v", n, err)
	}
	out, err := io.ReadAll(&v)
	if err != nil {
		t.Fatalf("read string failed: %v", err)
	}
	if string(out) != "\"a\\\"b\"" {
		t.Fatalf("unexpected string output: %q", out)
	}

	s.Reset(strings.NewReader("{\"a\":\"b\\\"c\"}"))
	v = valueReader{scanner: s}
	if err := v.Start(); err != nil {
		t.Fatalf("Start struct failed: %v", err)
	}
	out, err = io.ReadAll(&v)
	if err != nil {
		t.Fatalf("read struct failed: %v", err)
	}
	if string(out) != "{\"a\":\"b\\\"c\"}" {
		t.Fatalf("unexpected struct output: %q", out)
	}

	s.Reset(strings.NewReader("1 "))
	v = valueReader{scanner: s}
	if err := v.Start(); err != nil {
		t.Fatalf("Start scalar failed: %v", err)
	}
	out, err = io.ReadAll(&v)
	if err != nil {
		t.Fatalf("read scalar failed: %v", err)
	}
	if string(out) != "1" {
		t.Fatalf("unexpected scalar output: %q", out)
	}

	s.Reset(strings.NewReader(""))
	v = valueReader{scanner: s}
	if n, err := v.Read(make([]byte, 1)); !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("expected EOF on empty Read, got n=%d err=%v", n, err)
	}

	s.Reset(errReader{})
	v = valueReader{scanner: s}
	if _, err := v.Read(make([]byte, 1)); err == nil {
		t.Fatalf("expected Read error for errReader")
	}

	s.Reset(strings.NewReader("1"))
	v = valueReader{scanner: s}
	buf := make([]byte, 4)
	n, err := v.Read(buf)
	if err != nil || n != 1 {
		t.Fatalf("expected partial read, got n=%d err=%v", n, err)
	}
	if _, err := v.Read(buf); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after partial read, got %v", err)
	}

	s.Reset(strings.NewReader("1"))
	v = valueReader{scanner: s}
	if n, err := v.Read(make([]byte, 1)); err != nil || n != 1 {
		t.Fatalf("expected full buffer read, got n=%d err=%v", n, err)
	}

	v.done = true
	if n, err := v.Read(make([]byte, 1)); !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("expected EOF on done reader, got n=%d err=%v", n, err)
	}

	s.Reset(errReader{})
	v = valueReader{scanner: s, started: true, mode: modeString}
	if _, err := v.Read(make([]byte, 1)); err == nil {
		t.Fatalf("expected Read error from nextByte")
	}
}

func TestValueReaderNextByteBranches(t *testing.T) {
	var s scanner

	s.Reset(strings.NewReader("\\\"\""))
	v := valueReader{scanner: s, started: true, mode: modeString}
	if b, err := v.nextByte(); err != nil || b != '\\' || !v.escape {
		t.Fatalf("expected escape start, b=%q err=%v escape=%v", b, err, v.escape)
	}
	if b, err := v.nextByte(); err != nil || b != '"' || v.escape {
		t.Fatalf("expected escaped quote, b=%q err=%v escape=%v", b, err, v.escape)
	}
	if b, err := v.nextByte(); err != nil || b != '"' || !v.done {
		t.Fatalf("expected closing quote, b=%q err=%v done=%v", b, err, v.done)
	}

	s.Reset(errReader{})
	v = valueReader{scanner: s, started: true, mode: modeString}
	if _, err := v.nextByte(); err == nil {
		t.Fatalf("expected string readByte error")
	}

	s.Reset(strings.NewReader("}"))
	v = valueReader{scanner: s, started: true, mode: modeStruct, depth: 1}
	if b, err := v.nextByte(); err != nil || b != '}' || !v.done {
		t.Fatalf("expected struct close, b=%q err=%v done=%v", b, err, v.done)
	}

	s.Reset(strings.NewReader("["))
	v = valueReader{scanner: s, started: true, mode: modeStruct, depth: 1}
	if b, err := v.nextByte(); err != nil || b != '[' || v.depth != 2 {
		t.Fatalf("expected depth increment, b=%q err=%v depth=%d", b, err, v.depth)
	}

	s.Reset(strings.NewReader("\""))
	v = valueReader{scanner: s, started: true, mode: modeStruct, depth: 1}
	if b, err := v.nextByte(); err != nil || b != '"' || !v.inStr {
		t.Fatalf("expected enter string, b=%q err=%v inStr=%v", b, err, v.inStr)
	}

	s.Reset(strings.NewReader("\\\"\""))
	v = valueReader{scanner: s, started: true, mode: modeStruct, inStr: true}
	if b, err := v.nextByte(); err != nil || b != '\\' || !v.escape {
		t.Fatalf("expected struct escape start, b=%q err=%v escape=%v", b, err, v.escape)
	}
	if b, err := v.nextByte(); err != nil || b != '"' || v.escape {
		t.Fatalf("expected struct escaped quote, b=%q err=%v escape=%v", b, err, v.escape)
	}
	if b, err := v.nextByte(); err != nil || b != '"' || v.inStr {
		t.Fatalf("expected struct string end, b=%q err=%v inStr=%v", b, err, v.inStr)
	}

	s.Reset(strings.NewReader(""))
	v = valueReader{scanner: s, started: true, mode: modeScalar}
	if _, err := v.nextByte(); !errors.Is(err, io.EOF) || !v.done {
		t.Fatalf("expected scalar EOF, err=%v done=%v", err, v.done)
	}

	s.Reset(strings.NewReader("1"))
	v = valueReader{scanner: s, started: true, mode: modeScalar}
	if b, err := v.nextByte(); err != nil || b != '1' {
		t.Fatalf("expected scalar byte, b=%q err=%v", b, err)
	}
	if _, err := v.nextByte(); !errors.Is(err, io.EOF) || !v.done {
		t.Fatalf("expected scalar terminator EOF, err=%v done=%v", err, v.done)
	}

	s.Reset(errReader{})
	v = valueReader{scanner: s, started: true, mode: modeScalar}
	if _, err := v.nextByte(); err == nil {
		t.Fatalf("expected scalar peek error")
	}
}

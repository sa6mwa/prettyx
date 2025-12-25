package prettyx

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func readAllUnwrap(input string, depth int) (string, error) {
	ur := acquireUnwrapReader(strings.NewReader(input), depth)
	defer releaseUnwrapReader(ur)
	var buf bytes.Buffer
	_, err := io.Copy(&buf, ur)
	return buf.String(), err
}

func TestUnwrapReader_UnwrapsAndPreservesKeys(t *testing.T) {
	out, err := readAllUnwrap("{\"{\\\"a\\\":1}\":\"{\\\"b\\\":2}\"}", 2)
	if err != nil {
		t.Fatalf("unwrap failed: %v", err)
	}
	if out != "{\"{\\\"a\\\":1}\":{\"b\":2}}" {
		t.Fatalf("unexpected unwrap output: %q", out)
	}
}

func TestUnwrapReader_DepthZeroLeavesString(t *testing.T) {
	out, err := readAllUnwrap("{\"payload\":\"{\\\"a\\\":1}\"}", 0)
	if err != nil {
		t.Fatalf("unwrap failed: %v", err)
	}
	if out != "{\"payload\":\"{\\\"a\\\":1}\"}" {
		t.Fatalf("unexpected depth0 output: %q", out)
	}
}

func TestUnwrapReader_InvalidEmbeddedJSON(t *testing.T) {
	out, err := readAllUnwrap("{\"payload\":\"{bad}\"}", 2)
	if err != nil {
		t.Fatalf("unwrap failed: %v", err)
	}
	if out != "{\"payload\":\"{bad}\"}" {
		t.Fatalf("unexpected invalid output: %q", out)
	}
}

func TestUnwrapReader_EmptyContainers(t *testing.T) {
	out, err := readAllUnwrap("[]", 1)
	if err != nil || out != "[]" {
		t.Fatalf("expected empty array, got %q err %v", out, err)
	}
	out, err = readAllUnwrap("{}", 1)
	if err != nil || out != "{}" {
		t.Fatalf("expected empty object, got %q err %v", out, err)
	}
}

func TestUnwrapReader_ErrorBranches(t *testing.T) {
	cases := []string{
		"{\"a\":1,2}",
		"{\"a\" 1}",
		"{\"a\":1 \"b\":2}",
		"[1,]",
		"[1 2]",
		"{\"a\":1,}",
		"tru",
		"01",
		"x",
		"\"\\q\"",
		"\"\\u12G4\"",
		"\"\\uD800x\"",
		"\"bad\nstring\"",
		"{\"\\q\":1}",
	}
	for _, input := range cases {
		if _, err := readAllUnwrap(input, 2); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}

	var src unwrapSource
	src.scanner.Reset(strings.NewReader("2"))
	src.topValueSeen = true
	if _, err := src.nextByte(&unwrapReader{}); err == nil {
		t.Fatalf("expected multiple top-level values error")
	}

	src.scanner.Reset(strings.NewReader(""))
	src.topValueSeen = true
	if _, err := src.nextByte(&unwrapReader{}); err != io.EOF {
		t.Fatalf("expected EOF after top-level value, got %v", err)
	}
}

func TestUnwrapReader_ReadZeroLen(t *testing.T) {
	ur := acquireUnwrapReader(strings.NewReader("1"), 1)
	defer releaseUnwrapReader(ur)
	n, err := ur.Read(nil)
	if err != nil {
		t.Fatalf("expected nil error for zero read, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected zero bytes read")
	}
}

func TestUnwrapReader_PushSourceBranches(t *testing.T) {
	var u unwrapReader
	u.sources = make([]unwrapSource, 0, 1)
	u.used = 0
	u.pushSource([]byte("1"), 1)
	u.pushSource([]byte("2"), 1) // triggers append branch
	u.clear()
}

func TestUnwrapSource_InternalBranches(t *testing.T) {
	var s unwrapSource
	s.scanner.Reset(strings.NewReader("ruX"))
	if _, err := s.readLiteral('t'); err == nil {
		t.Fatalf("expected invalid literal error")
	}
	if _, err := s.readLiteral('x'); err == nil {
		t.Fatalf("expected invalid literal start error")
	}

	s.scanner.Reset(strings.NewReader("1e"))
	if _, err := s.readNumber('1'); err == nil {
		t.Fatalf("expected invalid number error")
	}
	s.scanner.Reset(strings.NewReader(""))
	if _, err := s.readNumber('x'); err == nil {
		t.Fatalf("expected invalid number start error")
	}
	s.scanner.Reset(errReader{})
	if _, err := s.readNumber('1'); err == nil {
		t.Fatalf("expected number peek error")
	}

	s.scanner.Reset(strings.NewReader("\\uZZZZ"))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected invalid unicode escape in key")
	}
	s.scanner.Reset(strings.NewReader("\\u12"))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected short unicode escape in key")
	}
	s.scanner.Reset(strings.NewReader("\\q\""))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected invalid escape in key")
	}
	s.scanner.Reset(strings.NewReader("abc"))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected unterminated key error")
	}
	s.scanner.Reset(strings.NewReader("\\"))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected unterminated escape error")
	}

	s.scanner.Reset(strings.NewReader("\\u0041\""))
	if token, err := s.readRawStringToken(); err != nil || string(token) != "\"\\u0041\"" {
		t.Fatalf("expected raw token, got %q err %v", token, err)
	}
	s.scanner.Reset(strings.NewReader("\x01\""))
	if _, err := s.readRawStringToken(); err == nil {
		t.Fatalf("expected invalid control character in key")
	}

	s.scanner.Reset(strings.NewReader("rue"))
	if token, err := s.readLiteral('t'); err != nil || string(token) != "true" {
		t.Fatalf("expected true literal, got %q err %v", token, err)
	}
	s.scanner.Reset(strings.NewReader("alse"))
	if token, err := s.readLiteral('f'); err != nil || string(token) != "false" {
		t.Fatalf("expected false literal, got %q err %v", token, err)
	}
	s.scanner.Reset(strings.NewReader("ull"))
	if token, err := s.readLiteral('n'); err != nil || string(token) != "null" {
		t.Fatalf("expected null literal, got %q err %v", token, err)
	}

	s.scanner.Reset(strings.NewReader("e+2"))
	if token, err := s.readNumber('1'); err != nil || string(token) != "1e+2" {
		t.Fatalf("expected number, got %q err %v", token, err)
	}

	s.scanner.Reset(strings.NewReader("\\\\\""))
	if val, err := s.readStringValue(); err != nil || string(val) != "\\" {
		t.Fatalf("expected escaped string, got %q err %v", val, err)
	}
	s.scanner.Reset(strings.NewReader("\\b\\f\\n\\r\\t\""))
	if val, err := s.readStringValue(); err != nil || string(val) != "\b\f\n\r\t" {
		t.Fatalf("expected escaped controls, got %q err %v", val, err)
	}
	s.scanner.Reset(strings.NewReader("\\u0041\""))
	if val, err := s.readStringValue(); err != nil || string(val) != "A" {
		t.Fatalf("expected unicode escape, got %q err %v", val, err)
	}
	s.scanner.Reset(strings.NewReader("\\q\""))
	if _, err := s.readStringValue(); err == nil {
		t.Fatalf("expected invalid escape error")
	}
	s.scanner.Reset(strings.NewReader("abc"))
	if _, err := s.readStringValue(); err == nil {
		t.Fatalf("expected unterminated string error")
	}
	s.scanner.Reset(strings.NewReader("\\"))
	if _, err := s.readStringValue(); err == nil {
		t.Fatalf("expected unterminated escape error")
	}
	s.scanner.Reset(strings.NewReader("\\u12G4\""))
	if _, err := s.readStringValue(); err == nil {
		t.Fatalf("expected invalid unicode error")
	}

	s.decodedBuf = make([]byte, maxScratchCap+1)
	s.scratch = make([]byte, maxScratchCap+1)
	s.rawBuf = make([]byte, maxScratchCap+1)
	s.clear()
}

func TestUnwrapReader_ValidatorAndRelease(t *testing.T) {
	u := acquireUnwrapReader(strings.NewReader("\"{\\\"a\\\":1}\""), 2)
	if !u.validateJSONBytes([]byte("{\"a\":1}")) {
		t.Fatalf("expected valid json")
	}
	if u.validateJSONBytes(nil) {
		t.Fatalf("expected invalid empty json")
	}
	u.validator.scratch = make([]byte, maxScratchCap+1)
	u.validator.decodedBuf = make([]byte, maxScratchCap+1)
	u.clear()
	releaseUnwrapReader(u)
	releaseUnwrapReader(nil)
}

func TestUnwrapSource_PushBranches(t *testing.T) {
	var s unwrapSource
	s.stack = make([]containerState, 1, 1)
	s.pushObject()
	s.stack = make([]containerState, 1, 1)
	s.pushArray()
}

func TestUnwrapReader_ReadEOF(t *testing.T) {
	ur := acquireUnwrapReader(strings.NewReader(""), 1)
	defer releaseUnwrapReader(ur)
	buf := make([]byte, 1)
	_, err := ur.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected EOF on empty read, got %v", err)
	}
}

func TestUnwrapReader_ReadError(t *testing.T) {
	ur := acquireUnwrapReader(errReader{}, 1)
	defer releaseUnwrapReader(ur)
	_, err := ur.Read(make([]byte, 1))
	if err == nil {
		t.Fatalf("expected reader error")
	}
}

func TestUnwrapSource_ReadUnicodeEscapeBranches(t *testing.T) {
	var s unwrapSource
	s.scanner.Reset(strings.NewReader("0041"))
	if r, err := s.readUnicodeEscape(); err != nil || r != 'A' {
		t.Fatalf("expected A, got %v err %v", r, err)
	}
	s.scanner.Reset(strings.NewReader("D834\\uDD1E"))
	if _, err := s.readUnicodeEscape(); err != nil {
		t.Fatalf("expected valid surrogate, got %v", err)
	}
	s.scanner.Reset(strings.NewReader("D800x"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid surrogate error")
	}
	s.scanner.Reset(strings.NewReader("D800\\u0001"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid low surrogate error")
	}
	s.scanner.Reset(strings.NewReader("ZZZZ"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid hex error")
	}
	s.scanner.Reset(strings.NewReader("D800"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected short surrogate error")
	}
	s.scanner.Reset(strings.NewReader("D800\\"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected short surrogate escape error")
	}
	s.scanner.Reset(strings.NewReader("D800\\u12"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected short low surrogate error")
	}
	s.scanner.Reset(strings.NewReader("D800\\x0000"))
	if _, err := s.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid surrogate prefix error")
	}

	s.scanner.Reset(strings.NewReader("12"))
	if _, err := s.readHex4(); err == nil {
		t.Fatalf("expected short hex error")
	}
}

func TestUnwrapReader_ReadPartial(t *testing.T) {
	ur := acquireUnwrapReader(strings.NewReader("1"), 1)
	defer releaseUnwrapReader(ur)
	buf := make([]byte, 8)
	n, err := ur.Read(buf)
	if err != nil || n != 1 {
		t.Fatalf("expected partial read, got n=%d err=%v", n, err)
	}
}

func TestUnwrapReader_ReadExact(t *testing.T) {
	ur := acquireUnwrapReader(strings.NewReader("1"), 1)
	defer releaseUnwrapReader(ur)
	buf := make([]byte, 1)
	n, err := ur.Read(buf)
	if err != nil || n != 1 {
		t.Fatalf("expected exact read, got n=%d err=%v", n, err)
	}
}

func TestUnwrapReader_ClearBranches(t *testing.T) {
	ur := acquireUnwrapReader(strings.NewReader("\"{\\\"a\\\":1}\""), 1)
	_, _ = io.ReadAll(ur)
	ur.validator.scratch = make([]byte, maxScratchCap+1)
	ur.validator.decodedBuf = make([]byte, maxScratchCap+1)
	ur.clear()
	releaseUnwrapReader(ur)
}

func TestUnwrapReader_HandleValueLiteral(t *testing.T) {
	out, err := readAllUnwrap("true", 1)
	if err != nil || out != "true" {
		t.Fatalf("expected true literal, got %q err %v", out, err)
	}
}

package prettyx

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFormatterBranches(t *testing.T) {
	var buf bytes.Buffer
	f := formatter{}
	opts := &Options{Prefix: ">", Indent: "  ", Width: 2}
	f.reset(&buf, NoColorPalette(), opts, false)
	if err := f.writeANSI(""); err != nil {
		t.Fatalf("writeANSI empty failed: %v", err)
	}
	if err := f.writeANSI("x"); err != nil {
		t.Fatalf("writeANSI non-empty failed: %v", err)
	}
	if err := f.writeString(""); err != nil {
		t.Fatalf("writeString empty failed: %v", err)
	}
	if err := f.writeString("a"); err != nil {
		t.Fatalf("writeString failed: %v", err)
	}
	if err := f.writeByte('b'); err != nil {
		t.Fatalf("writeByte failed: %v", err)
	}
	if err := f.writeStyledString("", "c"); err != nil {
		t.Fatalf("writeStyledString empty style failed: %v", err)
	}
	if err := f.writeStyledString("x", "c"); err != nil {
		t.Fatalf("writeStyledString styled failed: %v", err)
	}
	if err := f.writeStyledByte("", 'd'); err != nil {
		t.Fatalf("writeStyledByte empty style failed: %v", err)
	}
	if err := f.writeStyledByte("x", 'd'); err != nil {
		t.Fatalf("writeStyledByte styled failed: %v", err)
	}
	if err := f.ensureLineStart(1); err != nil {
		t.Fatalf("ensureLineStart failed: %v", err)
	}
	if err := f.newline(1); err != nil {
		t.Fatalf("newline failed: %v", err)
	}
	f.reset(&noStringWriter{}, NoColorPalette(), opts, false)
	if err := f.writeANSI("x"); err != nil {
		t.Fatalf("writeANSI fallback failed: %v", err)
	}
	if err := f.writeString("x"); err != nil {
		t.Fatalf("writeString fallback failed: %v", err)
	}
	f.reset(&byteWriter{}, NoColorPalette(), opts, false)
	if err := f.writeByte('x'); err != nil {
		t.Fatalf("writeByte bytewriter failed: %v", err)
	}
	f.reset(&buf, NoColorPalette(), nil, false)
	if err := f.writeBytes(nil); err != nil {
		t.Fatalf("writeBytes nil failed: %v", err)
	}
	f.reset(nil, NoColorPalette(), nil, false)
	f.reset(&buf, NoColorPalette(), &Options{}, false)
	if err := f.writeIndent(0); err != nil {
		t.Fatalf("writeIndent depth 0 failed: %v", err)
	}
	if err := f.writeIndent(1); err != nil {
		t.Fatalf("writeIndent empty indent failed: %v", err)
	}
	f.reset(&buf, NoColorPalette(), &Options{Indent: " ", Prefix: ""}, false)
	if err := f.writeIndent(2); err != nil {
		t.Fatalf("writeIndent depth loop failed: %v", err)
	}
	f.reset(errStringWriter{}, NoColorPalette(), opts, false)
	if err := f.writeANSI("x"); err == nil {
		t.Fatalf("expected writeANSI error")
	}
	f.reset(errStringWriter{}, NoColorPalette(), opts, false)
	if err := f.writeString("x"); err == nil {
		t.Fatalf("expected writeString error")
	}
	f.reset(errByteWriter{}, NoColorPalette(), opts, false)
	if err := f.writeByte('x'); err == nil {
		t.Fatalf("expected writeByte error")
	}
	f.reset(errWriter{}, NoColorPalette(), opts, false)
	if err := f.writeBytes([]byte("x")); err == nil {
		t.Fatalf("expected writeBytes error")
	}
	f.reset(errStringWriter{}, NoColorPalette(), opts, false)
	if err := f.writeStyledString("x", "y"); err == nil {
		t.Fatalf("expected writeStyledString error")
	}
	f.reset(errStringWriter{}, NoColorPalette(), opts, false)
	if err := f.writeStyledString("", "y"); err == nil {
		t.Fatalf("expected writeStyledString error without style")
	}
	f.reset(&failAfterStringWriter{fail: 2}, NoColorPalette(), opts, false)
	if err := f.writeStyledString("x", "y"); err == nil {
		t.Fatalf("expected writeStyledString reset error")
	}
	f.reset(errByteWriter{}, NoColorPalette(), opts, false)
	if err := f.writeStyledByte("x", 'y'); err == nil {
		t.Fatalf("expected writeStyledByte error")
	}
	f.reset(errByteWriter{}, NoColorPalette(), opts, false)
	if err := f.writeStyledByte("", 'y'); err == nil {
		t.Fatalf("expected writeStyledByte error without style")
	}
	f.reset(&failAfterByteStringWriter{fail: 1}, NoColorPalette(), opts, false)
	if err := f.writeStyledByte("x", 'y'); err == nil {
		t.Fatalf("expected writeStyledByte reset error")
	}
	f.reset(errByteWriter{}, NoColorPalette(), opts, false)
	if err := f.newline(1); err == nil {
		t.Fatalf("expected newline error")
	}
	f.reset(errWriter{}, NoColorPalette(), opts, false)
	if err := f.writeByte('x'); err == nil {
		t.Fatalf("expected writeByte error")
	}
	f.reset(errByteWriter{}, NoColorPalette(), opts, false)
	if err := f.writeByte('x'); err == nil {
		t.Fatalf("expected writeByte bytewriter error")
	}
	f.reset(errStringWriter{}, NoColorPalette(), &Options{Prefix: ">", Indent: " "}, false)
	if err := f.writeIndent(1); err == nil {
		t.Fatalf("expected writeIndent error")
	}
	f.reset(&failAfterStringWriter{fail: 1}, NoColorPalette(), &Options{Prefix: ">", Indent: " "}, false)
	if err := f.writeIndent(2); err == nil {
		t.Fatalf("expected writeIndent indent error")
	}
}

func TestWriteSeparatorBranches(t *testing.T) {
	var buf bytes.Buffer
	p := parser{}
	p.formatter = &p.fmt
	p.fmt.reset(&buf, NoColorPalette(), &Options{Width: 0, SemiCompact: true}, false)
	if broke, err := p.writeSeparator(1); err != nil || !broke {
		t.Fatalf("expected break with width <=0, err=%v", err)
	}
	buf.Reset()
	p.fmt.reset(&buf, NoColorPalette(), &Options{Width: 3, SemiCompact: true}, false)
	p.fmt.lineLen = 3
	if broke, err := p.writeSeparator(1); err != nil || !broke {
		t.Fatalf("expected break with lineLen >= width, err=%v", err)
	}
	buf.Reset()
	p.fmt.reset(&buf, NoColorPalette(), &Options{Width: 10, SemiCompact: true}, false)
	if broke, err := p.writeSeparator(1); err != nil || broke {
		t.Fatalf("expected inline separator, err=%v", err)
	}
	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{Width: 0, SemiCompact: true}, false)
	if _, err := p.writeSeparator(1); err == nil {
		t.Fatalf("expected separator error")
	}
	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{Width: 10, SemiCompact: true}, false)
	if _, err := p.writeSeparator(1); err == nil {
		t.Fatalf("expected separator error for inline")
	}
	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{Width: 1, SemiCompact: true}, false)
	p.fmt.lineLen = 1
	if _, err := p.writeSeparator(1); err == nil {
		t.Fatalf("expected separator error for wrapped")
	}

	p.fmt.reset(&byteFailStringWriter{}, NoColorPalette(), &Options{Width: 0, SemiCompact: true}, false)
	if _, err := p.writeSeparator(1); err == nil {
		t.Fatalf("expected separator newline error")
	}

	p.fmt.reset(&byteFailStringWriter{}, NoColorPalette(), &Options{Width: 2, SemiCompact: true}, false)
	p.fmt.lineLen = 2
	if _, err := p.writeSeparator(1); err == nil {
		t.Fatalf("expected separator newline error for width")
	}
}

func TestParserSilentError(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("x"))
	p.silentErr = true
	err := p.parseValue(0)
	if !errors.Is(err, errInvalidJSON) {
		t.Fatalf("expected errInvalidJSON, got %v", err)
	}
}

func TestStreamPrettyEmptyInput(t *testing.T) {
	if err := streamPretty(io.Discard, strings.NewReader(""), DefaultOptions, NoColorPalette(), false); err != nil {
		t.Fatalf("expected empty input to succeed: %v", err)
	}
	if err := streamPretty(io.Discard, errReader{}, DefaultOptions, NoColorPalette(), false); err == nil {
		t.Fatalf("expected streamPretty error from errReader")
	}
	if err := streamPretty(errByteWriter{}, strings.NewReader("1"), DefaultOptions, NoColorPalette(), false); err == nil {
		t.Fatalf("expected streamPretty writeByte error")
	}
}

func TestStreamPrettyNewlineError(t *testing.T) {
	w := &failSecondNewlineWriter{newlines: 1}
	if err := streamPretty(w, strings.NewReader("1"), DefaultOptions, NoColorPalette(), true); err == nil {
		t.Fatalf("expected streamPretty newline error")
	}
}

func TestStreamPrettyInvalidInputs(t *testing.T) {
	cases := []string{
		"x",
		"{\"a\":1, 2}",
		"{\"a\" 1}",
		"{\"a\":1 \"b\":2}",
		"[1 2]",
		"[1,]",
		"tru",
		"01",
		"1e",
		"1.",
		"\"\\q\"",
		"\"\\u12G4\"",
		"\"bad\nstring\"",
	}
	for _, input := range cases {
		var buf bytes.Buffer
		err := streamPretty(&buf, strings.NewReader(input), DefaultOptions, NoColorPalette(), false)
		if err == nil {
			t.Fatalf("expected error for input %q", input)
		}
	}
}

func TestStreamPrettyWidthModes(t *testing.T) {
	opts := *DefaultOptions
	opts.Width = 0
	opts.SemiCompact = true
	opts.Palette = "none"
	in := strings.NewReader("[1,2]")
	var buf bytes.Buffer
	if err := streamPretty(&buf, in, &opts, NoColorPalette(), false); err != nil {
		t.Fatalf("streamPretty width 0 failed: %v", err)
	}

	opts.Width = 4
	in = strings.NewReader("[1,2,3]")
	buf.Reset()
	if err := streamPretty(&buf, in, &opts, NoColorPalette(), false); err != nil {
		t.Fatalf("streamPretty width 4 failed: %v", err)
	}
	buf.Reset()
	in = strings.NewReader("{\"a\":1}")
	if err := streamPretty(&buf, in, &opts, NoColorPalette(), true); err != nil {
		t.Fatalf("streamPretty compact failed: %v", err)
	}
}

func TestHelpersHexAndNum(t *testing.T) {
	if !isHex('a') || !isHex('F') || !isHex('9') {
		t.Fatalf("isHex expected true for hex digits")
	}
	if isHex('g') {
		t.Fatalf("isHex expected false for non-hex")
	}
	if fromHex('a') == 0 || fromHex('F') == 0 || fromHex('9') == 0 {
		t.Fatalf("fromHex expected non-zero for hex inputs")
	}
	if fromHex('g') != 0 {
		t.Fatalf("fromHex expected 0 for invalid hex")
	}
	if hexDigit(0) != '0' || hexDigit(10) != 'a' {
		t.Fatalf("hexDigit mismatch")
	}

	if st, ok := numStartState('0'); !ok || st != numZero {
		t.Fatalf("expected numZero for '0'")
	}
	if st, ok := numStartState('5'); !ok || st != numInt {
		t.Fatalf("expected numInt for '5'")
	}
	if _, ok := numStartState('x'); ok {
		t.Fatalf("expected invalid num start")
	}
	if _, ok := numStartState('-'); !ok {
		t.Fatalf("expected valid num start")
	}
	if _, ok := numNextState(numSign, '1'); !ok {
		t.Fatalf("expected numSign transition")
	}
	if _, ok := numNextState(numZero, '.'); !ok {
		t.Fatalf("expected numZero dot transition")
	}
	if _, ok := numNextState(numZero, 'e'); !ok {
		t.Fatalf("expected numZero exp transition")
	}
	if _, ok := numNextState(numZero, '1'); ok {
		t.Fatalf("expected invalid transition for leading zero")
	}
	if _, ok := numNextState(numZero, 'x'); ok {
		t.Fatalf("expected invalid transition for numZero other")
	}
	if _, ok := numNextState(numInt, '.'); !ok {
		t.Fatalf("expected valid dot transition")
	}
	if _, ok := numNextState(numInt, 'e'); !ok {
		t.Fatalf("expected numInt exp transition")
	}
	if _, ok := numNextState(numInt, '5'); !ok {
		t.Fatalf("expected numInt digit transition")
	}
	if _, ok := numNextState(numInt, 'x'); ok {
		t.Fatalf("expected invalid transition for numInt other")
	}
	if _, ok := numNextState(numDot, '5'); !ok {
		t.Fatalf("expected numDot frac transition")
	}
	if _, ok := numNextState(numFrac, 'e'); !ok {
		t.Fatalf("expected numFrac exp transition")
	}
	if _, ok := numNextState(numFrac, '5'); !ok {
		t.Fatalf("expected numFrac digit transition")
	}
	if _, ok := numNextState(numExp, '+'); !ok {
		t.Fatalf("expected exp sign transition")
	}
	if _, ok := numNextState(numExp, '5'); !ok {
		t.Fatalf("expected exp digit transition")
	}
	if _, ok := numNextState(numExpSign, '5'); !ok {
		t.Fatalf("expected exp sign digit transition")
	}
	if _, ok := numNextState(numExpDigits, '5'); !ok {
		t.Fatalf("expected exp digits transition")
	}
	if _, ok := numNextState(numInvalid, '5'); ok {
		t.Fatalf("expected invalid state transition")
	}
	if _, ok := numNextState(numSign, 'x'); ok {
		t.Fatalf("expected invalid numSign transition")
	}
	if _, ok := numNextState(numDot, 'x'); ok {
		t.Fatalf("expected invalid numDot transition")
	}
	if _, ok := numNextState(numFrac, 'x'); ok {
		t.Fatalf("expected invalid numFrac transition")
	}
	if _, ok := numNextState(numExp, 'x'); ok {
		t.Fatalf("expected invalid numExp transition")
	}
	if _, ok := numNextState(numExpSign, 'x'); ok {
		t.Fatalf("expected invalid numExpSign transition")
	}
	if _, ok := numNextState(numExpDigits, 'x'); ok {
		t.Fatalf("expected invalid numExpDigits transition")
	}
	if numIsTerminal(numDot) {
		t.Fatalf("expected numDot to be non-terminal")
	}
	if !numIsTerminal(numInt) {
		t.Fatalf("expected numInt to be terminal")
	}
	if !isTerminator('}') || !isTerminator(',') || isTerminator('9') {
		t.Fatalf("isTerminator mismatch")
	}
}

func TestReadUnicodeEscapeBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)

	p.scanner.Reset(strings.NewReader("0041"))
	if r, err := p.readUnicodeEscape(); err != nil || r != 'A' {
		t.Fatalf("expected rune A, got %v err %v", r, err)
	}

	p.scanner.Reset(strings.NewReader("D834\\uDD1E"))
	if _, err := p.readUnicodeEscape(); err != nil {
		t.Fatalf("expected valid surrogate pair, got %v", err)
	}

	p.scanner.Reset(strings.NewReader("D800x"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid surrogate error")
	}

	p.scanner.Reset(strings.NewReader("D800\\u0001"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid low surrogate error")
	}

	p.scanner.Reset(strings.NewReader("D800\\x0000"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid surrogate prefix error")
	}

	p.scanner.Reset(strings.NewReader("ZZZZ"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected invalid hex error")
	}

	p.scanner.Reset(strings.NewReader("D800\\"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected short surrogate error")
	}

	p.scanner.Reset(strings.NewReader("D800"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected EOF after high surrogate")
	}

	p.scanner.Reset(strings.NewReader("D800\\u12"))
	if _, err := p.readUnicodeEscape(); err == nil {
		t.Fatalf("expected short low surrogate error")
	}

	p.scanner.Reset(strings.NewReader("12"))
	if _, err := p.readHex4(); err == nil {
		t.Fatalf("expected short hex error")
	}
}

func TestParseLiteralsNumbersAndStrings(t *testing.T) {
	opts := *DefaultOptions
	opts.Palette = "none"

	valid := []string{
		"true",
		"false",
		"null",
		"0",
		"-0",
		"-1.23e+4",
		"[1,2,3]",
		"[]",
		"{}",
		"\"simple\"",
		"\"\\\\\\/\\b\\f\\n\\r\\t\"",
		"\"\\u0041\"",
	}
	for _, input := range valid {
		var buf bytes.Buffer
		if err := streamPretty(&buf, strings.NewReader(input), &opts, NoColorPalette(), false); err != nil {
			t.Fatalf("expected valid input %q, got %v", input, err)
		}
	}

	opts.Unwrap = true
	if err := streamPretty(io.Discard, strings.NewReader("\"\\\\\\/\\b\\f\\n\\r\\t\""), &opts, NoColorPalette(), false); err != nil {
		t.Fatalf("unwrap readStringValue failed: %v", err)
	}
	if err := streamPretty(io.Discard, strings.NewReader("\"\\u0041\""), &opts, NoColorPalette(), false); err != nil {
		t.Fatalf("unwrap unicode failed: %v", err)
	}
	if err := streamPretty(io.Discard, strings.NewReader("\"\\uD834\\uDD1E\""), &opts, NoColorPalette(), false); err != nil {
		t.Fatalf("unwrap surrogate failed: %v", err)
	}
	if err := streamPretty(io.Discard, strings.NewReader("\"\\q\""), &opts, NoColorPalette(), false); err == nil {
		t.Fatalf("expected unwrap invalid escape error")
	}
}

func TestHelpersTrimAndLooksLikeJSON(t *testing.T) {
	if got := trimSpaceBytes([]byte("  a ")); string(got) != "a" {
		t.Fatalf("trimSpaceBytes failed: %q", string(got))
	}
	if got := trimSpaceBytes([]byte("a")); string(got) != "a" {
		t.Fatalf("trimSpaceBytes no trim failed: %q", string(got))
	}
	if looksLikeJSONBytes([]byte("x")) {
		t.Fatalf("looksLikeJSONBytes should be false for short input")
	}
	if looksLikeJSONBytes([]byte("xx")) {
		t.Fatalf("looksLikeJSONBytes should be false for non-json")
	}
	if !looksLikeJSONBytes([]byte("{}")) || !looksLikeJSONBytes([]byte("[]")) {
		t.Fatalf("looksLikeJSONBytes should be true for json")
	}
}

func TestAppendQuotedBytes(t *testing.T) {
	in := []byte("a\"b\\c\t\n\r\f\b\x01")
	out := appendQuotedBytes(nil, in)
	if len(out) == 0 || out[0] != '"' || out[len(out)-1] != '"' {
		t.Fatalf("appendQuotedBytes missing quotes")
	}
}

func TestParserUnwrapBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.unwrapDepth = 1

	p.scanner.Reset(strings.NewReader("\"{bad}\""))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected invalid json to remain string, got %v", err)
	}

	p.scanner.Reset(strings.NewReader("\"{\\\"a\\\":1}\""))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected unwrap to succeed: %v", err)
	}

	p.scanner.Reset(errReader{})
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected parseValue read error")
	}

	p.scanner.Reset(strings.NewReader("x"))
	if err := p.parseValueWithFirst(0, 'x'); err == nil {
		t.Fatalf("expected parseValueWithFirst invalid error")
	}
}

func TestParseNumberWithStyle(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, ColorPalette{Number: "x"}, &Options{}, false)
	p.scanner.Reset(strings.NewReader("1"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("parseValue number failed: %v", err)
	}
}

func TestParseLiteralWithStyle(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, ColorPalette{True: "x", False: "x", Null: "x"}, &Options{}, false)
	p.scanner.Reset(strings.NewReader("true"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("parseValue true failed: %v", err)
	}
	p.scanner.Reset(strings.NewReader("false"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("parseValue false failed: %v", err)
	}
	p.scanner.Reset(strings.NewReader("null"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("parseValue null failed: %v", err)
	}
}

func TestCopyStringTokenErrors(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\x01\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected control character error")
	}
	p.scanner.Reset(strings.NewReader("\\u12G4\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected invalid unicode escape error")
	}
	p.scanner.Reset(strings.NewReader("hi\\n\""))
	if err := p.copyStringToken("x"); err != nil {
		t.Fatalf("expected valid copyStringToken, got %v", err)
	}
	p.scanner.Reset(strings.NewReader("\\u0041\""))
	if err := p.copyStringToken(""); err != nil {
		t.Fatalf("expected valid unicode copyStringToken, got %v", err)
	}
}

func TestWriteQuotedBytesStyle(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(&byteWriter{}, ColorPalette{}, &Options{}, false)
	if err := p.writeQuotedBytes([]byte("hi"), "x"); err != nil {
		t.Fatalf("writeQuotedBytes failed: %v", err)
	}

	p.fmt.reset(errWriter{}, ColorPalette{}, &Options{}, false)
	if err := p.writeQuotedBytes([]byte("hi"), ""); err == nil {
		t.Fatalf("expected writeQuotedBytes error")
	}
}

func TestParseObjectArrayErrorPaths(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected parseValue error on writeBracket")
	}

	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("[]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected parseValue error on writeBracket array")
	}
}

func TestExpectColonErrors(t *testing.T) {
	var p parser
	p.scanner.Reset(strings.NewReader(""))
	if err := p.expectColon(); err == nil {
		t.Fatalf("expected expectColon EOF error")
	}
	p.scanner.Reset(strings.NewReader("x"))
	if err := p.expectColon(); err == nil {
		t.Fatalf("expected expectColon mismatch error")
	}
}

func TestParseNumberInvalidTransition(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("x"))
	if err := p.parseNumber('-'); err == nil {
		t.Fatalf("expected invalid number after '-'")
	}
}

func TestTryUnwrapBytesTrailingData(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	ok, err := p.tryUnwrapBytes([]byte("{\"a\":1}1"), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected unwrap to fail with trailing data")
	}
}

func TestParserResetBranches(t *testing.T) {
	prev := MaxNestedJSONDepth
	t.Cleanup(func() { MaxNestedJSONDepth = prev })
	MaxNestedJSONDepth = 0

	var p parser
	opts := &Options{Unwrap: true}
	p.reset(strings.NewReader(""), io.Discard, opts, NoColorPalette(), false)
	if p.unwrapDepth != 1 {
		t.Fatalf("expected unwrap depth fallback to 1, got %d", p.unwrapDepth)
	}

	p.reset(strings.NewReader(""), io.Discard, &Options{Unwrap: false}, NoColorPalette(), false)
	if p.unwrapDepth != 0 {
		t.Fatalf("expected unwrap depth 0 for disabled unwrap")
	}

	p.reset(strings.NewReader(""), io.Discard, nil, NoColorPalette(), false)
	if p.unwrapDepth != 0 {
		t.Fatalf("expected unwrap depth 0 for nil opts")
	}
}

func TestParseValueWithFirstIndentError(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(errStringWriter{}, NoColorPalette(), &Options{Prefix: ">", Indent: " "}, false)
	p.scanner.Reset(strings.NewReader("rue"))
	if err := p.parseValueWithFirst(0, 't'); err == nil {
		t.Fatalf("expected indent error")
	}
}

func TestParseObjectBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(&errAfterReader{data: []byte("{")})
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after object start")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected invalid object key error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"\x01\":1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected control character in key error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\" 1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected missing colon error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":x}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected invalid value error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after object value")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1,"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after object comma")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 0}, NoColorPalette(), &Options{}, true)
	p.scanner.Reset(strings.NewReader("{\"a\":1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected compact colon error")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 0}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected pretty colon error")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 1}, NoColorPalette(), &Options{}, true)
	p.scanner.Reset(strings.NewReader("{\"a\":1,\"b\":2}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected compact punctuation error")
	}

	p.fmt.reset(&failSecondNewlineWriter{}, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1,\"b\":2}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected separator newline error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1,\"b\":2}"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected broke separator success, got %v", err)
	}

	p.fmt.reset(&failSecondNewlineWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected closing newline error")
	}

	p.fmt.reset(&failSecondNewlineWriter{newlines: 1}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected initial newline error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("{\"a\":1 \"b\":2}"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected missing comma error")
	}
}

func TestParseArrayBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(&errAfterReader{data: []byte("[")})
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after array start")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{Width: 10, SemiCompact: true}, false)
	p.scanner.Reset(strings.NewReader("[1]"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected no-wrap array, got %v", err)
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{Width: 1, SemiCompact: true}, false)
	p.scanner.Reset(strings.NewReader("[1]"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected width wrap array, got %v", err)
	}

	p.fmt.reset(&failSecondNewlineWriter{newlines: 1}, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("[1]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected initial newline error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("[x]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected invalid array value error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("[1"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after array value")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("[1,"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected read error after array comma")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 0}, NoColorPalette(), &Options{}, true)
	p.scanner.Reset(strings.NewReader("[1,2]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected compact comma error")
	}

	p.fmt.reset(&failSecondNewlineWriter{}, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("[1,2]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected separator newline error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("[1,2]"))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected broke separator success, got %v", err)
	}

	p.fmt.reset(&failSecondNewlineWriter{}, NoColorPalette(), &Options{Width: 0}, false)
	p.scanner.Reset(strings.NewReader("[1]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected closing newline error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("[1 2]"))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected missing comma error")
	}
}

func TestParseLiteralErrorBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)

	p.scanner.Reset(strings.NewReader(""))
	if err := p.parseLiteral('t'); err == nil {
		t.Fatalf("expected literal EOF error")
	}

	p.scanner.Reset(strings.NewReader("x"))
	if err := p.parseLiteral('t'); err == nil {
		t.Fatalf("expected literal mismatch error")
	}

	if err := p.parseLiteral('x'); err == nil {
		t.Fatalf("expected invalid literal error")
	}
}

func TestParseNumberErrorBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt

	p.fmt.reset(errStringWriter{}, ColorPalette{Number: "x"}, &Options{}, false)
	p.scanner.Reset(strings.NewReader(""))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected number style error")
	}

	p.fmt.reset(errByteWriter{}, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(strings.NewReader(""))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected number writeByte error")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 1}, ColorPalette{Number: "x"}, &Options{}, false)
	p.scanner.Reset(strings.NewReader(""))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected number reset error")
	}

	p.fmt.reset(&failAfterByteWriter{fail: 1}, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(strings.NewReader("2"))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected number writeByte error on digit")
	}

	p.fmt.reset(io.Discard, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(errReader{})
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected peekByte error")
	}

	p.fmt.reset(io.Discard, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(strings.NewReader("x"))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected invalid number transition")
	}

	p.fmt.reset(io.Discard, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(strings.NewReader("e"))
	if err := p.parseNumber('1'); err == nil {
		t.Fatalf("expected invalid number terminal")
	}

	p.fmt.reset(io.Discard, ColorPalette{}, &Options{}, false)
	p.scanner.Reset(strings.NewReader(""))
	if err := p.parseNumber('x'); err == nil {
		t.Fatalf("expected invalid number start")
	}
}

func TestParseStringValueBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)

	p.unwrapDepth = 1
	p.scanner.Reset(strings.NewReader("\"plain\""))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected unwrap non-json string, got %v", err)
	}

	p.unwrapDepth = 1
	p.scanner.Reset(strings.NewReader("\"{bad}\""))
	if err := p.parseValue(0); err != nil {
		t.Fatalf("expected invalid embedded json to pass through, got %v", err)
	}

	p.unwrapDepth = 1
	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\"{\\\"a\\\":1}\""))
	if err := p.parseValue(0); err == nil {
		t.Fatalf("expected unwrap formatting error")
	}
}

func TestTryUnwrapBytesErrorBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	ok, err := p.tryUnwrapBytes([]byte("{bad}"), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected unwrap to fail on invalid json")
	}

	p.fmt.reset(errWriter{}, NoColorPalette(), &Options{}, false)
	if _, err := p.tryUnwrapBytes([]byte("{\"a\":1}"), 0); err == nil {
		t.Fatalf("expected unwrap write error")
	}
}

func TestCopyStringTokenMoreErrors(t *testing.T) {
	var p parser
	p.formatter = &p.fmt

	p.fmt.reset(errStringWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("a\""))
	if err := p.copyStringToken("x"); err == nil {
		t.Fatalf("expected style write error")
	}

	p.fmt.reset(errByteWriter{}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("a\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected writeByte error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\\q\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected invalid escape error")
	}

	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\\"))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected unterminated escape error")
	}

	p.scanner.Reset(strings.NewReader("abc"))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected unterminated string error")
	}

	p.scanner.Reset(strings.NewReader("\\u12"))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected short unicode escape error")
	}

	p.fmt.reset(&failAfterByteWriter{fail: 1}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("a\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected writeByte error for content")
	}

	p.fmt.reset(&failAfterByteWriter{fail: 2}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\\\"\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected writeByte error for escape")
	}

	p.fmt.reset(&failAfterByteWriter{fail: 3}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("\\u0041\""))
	if err := p.copyStringToken(""); err == nil {
		t.Fatalf("expected writeByte error for unicode")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 1}, NoColorPalette(), &Options{}, false)
	p.scanner.Reset(strings.NewReader("a\""))
	if err := p.copyStringToken("x"); err == nil {
		t.Fatalf("expected reset write error")
	}
}

func TestWriteQuotedBytesStyleErrors(t *testing.T) {
	var p parser
	p.formatter = &p.fmt

	p.fmt.reset(errStringWriter{}, ColorPalette{}, &Options{}, false)
	if err := p.writeQuotedBytes([]byte("hi"), "x"); err == nil {
		t.Fatalf("expected style write error")
	}

	p.fmt.reset(&failAfterStringWriter{fail: 1}, ColorPalette{}, &Options{}, false)
	if err := p.writeQuotedBytes([]byte("hi"), "x"); err == nil {
		t.Fatalf("expected reset error")
	}
}

func TestReadStringValueErrorBranches(t *testing.T) {
	var p parser
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, NoColorPalette(), &Options{}, false)

	p.scanner.Reset(strings.NewReader("abc"))
	if _, err := p.readStringValue(); err == nil {
		t.Fatalf("expected unterminated string error")
	}

	p.scanner.Reset(strings.NewReader("\\"))
	if _, err := p.readStringValue(); err == nil {
		t.Fatalf("expected unterminated escape error")
	}

	p.scanner.Reset(strings.NewReader("\\u12G4\""))
	if _, err := p.readStringValue(); err == nil {
		t.Fatalf("expected invalid unicode error")
	}

	p.scanner.Reset(strings.NewReader("\x01\""))
	if _, err := p.readStringValue(); err == nil {
		t.Fatalf("expected invalid control char error")
	}
}

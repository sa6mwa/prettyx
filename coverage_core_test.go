package prettyx

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"pkt.systems/prettyx/internal/ansi"
)

func TestPaletteResolutionBranches(t *testing.T) {
	opts := *DefaultOptions
	opts.Palette = "none"
	if _, err := resolvePalette(&opts, true); err != nil {
		t.Fatalf("resolvePalette none failed: %v", err)
	}

	opts.Palette = "does-not-exist"
	if _, err := resolvePalette(&opts, true); err == nil {
		t.Fatalf("expected error for unknown palette")
	}

	opts.Palette = ""
	if _, err := resolvePalette(&opts, false); err != nil {
		t.Fatalf("resolvePalette default failed: %v", err)
	}
	opts.Palette = "jq"
	if _, err := resolvePalette(&opts, true); err != nil {
		t.Fatalf("resolvePalette jq failed: %v", err)
	}

	custom := ansi.Palette{
		Key:    "k",
		String: "s",
		Num:    "n",
		Bool:   "b",
		Nil:    "x",
	}
	pal := colorPaletteFromAnsi(custom)
	if pal.Brackets != "x" || pal.Punctuation != "x" {
		t.Fatalf("expected fallback brackets/punctuation, got %+v", pal)
	}

	custom.Brackets = "["
	pal = colorPaletteFromAnsi(custom)
	if pal.Punctuation != "[" {
		t.Fatalf("expected punctuation fallback to brackets, got %+v", pal)
	}
}

func TestShouldColorBranches(t *testing.T) {
	opts := *DefaultOptions
	opts.Palette = "none"
	if shouldColor(io.Discard, &opts) {
		t.Fatalf("expected palette none to disable color")
	}

	opts.Palette = ""
	opts.ForceColor = true
	if !shouldColor(io.Discard, &opts) {
		t.Fatalf("expected ForceColor to enable color")
	}

	opts.ForceColor = false
	if shouldColor(&noStringWriter{}, &opts) {
		t.Fatalf("expected non-tty writer to disable color")
	}

	if shouldColor(fdWriter{}, nil) {
		t.Fatalf("expected fd writer without opts to disable color on non-tty")
	}
}

func TestPrettyReaderAndRenderer(t *testing.T) {
	input := []byte("{\"a\":1}")
	out, err := PrettyWithRenderer(input, DefaultOptions, nil)
	if err != nil {
		t.Fatalf("PrettyWithRenderer failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected output from PrettyWithRenderer")
	}

	reader := PrettyReader(bytes.NewReader(input), DefaultOptions)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("PrettyReader read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected data from PrettyReader")
	}
}

func TestCompactToBufferAndNewline(t *testing.T) {
	out, err := CompactToBuffer(strings.NewReader("{\"a\": 1}\n"), DefaultOptions)
	if err != nil {
		t.Fatalf("CompactToBuffer failed: %v", err)
	}
	if string(out) != "{\"a\":1}\n" {
		t.Fatalf("unexpected compact buffer: %q", out)
	}

	bw := &byteWriter{}
	if err := writeNewline(bw); err != nil {
		t.Fatalf("writeNewline bytewriter failed: %v", err)
	}
	if bw.String() != "\n" {
		t.Fatalf("unexpected newline via bytewriter: %q", bw.String())
	}

	nw := &noStringWriter{}
	if err := writeNewline(nw); err != nil {
		t.Fatalf("writeNewline fallback failed: %v", err)
	}
	if nw.String() != "\n" {
		t.Fatalf("unexpected newline via fallback: %q", nw.String())
	}
}

func TestPoolReleaseBranches(t *testing.T) {
	releaseParser(nil)
	releaseValueReader(nil)

	p := acquireParser()
	p.scratch = make([]byte, maxScratchCap+1)
	p.decodedBuf = make([]byte, maxScratchCap+1)
	releaseParser(p)

	v := acquireValueReader(bytes.NewReader(nil))
	releaseValueReader(v)
}

func TestScannerFillBranches(t *testing.T) {
	var s scanner
	s.Reset(&zeroReader{})
	if _, err := s.readByte(); err != io.EOF {
		t.Fatalf("expected EOF from zeroReader, got %v", err)
	}

	s.Reset(errReader{})
	if _, err := s.readByte(); err == nil {
		t.Fatalf("expected error from errReader")
	}
}

func TestPrettyStreamOptionsBranches(t *testing.T) {
	var buf bytes.Buffer
	if err := PrettyStream(&buf, strings.NewReader("{\"a\":1}"), nil); err != nil {
		t.Fatalf("PrettyStream default opts failed: %v", err)
	}

	opts := *DefaultOptions
	opts.Palette = "does-not-exist"
	if err := PrettyStream(&buf, strings.NewReader("{\"a\":1}"), &opts); err == nil {
		t.Fatalf("expected PrettyStream palette error")
	}
}

func TestPrettyToBufferPaletteError(t *testing.T) {
	opts := *DefaultOptions
	opts.Palette = "does-not-exist"
	if _, err := PrettyToBuffer(io.Discard, []byte("{\"a\":1}"), &opts); err == nil {
		t.Fatalf("expected PrettyToBuffer palette error")
	}

	if _, err := PrettyToBuffer(io.Discard, []byte("{\"a\":1}"), nil); err != nil {
		t.Fatalf("expected PrettyToBuffer with nil opts to succeed: %v", err)
	}

	opts = *DefaultOptions
	if _, err := PrettyToBuffer(io.Discard, []byte("{\"a\":"), &opts); err == nil {
		t.Fatalf("expected PrettyToBuffer parse error")
	}
}

package prettyx

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// MaxNestedJSONDepth controls how deep we recursively parse JSON that appears
// inside string values when Unwrap is enabled. Set to 10 by default. Special case:
//   - If MaxNestedJSONDepth == 0, we still unwrap one level (i.e., parse the
//     string as JSON once, but do not recurse further).
//
// Example meanings:
//
//	0  -> unwrap once (non-recursive)
//	1  -> unwrap once (same as 0)
//	2+ -> unwrap up to that many recursive levels
var MaxNestedJSONDepth = 10

// Options controls the pretty-printing behavior.
type Options struct {
	// Width is the soft-wrap column width for SemiCompact mode. When <= 0,
	// SemiCompact always breaks after commas.
	Width int
	// SemiCompact enables tidwall-style formatting with soft wrapping.
	// When false, output is jq-ish (one element/key per line).
	SemiCompact bool
	// Prefix is applied to every output line. Default "".
	Prefix string
	// Indent defines the nested indentation. Default two spaces.
	Indent string
	// Unwrap enables recursive decoding of JSON strings. This mirrors the CLI's
	// -u/--unwrap flag. When false, prettyx leaves any JSON-looking strings as-is.
	Unwrap bool
	// ForceColor emits ANSI color even when the destination is not a TTY.
	ForceColor bool
	// Palette selects the named colour palette. Empty chooses the default.
	// Use "none" to disable colour.
	Palette string
}

// DefaultOptions holds the fallback pretty-print configuration.
var DefaultOptions = &Options{
	Width:       80,
	SemiCompact: false,
	Prefix:      "",
	Indent:      "  ",
	Unwrap:      false,
	ForceColor:  false,
	Palette:     "default",
}

// shouldColor decides whether to emit ANSI sequences for the target writer.
func shouldColor(w io.Writer, opts *Options) bool {
	if opts != nil && strings.ToLower(strings.TrimSpace(opts.Palette)) == paletteNoneName {
		return false
	}
	if opts != nil && opts.ForceColor {
		return true
	}
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		return isatty.IsTerminal(f.Fd())
	}
	return false
}

// Pretty parses the input JSON, optionally unwraps nested JSON strings
// (recursing up to MaxNestedJSONDepth when Unwrap is enabled), formats it, and
// colorizes it using ANSI SGR sequences before returning the resulting bytes.
// Color output is automatically disabled when the destination is not a TTY
// unless ForceColor is set.
func Pretty(in []byte, opts *Options) ([]byte, error) {
	return PrettyToBuffer(os.Stdout, in, opts)
}

// PrettyWithRenderer mirrored Pretty when lipgloss was used. It is kept for
// callers temporarily but ignores the renderer. Prefer Pretty or PrettyTo.
func PrettyWithRenderer(in []byte, opts *Options, _ any) ([]byte, error) {
	return Pretty(in, opts)
}

// PrettyTo writes a pretty-printed, colorized JSON document to the provided
// writer. Colors degrade automatically when the writer is not a TTY.
func PrettyTo(w io.Writer, in []byte, opts *Options) error {
	return PrettyStream(w, bytes.NewReader(in), opts)
}

// PrettyToBuffer renders a pretty-printed JSON document into memory and returns
// the resulting bytes.
func PrettyToBuffer(w io.Writer, in []byte, opts *Options) ([]byte, error) {
	if opts == nil {
		opts = DefaultOptions
	}
	pal, err := resolvePalette(opts, shouldColor(w, opts))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := streamPretty(&buf, bytes.NewReader(in), opts, pal, false); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PrettyStream formats one or more JSON documents from r and writes them to w.
// It is the streaming, zero-alloc path when opts.Unwrap is false.
// When opts.SemiCompact is true, output uses tidwall-style soft wrapping.
func PrettyStream(w io.Writer, r io.Reader, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions
	}
	pal, err := resolvePalette(opts, shouldColor(w, opts))
	if err != nil {
		return err
	}
	return streamPretty(w, r, opts, pal, false)
}

// PrettyReader returns a reader that streams pretty-printed JSON from r.
// The returned reader must be closed to stop the internal goroutine.
func PrettyReader(r io.Reader, opts *Options) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		err := PrettyStream(pw, r, opts)
		_ = pw.CloseWithError(err)
	}()
	return pr
}

// ColorPalette configures the ANSI styles for each JSON token class.
type ColorPalette struct {
	Key         string
	String      string
	Number      string
	True        string
	False       string
	Null        string
	Brackets    string
	Punctuation string
}

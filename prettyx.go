// prettyx builds on the efficient formatting algorithm from Josh Baker's
// github.com/tidwall/pretty package. We inline the core routines so we can
// extend them with recursive JSON unwrapping and Lip Gloss styling.
package prettyx

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MaxNestedJSONDepth controls how deep we recursively parse JSON that appears
// inside string values. Set to 10 by default. Special case:
//   - If MaxNestedJSONDepth == 0, we still unwrap one level (i.e., parse the
//     string as JSON once, but do not recurse further).
//
// Example meanings:
//
//	0  -> unwrap once (non-recursive)
//	1  -> unwrap once (same as 0)
//	2+ -> unwrap up to that many recursive levels
var MaxNestedJSONDepth = 10

// Options controls the pretty-printing behavior. It mirrors the struct from
// github.com/tidwall/pretty.
type Options struct {
	// Width is the max column width for single-line arrays. Default 80.
	Width int
	// Prefix is applied to every output line. Default "".
	Prefix string
	// Indent defines the nested indentation. Default two spaces.
	Indent string
	// SortKeys sorts object keys alphabetically when true. Default false.
	SortKeys bool
	// NoUnwrap disables recursive decoding of JSON strings. This is equivalent to
	// jq's default behaviour (unless you call fromjson) and mirrors the CLI's
	// -no-unwrap flag. When true, prettyx leaves any JSON-looking strings as-is.
	NoUnwrap bool
}

// DefaultOptions holds the fallback pretty-print configuration.
var DefaultOptions = &Options{Width: 80, Prefix: "", Indent: "  ", SortKeys: false, NoUnwrap: false}

// prettyBuffer formats JSON using the provided options (or DefaultOptions).
// It retains the core tidwall/pretty behavior without unwrapping or coloring
// and is used internally before we apply lipgloss styling.
func prettyBuffer(jsonBytes []byte, opts *Options) []byte {
	if opts == nil {
		opts = DefaultOptions
	}
	buf := make([]byte, 0, len(jsonBytes))
	if len(opts.Prefix) != 0 {
		buf = append(buf, opts.Prefix...)
	}
	buf, _, _, _ = appendPrettyAny(buf, jsonBytes, 0, true,
		opts.Width, opts.Prefix, opts.Indent, opts.SortKeys,
		0, 0, -1)
	if len(buf) > 0 {
		buf = append(buf, '\n')
	}
	return buf
}

// Pretty parses the input JSON, unwraps nested JSON strings (recursing up to
// MaxNestedJSONDepth), formats it, and colorizes it with lipgloss before
// returning the resulting bytes. The renderer automatically adapts to the
// detected color capabilities of os.Stdout.
func Pretty(in []byte, opts *Options) ([]byte, error) {
	renderer := lipgloss.NewRenderer(os.Stdout)
	return PrettyWithRenderer(in, opts, renderer, nil)
}

// PrettyWithRenderer mirrors Pretty but allows callers to provide a custom
// lipgloss renderer and palette. Passing palette == nil uses the
// DefaultColorPalette derived from the renderer.
func PrettyWithRenderer(in []byte, opts *Options, renderer *lipgloss.Renderer, palette *ColorPalette) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.UseNumber() // avoid float64 surprises
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultOptions
	}

	if !opts.NoUnwrap {
		depth := MaxNestedJSONDepth
		if depth <= 0 {
			depth = 1 // "0" means unwrap once
		}
		v = unwrapNested(v, depth)
	}

	min, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	pretty := prettyBuffer(min, opts)

	if renderer == nil {
		renderer = lipgloss.NewRenderer(os.Stdout)
	}
	pal := ColorPalette{}
	if palette == nil {
		pal = DefaultColorPalette(renderer)
	} else {
		pal = *palette
	}
	colored := colorizeJSON(pretty, pal)
	return []byte(colored), nil
}

// PrettyTo writes a pretty-printed, colorized JSON document to the provided
// writer using a renderer bound to that writer. Colors degrade
// automatically when the writer is not a TTY.
func PrettyTo(w io.Writer, in []byte, opts *Options, palette *ColorPalette) error {
	renderer := lipgloss.NewRenderer(w)
	out, err := PrettyWithRenderer(in, opts, renderer, palette)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// unwrapNested recursively parses JSON-looking strings.
func unwrapNested(v any, depth int) any {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			x[k] = unwrapNested(vv, depth)
		}
		return x
	case []any:
		for i, vv := range x {
			x[i] = unwrapNested(vv, depth)
		}
		return x
	case json.Number, bool, nil:
		return x
	case string:
		if depth > 0 {
			if parsed, ok := tryParseInlineJSON(x, depth-1); ok {
				return parsed
			}
		}
		return x
	default:
		return x
	}
}

func tryParseInlineJSON(s string, nextDepth int) (any, bool) {
	b := trimSpaceBytes(s)
	if len(b) < 2 {
		return nil, false
	}
	first, last := b[0], b[len(b)-1]
	if !((first == '{' && last == '}') || (first == '[' && last == ']')) {
		return nil, false
	}

	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, false
	}
	return unwrapNested(v, nextDepth), true
}

func trimSpaceBytes(s string) []byte {
	b := []byte(s)
	return bytes.TrimSpace(b)
}

// ColorPalette configures the Lip Gloss styles for each JSON token class.
type ColorPalette struct {
	Key         lipgloss.Style
	String      lipgloss.Style
	Number      lipgloss.Style
	True        lipgloss.Style
	False       lipgloss.Style
	Null        lipgloss.Style
	Brackets    lipgloss.Style
	Punctuation lipgloss.Style
}

// DefaultColorPalette returns a VS Code-inspired theme tuned for Lip Gloss. The
// renderer governs how colors degrade on limited terminals.
func DefaultColorPalette(renderer *lipgloss.Renderer) ColorPalette {
	if renderer == nil {
		renderer = lipgloss.NewRenderer(os.Stdout)
	}
	return ColorPalette{
		Key:         renderer.NewStyle().Foreground(lipgloss.Color("#61AFEF")).Bold(true),
		String:      renderer.NewStyle().Foreground(lipgloss.Color("#98C379")),
		Number:      renderer.NewStyle().Foreground(lipgloss.Color("#D19A66")),
		True:        renderer.NewStyle().Foreground(lipgloss.Color("#56B6C2")),
		False:       renderer.NewStyle().Foreground(lipgloss.Color("#56B6C2")),
		Null:        renderer.NewStyle().Foreground(lipgloss.Color("#5C6370")).Faint(true),
		Brackets:    renderer.NewStyle().Foreground(lipgloss.Color("#ABB2BF")).Bold(true),
		Punctuation: renderer.NewStyle().Foreground(lipgloss.Color("#ABB2BF")),
	}
}

// NoColorPalette disables all styling while still routing through lipgloss so we
// benefit from its rendering decisions (width handling, etc.).
func NoColorPalette(renderer *lipgloss.Renderer) ColorPalette {
	if renderer == nil {
		renderer = lipgloss.NewRenderer(os.Stdout)
	}
	base := renderer.NewStyle()
	return ColorPalette{
		Key:         base,
		String:      base,
		Number:      base,
		True:        base,
		False:       base,
		Null:        base,
		Brackets:    base,
		Punctuation: base,
	}
}

// colorizeJSON walks the pretty-printed JSON and applies the palette styles.
func colorizeJSON(src []byte, palette ColorPalette) string {
	var sb strings.Builder
	sb.Grow(len(src) + len(src)/2)

	type stackFrame struct {
		kind      byte
		expectKey bool
	}
	stack := make([]stackFrame, 0, 8)

	for i := 0; i < len(src); {
		ch := src[i]
		switch ch {
		case '{':
			stack = append(stack, stackFrame{kind: '{', expectKey: true})
			sb.WriteString(palette.Brackets.Render("{"))
			i++
		case '[':
			stack = append(stack, stackFrame{kind: '[', expectKey: false})
			sb.WriteString(palette.Brackets.Render("["))
			i++
		case '}':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			sb.WriteString(palette.Brackets.Render("}"))
			i++
			if len(stack) > 0 && stack[len(stack)-1].kind == '{' {
				stack[len(stack)-1].expectKey = false
			}
		case ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			sb.WriteString(palette.Brackets.Render("]"))
			i++
		case ':':
			sb.WriteString(palette.Punctuation.Render(":"))
			if len(stack) > 0 && stack[len(stack)-1].kind == '{' {
				stack[len(stack)-1].expectKey = false
			}
			i++
		case ',':
			sb.WriteString(palette.Punctuation.Render(","))
			if len(stack) > 0 && stack[len(stack)-1].kind == '{' {
				stack[len(stack)-1].expectKey = true
			}
			i++
		case '"':
			start := i
			i++
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
					continue
				}
				if src[i] == '"' {
					i++
					break
				}
				i++
			}
			segment := string(src[start:i])
			isKey := len(stack) > 0 && stack[len(stack)-1].kind == '{' && stack[len(stack)-1].expectKey
			if isKey {
				sb.WriteString(palette.Key.Render(segment))
				stack[len(stack)-1].expectKey = false
			} else {
				sb.WriteString(palette.String.Render(segment))
			}
		default:
			if (ch >= '0' && ch <= '9') || ch == '-' {
				start := i
				i++
				for i < len(src) {
					c := src[i]
					if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-' {
						i++
					} else {
						break
					}
				}
				sb.WriteString(palette.Number.Render(string(src[start:i])))
				continue
			}
			if len(src)-i >= 4 && bytes.Equal(src[i:i+4], []byte("true")) {
				sb.WriteString(palette.True.Render("true"))
				i += 4
				continue
			}
			if len(src)-i >= 5 && bytes.Equal(src[i:i+5], []byte("false")) {
				sb.WriteString(palette.False.Render("false"))
				i += 5
				continue
			}
			if len(src)-i >= 4 && bytes.Equal(src[i:i+4], []byte("null")) {
				sb.WriteString(palette.Null.Render("null"))
				i += 4
				continue
			}
			sb.WriteByte(ch)
			i++
		}
	}
	return sb.String()
}

// The remaining functions are adapted from github.com/tidwall/pretty to keep the
// efficient formatting logic bundled with prettyx.

func appendPrettyAny(buf, jsonBytes []byte, i int, pretty bool, width int, prefix, indent string, sortkeys bool, tabs, nl, max int) ([]byte, int, int, bool) {
	for ; i < len(jsonBytes); i++ {
		if jsonBytes[i] <= ' ' {
			continue
		}
		if jsonBytes[i] == '"' {
			return appendPrettyString(buf, jsonBytes, i, nl)
		}

		if (jsonBytes[i] >= '0' && jsonBytes[i] <= '9') || jsonBytes[i] == '-' || isNaNOrInf(jsonBytes[i:]) {
			return appendPrettyNumber(buf, jsonBytes, i, nl)
		}
		if jsonBytes[i] == '{' {
			return appendPrettyObject(buf, jsonBytes, i, '{', '}', pretty, width, prefix, indent, sortkeys, tabs, nl, max)
		}
		if jsonBytes[i] == '[' {
			return appendPrettyObject(buf, jsonBytes, i, '[', ']', pretty, width, prefix, indent, sortkeys, tabs, nl, max)
		}
		switch jsonBytes[i] {
		case 't':
			return append(buf, 't', 'r', 'u', 'e'), i + 4, nl, true
		case 'f':
			return append(buf, 'f', 'a', 'l', 's', 'e'), i + 5, nl, true
		case 'n':
			return append(buf, 'n', 'u', 'l', 'l'), i + 4, nl, true
		}
	}
	return buf, i, nl, true
}

type pair struct {
	kstart, kend int
	vstart, vend int
}

type byKeyVal struct {
	sorted bool
	json   []byte
	buf    []byte
	pairs  []pair
}

func (arr *byKeyVal) Len() int { return len(arr.pairs) }
func (arr *byKeyVal) Less(i, j int) bool {
	if arr.isLess(i, j, byKey) {
		return true
	}
	if arr.isLess(j, i, byKey) {
		return false
	}
	return arr.isLess(i, j, byVal)
}
func (arr *byKeyVal) Swap(i, j int) {
	arr.pairs[i], arr.pairs[j] = arr.pairs[j], arr.pairs[i]
	arr.sorted = true
}

type byKind int

const (
	byKey byKind = 0
	byVal byKind = 1
)

type jtype int

const (
	jnull jtype = iota
	jfalse
	jnumber
	jstring
	jtrue
	jjson
)

func getjtype(v []byte) jtype {
	if len(v) == 0 {
		return jnull
	}
	switch v[0] {
	case '"':
		return jstring
	case 'f':
		return jfalse
	case 't':
		return jtrue
	case 'n':
		return jnull
	case '[', '{':
		return jjson
	default:
		return jnumber
	}
}

func (arr *byKeyVal) isLess(i, j int, kind byKind) bool {
	k1 := arr.json[arr.pairs[i].kstart:arr.pairs[i].kend]
	k2 := arr.json[arr.pairs[j].kstart:arr.pairs[j].kend]
	var v1, v2 []byte
	if kind == byKey {
		v1 = k1
		v2 = k2
	} else {
		v1 = bytes.TrimSpace(arr.buf[arr.pairs[i].vstart:arr.pairs[i].vend])
		v2 = bytes.TrimSpace(arr.buf[arr.pairs[j].vstart:arr.pairs[j].vend])
		if len(v1) >= len(k1)+1 {
			v1 = bytes.TrimSpace(v1[len(k1)+1:])
		}
		if len(v2) >= len(k2)+1 {
			v2 = bytes.TrimSpace(v2[len(k2)+1:])
		}
	}
	t1 := getjtype(v1)
	t2 := getjtype(v2)
	if t1 < t2 {
		return true
	}
	if t1 > t2 {
		return false
	}
	if t1 == jstring {
		s1 := parsestr(v1)
		s2 := parsestr(v2)
		return string(s1) < string(s2)
	}
	if t1 == jnumber {
		n1, _ := strconv.ParseFloat(string(v1), 64)
		n2, _ := strconv.ParseFloat(string(v2), 64)
		return n1 < n2
	}
	return string(v1) < string(v2)
}

func parsestr(s []byte) []byte {
	for i := 1; i < len(s); i++ {
		if s[i] == '\\' {
			var str string
			json.Unmarshal(s, &str)
			return []byte(str)
		}
		if s[i] == '"' {
			return s[1:i]
		}
	}
	return nil
}

func appendPrettyObject(buf, jsonBytes []byte, i int, open, close byte, pretty bool, width int, prefix, indent string, sortkeys bool, tabs, nl, max int) ([]byte, int, int, bool) {
	var ok bool
	if width > 0 {
		if pretty && open == '[' && max == -1 {
			max := width - (len(buf) - nl)
			if max > 3 {
				s1, s2 := len(buf), i
				buf, i, _, ok = appendPrettyObject(buf, jsonBytes, i, '[', ']', false, width, prefix, "", sortkeys, 0, 0, max)
				if ok && len(buf)-s1 <= max {
					return buf, i, nl, true
				}
				buf = buf[:s1]
				i = s2
			}
		} else if max != -1 && open == '{' {
			return buf, i, nl, false
		}
	}
	buf = append(buf, open)
	i++
	var pairs []pair
	if open == '{' && sortkeys {
		pairs = make([]pair, 0, 8)
	}
	var n int
	for ; i < len(jsonBytes); i++ {
		if jsonBytes[i] <= ' ' {
			continue
		}
		if jsonBytes[i] == close {
			if pretty {
				if open == '{' && sortkeys {
					buf = sortPairs(jsonBytes, buf, pairs)
				}
				if n > 0 {
					nl = len(buf)
					if buf[nl-1] == ' ' {
						buf[nl-1] = '\n'
					} else {
						buf = append(buf, '\n')
					}
				}
				if buf[len(buf)-1] != open {
					buf = appendTabs(buf, prefix, indent, tabs)
				}
			}
			buf = append(buf, close)
			return buf, i + 1, nl, open != '{'
		}
		if open == '[' || jsonBytes[i] == '"' {
			if n > 0 {
				buf = append(buf, ',')
				if width != -1 && open == '[' {
					buf = append(buf, ' ')
				}
			}
			var p pair
			if pretty {
				nl = len(buf)
				if buf[nl-1] == ' ' {
					buf[nl-1] = '\n'
				} else {
					buf = append(buf, '\n')
				}
				if open == '{' && sortkeys {
					p.kstart = i
					p.vstart = len(buf)
				}
				buf = appendTabs(buf, prefix, indent, tabs+1)
			}
			if open == '{' {
				buf, i, nl, _ = appendPrettyString(buf, jsonBytes, i, nl)
				if sortkeys {
					p.kend = i
				}
				buf = append(buf, ':')
				if pretty {
					buf = append(buf, ' ')
				}
			}
			buf, i, nl, ok = appendPrettyAny(buf, jsonBytes, i, pretty, width, prefix, indent, sortkeys, tabs+1, nl, max)
			if max != -1 && !ok {
				return buf, i, nl, false
			}
			if pretty && open == '{' && sortkeys {
				p.vend = len(buf)
				if p.kstart > p.kend || p.vstart > p.vend {
					sortkeys = false
				} else {
					pairs = append(pairs, p)
				}
			}
			i--
			n++
		}
	}
	return buf, i, nl, open != '{'
}

func sortPairs(jsonBytes, buf []byte, pairs []pair) []byte {
	if len(pairs) == 0 {
		return buf
	}
	vstart := pairs[0].vstart
	vend := pairs[len(pairs)-1].vend
	arr := byKeyVal{false, jsonBytes, buf, pairs}
	sort.Stable(&arr)
	if !arr.sorted {
		return buf
	}
	nbuf := make([]byte, 0, vend-vstart)
	for i, p := range pairs {
		nbuf = append(nbuf, buf[p.vstart:p.vend]...)
		if i < len(pairs)-1 {
			nbuf = append(nbuf, ',')
			nbuf = append(nbuf, '\n')
		}
	}
	return append(buf[:vstart], nbuf...)
}

func appendPrettyString(buf, jsonBytes []byte, i, nl int) ([]byte, int, int, bool) {
	s := i
	i++
	for ; i < len(jsonBytes); i++ {
		if jsonBytes[i] == '"' {
			var sc int
			for j := i - 1; j > s; j-- {
				if jsonBytes[j] == '\\' {
					sc++
				} else {
					break
				}
			}
			if sc%2 == 1 {
				continue
			}
			i++
			break
		}
	}
	return append(buf, jsonBytes[s:i]...), i, nl, true
}

func appendPrettyNumber(buf, jsonBytes []byte, i, nl int) ([]byte, int, int, bool) {
	s := i
	i++
	for ; i < len(jsonBytes); i++ {
		if jsonBytes[i] <= ' ' || jsonBytes[i] == ',' || jsonBytes[i] == ':' || jsonBytes[i] == ']' || jsonBytes[i] == '}' {
			break
		}
	}
	return append(buf, jsonBytes[s:i]...), i, nl, true
}

func appendTabs(buf []byte, prefix, indent string, tabs int) []byte {
	if len(prefix) != 0 {
		buf = append(buf, prefix...)
	}
	if len(indent) == 2 && indent[0] == ' ' && indent[1] == ' ' {
		for range tabs {
			buf = append(buf, ' ', ' ')
		}
	} else {
		for range tabs {
			buf = append(buf, indent...)
		}
	}
	return buf
}

func isNaNOrInf(src []byte) bool {
	return src[0] == 'i' ||
		src[0] == 'I' ||
		src[0] == '+' ||
		src[0] == 'N' ||
		(src[0] == 'n' && len(src) > 1 && src[1] != 'u')
}

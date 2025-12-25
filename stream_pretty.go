package prettyx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	"pkt.systems/prettyx/internal/ansi"
)

func streamPretty(w io.Writer, r io.Reader, opts *Options, pal ColorPalette, compact bool) error {
	p := acquireParser()
	defer releaseParser(p)
	p.reset(r, w, opts, pal, compact)

	for {
		err := p.scanner.skipSpace()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := p.parseValue(0); err != nil {
			return err
		}
		if err := p.formatter.writeByte('\n'); err != nil {
			return err
		}
		p.formatter.lineLen = 0
	}
}

type formatter struct {
	w           io.Writer
	bw          io.ByteWriter
	sw          io.StringWriter
	pal         ColorPalette
	prefix      string
	indent      string
	width       int
	compact     bool
	semiCompact bool
	lineLen     int
	byteBuf     [1]byte
}

func (f *formatter) reset(w io.Writer, pal ColorPalette, opts *Options, compact bool) {
	f.w = w
	f.pal = pal
	if opts != nil {
		f.prefix = opts.Prefix
		f.indent = opts.Indent
		f.width = opts.Width
		f.semiCompact = opts.SemiCompact
	} else {
		f.prefix = ""
		f.indent = ""
		f.width = 0
		f.semiCompact = false
	}
	f.compact = compact
	f.lineLen = 0
	if w == nil {
		f.bw = nil
		f.sw = nil
		return
	}
	if bw, ok := w.(io.ByteWriter); ok {
		f.bw = bw
	} else {
		f.bw = nil
	}
	if sw, ok := w.(io.StringWriter); ok {
		f.sw = sw
	} else {
		f.sw = nil
	}
}

func (f *formatter) clear() {
	f.w = nil
	f.bw = nil
	f.sw = nil
	f.pal = ColorPalette{}
	f.prefix = ""
	f.indent = ""
	f.width = 0
	f.compact = false
	f.semiCompact = false
	f.lineLen = 0
}

func (f *formatter) writeANSI(seq string) error {
	if seq == "" {
		return nil
	}
	var err error
	if f.sw != nil {
		_, err = f.sw.WriteString(seq)
	} else {
		_, err = io.WriteString(f.w, seq)
	}
	return err
}

func (f *formatter) writeBytes(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	_, err := f.w.Write(b)
	if err == nil {
		f.lineLen += len(b)
	}
	return err
}

func (f *formatter) writeString(s string) error {
	if s == "" {
		return nil
	}
	var err error
	if f.sw != nil {
		_, err = f.sw.WriteString(s)
	} else {
		_, err = io.WriteString(f.w, s)
	}
	if err == nil {
		f.lineLen += len(s)
	}
	return err
}

func (f *formatter) writeByte(b byte) error {
	if f.bw != nil {
		if err := f.bw.WriteByte(b); err != nil {
			return err
		}
	} else {
		f.byteBuf[0] = b
		if _, err := f.w.Write(f.byteBuf[:]); err != nil {
			return err
		}
	}
	f.lineLen++
	return nil
}

func (f *formatter) writeStyledString(style string, s string) error {
	if style != "" {
		if err := f.writeANSI(style); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(f.w, s); err != nil {
		return err
	}
	f.lineLen += len(s)
	if style != "" {
		if err := f.writeANSI(ansi.Reset); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) writeStyledByte(style string, b byte) error {
	if style != "" {
		if err := f.writeANSI(style); err != nil {
			return err
		}
	}
	if err := f.writeByte(b); err != nil {
		return err
	}
	if style != "" {
		if err := f.writeANSI(ansi.Reset); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) ensureLineStart(depth int) error {
	if f.lineLen != 0 {
		return nil
	}
	return f.writeIndent(depth)
}

func (f *formatter) writeIndent(depth int) error {
	if f.prefix != "" {
		if err := f.writeString(f.prefix); err != nil {
			return err
		}
	}
	if depth <= 0 || f.indent == "" {
		return nil
	}
	for i := 0; i < depth; i++ {
		if err := f.writeString(f.indent); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) newline(depth int) error {
	if err := f.writeByte('\n'); err != nil {
		return err
	}
	f.lineLen = 0
	return f.writeIndent(depth)
}

func (f *formatter) writeBracket(b byte) error {
	return f.writeStyledByte(f.pal.Brackets, b)
}

func (f *formatter) writePunctuation(s string) error {
	return f.writeStyledString(f.pal.Punctuation, s)
}

func (f *formatter) writeLiteral(lit string, style string) error {
	return f.writeStyledString(style, lit)
}

type parser struct {
	scanner     scanner
	formatter   *formatter
	fmt         formatter
	unwrapDepth int
	silentErr   bool
	scratch     []byte
	decodedBuf  []byte
	sliceReader bytes.Reader
}

func (p *parser) reset(r io.Reader, w io.Writer, opts *Options, pal ColorPalette, compact bool) {
	p.scanner.Reset(r)
	p.formatter = &p.fmt
	p.fmt.reset(w, pal, opts, compact)
	p.unwrapDepth = 0
	p.silentErr = false
	if opts != nil && opts.Unwrap {
		p.unwrapDepth = MaxNestedJSONDepth
		if p.unwrapDepth <= 0 {
			p.unwrapDepth = 1
		}
	}
}

var errInvalidJSON = errors.New("json: invalid")

func (p *parser) errorf(format string, args ...any) error {
	if p != nil && p.silentErr {
		return errInvalidJSON
	}
	return fmt.Errorf(format, args...)
}

func (p *parser) parseValue(depth int) error {
	b, err := p.scanner.readNonSpace()
	if err != nil {
		return err
	}
	return p.parseValueWithFirst(depth, b)
}

func (p *parser) parseValueWithFirst(depth int, first byte) error {
	if err := p.formatter.ensureLineStart(depth); err != nil {
		return err
	}
	switch first {
	case '{':
		return p.parseObject(depth)
	case '[':
		return p.parseArray(depth)
	case '"':
		return p.parseStringValue(depth)
	case 't', 'f', 'n':
		return p.parseLiteral(first)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber(first)
	default:
		return p.errorf("json: unexpected character %q", first)
	}
}

func (p *parser) parseObject(depth int) error {
	if err := p.formatter.writeBracket('{'); err != nil {
		return err
	}
	innerDepth := depth + 1

	b, err := p.scanner.readNonSpace()
	if err != nil {
		return err
	}
	if b == '}' {
		return p.formatter.writeBracket('}')
	}

	multiline := false
	if !p.formatter.compact {
		if err := p.formatter.newline(innerDepth); err != nil {
			return err
		}
		multiline = true
	}

	for {
		if b != '"' {
			return p.errorf("json: expected object key")
		}
		if err := p.copyStringToken(p.formatter.pal.Key); err != nil {
			return err
		}
		if err := p.expectColon(); err != nil {
			return err
		}
		if p.formatter.compact {
			if err := p.formatter.writePunctuation(":"); err != nil {
				return err
			}
		} else {
			if err := p.formatter.writePunctuation(": "); err != nil {
				return err
			}
		}
		if err := p.parseValue(innerDepth); err != nil {
			return err
		}
		b, err = p.scanner.readNonSpace()
		if err != nil {
			return err
		}
		switch b {
		case ',':
			if p.formatter.compact {
				if err := p.formatter.writePunctuation(","); err != nil {
					return err
				}
			} else {
				broke, err := p.writeSeparator(innerDepth)
				if err != nil {
					return err
				}
				if broke {
					multiline = true
				}
			}
			b, err = p.scanner.readNonSpace()
			if err != nil {
				return err
			}
			continue
		case '}':
			if !p.formatter.compact && multiline {
				if err := p.formatter.newline(depth); err != nil {
					return err
				}
			}
			return p.formatter.writeBracket('}')
		default:
			return p.errorf("json: expected ',' or '}'")
		}
	}
}

func (p *parser) parseArray(depth int) error {
	if err := p.formatter.writeBracket('['); err != nil {
		return err
	}
	innerDepth := depth + 1

	b, err := p.scanner.readNonSpace()
	if err != nil {
		return err
	}
	if b == ']' {
		return p.formatter.writeBracket(']')
	}

	multiline := false
	if !p.formatter.compact {
		shouldBreak := !p.formatter.semiCompact
		if p.formatter.semiCompact && (p.formatter.width <= 0 || p.formatter.lineLen >= p.formatter.width) {
			shouldBreak = true
		}
		if shouldBreak {
			if err := p.formatter.newline(innerDepth); err != nil {
				return err
			}
			multiline = true
		}
	}

	for {
		if err := p.parseValueWithFirst(innerDepth, b); err != nil {
			return err
		}
		b, err = p.scanner.readNonSpace()
		if err != nil {
			return err
		}
		switch b {
		case ',':
			if p.formatter.compact {
				if err := p.formatter.writePunctuation(","); err != nil {
					return err
				}
			} else {
				broke, err := p.writeSeparator(innerDepth)
				if err != nil {
					return err
				}
				if broke {
					multiline = true
				}
			}
			b, err = p.scanner.readNonSpace()
			if err != nil {
				return err
			}
			continue
		case ']':
			if !p.formatter.compact && multiline {
				if err := p.formatter.newline(depth); err != nil {
					return err
				}
			}
			return p.formatter.writeBracket(']')
		default:
			return p.errorf("json: expected ',' or ']'")
		}
	}
}

func (p *parser) writeSeparator(depth int) (bool, error) {
	if !p.formatter.semiCompact {
		if err := p.formatter.writePunctuation(","); err != nil {
			return false, err
		}
		if err := p.formatter.newline(depth); err != nil {
			return false, err
		}
		return true, nil
	}
	if p.formatter.width <= 0 {
		if err := p.formatter.writePunctuation(","); err != nil {
			return false, err
		}
		if err := p.formatter.newline(depth); err != nil {
			return false, err
		}
		return true, nil
	}
	if p.formatter.lineLen >= p.formatter.width {
		if err := p.formatter.writePunctuation(","); err != nil {
			return false, err
		}
		if err := p.formatter.newline(depth); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := p.formatter.writePunctuation(", "); err != nil {
		return false, err
	}
	return false, nil
}

func (p *parser) parseStringValue(depth int) error {
	if p.unwrapDepth <= 0 {
		return p.copyStringToken(p.formatter.pal.String)
	}

	val, err := p.readStringValue()
	if err != nil {
		return err
	}
	trimmed := trimSpaceBytes(val)
	if looksLikeJSONBytes(trimmed) {
		ok, err := p.tryUnwrapBytes(trimmed, depth)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return p.writeQuotedBytes(val, p.formatter.pal.String)
}

func (p *parser) tryUnwrapBytes(src []byte, depth int) (bool, error) {
	v := acquireParser()
	v.sliceReader.Reset(src)
	v.scanner.Reset(&v.sliceReader)
	v.formatter = &v.fmt
	v.fmt.reset(io.Discard, ColorPalette{}, nil, p.formatter.compact)
	v.fmt.width = p.formatter.width
	v.fmt.semiCompact = p.formatter.semiCompact
	v.unwrapDepth = 0
	v.silentErr = true
	if err := v.parseValue(0); err != nil {
		releaseParser(v)
		return false, nil
	}
	if err := v.scanner.skipSpace(); err != io.EOF {
		releaseParser(v)
		return false, nil
	}

	v.sliceReader.Reset(src)
	v.scanner.Reset(&v.sliceReader)
	v.formatter = p.formatter
	v.unwrapDepth = p.unwrapDepth - 1
	v.silentErr = false
	err := v.parseValue(depth)
	releaseParser(v)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *parser) copyStringToken(style string) error {
	if style != "" {
		if err := p.formatter.writeANSI(style); err != nil {
			return err
		}
	}
	if err := p.formatter.writeByte('"'); err != nil {
		return err
	}
	for {
		b, err := p.scanner.readByte()
		if err != nil {
			return err
		}
		if b < 0x20 {
			return p.errorf("json: invalid control character in string")
		}
		if err := p.formatter.writeByte(b); err != nil {
			return err
		}
		if b == '"' {
			break
		}
		if b == '\\' {
			esc, err := p.scanner.readByte()
			if err != nil {
				return err
			}
			if err := p.formatter.writeByte(esc); err != nil {
				return err
			}
			switch esc {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				continue
			case 'u':
				for i := 0; i < 4; i++ {
					ch, err := p.scanner.readByte()
					if err != nil {
						return err
					}
					if err := p.formatter.writeByte(ch); err != nil {
						return err
					}
					if !isHex(ch) {
						return p.errorf("json: invalid unicode escape")
					}
				}
			default:
				return p.errorf("json: invalid escape sequence")
			}
		}
	}
	if style != "" {
		if err := p.formatter.writeANSI(ansi.Reset); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) writeQuotedBytes(val []byte, style string) error {
	if style != "" {
		if err := p.formatter.writeANSI(style); err != nil {
			return err
		}
	}
	p.scratch = appendQuotedBytes(p.scratch[:0], val)
	if err := p.formatter.writeBytes(p.scratch); err != nil {
		return err
	}
	if style != "" {
		if err := p.formatter.writeANSI(ansi.Reset); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) readStringValue() ([]byte, error) {
	p.decodedBuf = p.decodedBuf[:0]
	for {
		b, err := p.scanner.readByte()
		if err != nil {
			return nil, err
		}
		if b == '"' {
			return p.decodedBuf, nil
		}
		if b < 0x20 {
			return nil, p.errorf("json: invalid control character in string")
		}
		if b != '\\' {
			p.decodedBuf = append(p.decodedBuf, b)
			continue
		}
		esc, err := p.scanner.readByte()
		if err != nil {
			return nil, err
		}
		switch esc {
		case '"', '\\', '/':
			p.decodedBuf = append(p.decodedBuf, esc)
		case 'b':
			p.decodedBuf = append(p.decodedBuf, '\b')
		case 'f':
			p.decodedBuf = append(p.decodedBuf, '\f')
		case 'n':
			p.decodedBuf = append(p.decodedBuf, '\n')
		case 'r':
			p.decodedBuf = append(p.decodedBuf, '\r')
		case 't':
			p.decodedBuf = append(p.decodedBuf, '\t')
		case 'u':
			r, err := p.readUnicodeEscape()
			if err != nil {
				return nil, err
			}
			p.decodedBuf = utf8.AppendRune(p.decodedBuf, r)
		default:
			return nil, p.errorf("json: invalid escape sequence")
		}
	}
}

func (p *parser) readUnicodeEscape() (rune, error) {
	n1, err := p.readHex4()
	if err != nil {
		return 0, err
	}
	if n1 < 0xD800 || n1 > 0xDBFF {
		return n1, nil
	}
	b, err := p.scanner.readByte()
	if err != nil {
		return 0, err
	}
	if b != '\\' {
		return utf8.RuneError, p.errorf("json: invalid surrogate pair")
	}
	b, err = p.scanner.readByte()
	if err != nil {
		return 0, err
	}
	if b != 'u' {
		return utf8.RuneError, p.errorf("json: invalid surrogate pair")
	}
	n2, err := p.readHex4()
	if err != nil {
		return 0, err
	}
	if n2 < 0xDC00 || n2 > 0xDFFF {
		return utf8.RuneError, p.errorf("json: invalid surrogate pair")
	}
	return utf16.DecodeRune(n1, n2), nil
}

func (p *parser) readHex4() (rune, error) {
	var val rune
	for i := 0; i < 4; i++ {
		b, err := p.scanner.readByte()
		if err != nil {
			return 0, err
		}
		if !isHex(b) {
			return 0, p.errorf("json: invalid unicode escape")
		}
		val = val<<4 | rune(fromHex(b))
	}
	return val, nil
}

func (p *parser) expectColon() error {
	b, err := p.scanner.readNonSpace()
	if err != nil {
		return err
	}
	if b != ':' {
		return p.errorf("json: expected ':' after object key")
	}
	return nil
}

func (p *parser) parseLiteral(first byte) error {
	var lit string
	var style string
	switch first {
	case 't':
		lit = "true"
		style = p.formatter.pal.True
	case 'f':
		lit = "false"
		style = p.formatter.pal.False
	case 'n':
		lit = "null"
		style = p.formatter.pal.Null
	default:
		return p.errorf("json: invalid literal")
	}
	for i := 1; i < len(lit); i++ {
		b, err := p.scanner.readByte()
		if err != nil {
			return err
		}
		if b != lit[i] {
			return p.errorf("json: invalid literal")
		}
	}
	return p.formatter.writeLiteral(lit, style)
}

func (p *parser) parseNumber(first byte) error {
	state, ok := numStartState(first)
	if !ok {
		return p.errorf("json: invalid number")
	}
	if p.formatter.pal.Number != "" {
		if err := p.formatter.writeANSI(p.formatter.pal.Number); err != nil {
			return err
		}
	}
	if err := p.formatter.writeByte(first); err != nil {
		return err
	}
	for {
		b, err := p.scanner.peekByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if isTerminator(b) {
			break
		}
		next, ok := numNextState(state, b)
		if !ok {
			return p.errorf("json: invalid number")
		}
		state = next
		_, _ = p.scanner.readByte()
		if err := p.formatter.writeByte(b); err != nil {
			return err
		}
	}
	if !numIsTerminal(state) {
		return p.errorf("json: invalid number")
	}
	if p.formatter.pal.Number != "" {
		if err := p.formatter.writeANSI(ansi.Reset); err != nil {
			return err
		}
	}
	return nil
}

func trimSpaceBytes(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && b[start] <= ' ' {
		start++
	}
	for start < end && b[end-1] <= ' ' {
		end--
	}
	return b[start:end]
}

func looksLikeJSONBytes(trimmed []byte) bool {
	if len(trimmed) < 2 {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

type scanner struct {
	r   io.Reader
	buf [4096]byte
	pos int
	n   int
}

func (s *scanner) Reset(r io.Reader) {
	s.r = r
	s.pos = 0
	s.n = 0
}

func (s *scanner) fill() error {
	n, err := s.r.Read(s.buf[:])
	if n == 0 {
		if err == nil {
			return io.EOF
		}
		return err
	}
	s.pos = 0
	s.n = n
	return nil
}

func (s *scanner) readByte() (byte, error) {
	if s.pos >= s.n {
		if err := s.fill(); err != nil {
			return 0, err
		}
	}
	b := s.buf[s.pos]
	s.pos++
	return b, nil
}

func (s *scanner) peekByte() (byte, error) {
	if s.pos >= s.n {
		if err := s.fill(); err != nil {
			return 0, err
		}
	}
	return s.buf[s.pos], nil
}

func (s *scanner) skipSpace() error {
	for {
		b, err := s.peekByte()
		if err != nil {
			return err
		}
		if b > ' ' {
			return nil
		}
		_, _ = s.readByte()
	}
}

func (s *scanner) readNonSpace() (byte, error) {
	if err := s.skipSpace(); err != nil {
		return 0, err
	}
	return s.readByte()
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func fromHex(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	default:
		return 0
	}
}

func isTerminator(b byte) bool {
	return b <= ' ' || b == ',' || b == '}' || b == ']'
}

type numState int

const (
	numInvalid numState = iota
	numSign
	numZero
	numInt
	numDot
	numFrac
	numExp
	numExpSign
	numExpDigits
)

func numStartState(first byte) (numState, bool) {
	switch {
	case first == '-':
		return numSign, true
	case first == '0':
		return numZero, true
	case first >= '1' && first <= '9':
		return numInt, true
	default:
		return numInvalid, false
	}
}

func numNextState(state numState, b byte) (numState, bool) {
	switch state {
	case numSign:
		if b == '0' {
			return numZero, true
		}
		if b >= '1' && b <= '9' {
			return numInt, true
		}
		return numInvalid, false
	case numZero:
		switch b {
		case '.':
			return numDot, true
		case 'e', 'E':
			return numExp, true
		}
		if b >= '0' && b <= '9' {
			return numInvalid, false
		}
		return numInvalid, false
	case numInt:
		switch b {
		case '.':
			return numDot, true
		case 'e', 'E':
			return numExp, true
		}
		if b >= '0' && b <= '9' {
			return numInt, true
		}
		return numInvalid, false
	case numDot:
		if b >= '0' && b <= '9' {
			return numFrac, true
		}
		return numInvalid, false
	case numFrac:
		switch b {
		case 'e', 'E':
			return numExp, true
		}
		if b >= '0' && b <= '9' {
			return numFrac, true
		}
		return numInvalid, false
	case numExp:
		switch b {
		case '+', '-':
			return numExpSign, true
		}
		if b >= '0' && b <= '9' {
			return numExpDigits, true
		}
		return numInvalid, false
	case numExpSign:
		if b >= '0' && b <= '9' {
			return numExpDigits, true
		}
		return numInvalid, false
	case numExpDigits:
		if b >= '0' && b <= '9' {
			return numExpDigits, true
		}
		return numInvalid, false
	default:
		return numInvalid, false
	}
}

func numIsTerminal(state numState) bool {
	switch state {
	case numZero, numInt, numFrac, numExpDigits:
		return true
	default:
		return false
	}
}

func appendQuotedBytes(dst []byte, s []byte) []byte {
	buf := dst
	if buf == nil {
		buf = make([]byte, 0, len(s)+2)
	} else {
		buf = buf[:0]
	}
	buf = append(buf, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"':
			buf = append(buf, '\\', c)
		case '\b':
			buf = append(buf, '\\', 'b')
		case '\f':
			buf = append(buf, '\\', 'f')
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		default:
			if c < 0x20 {
				buf = append(buf, '\\', 'u', '0', '0', hexDigit(c>>4), hexDigit(c&0x0f))
				continue
			}
			buf = append(buf, c)
		}
	}
	buf = append(buf, '"')
	return buf
}

func hexDigit(v byte) byte {
	if v < 10 {
		return '0' + v
	}
	return 'a' + (v - 10)
}

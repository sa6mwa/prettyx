package prettyx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	defaultUnwrapDepth      = 16
	defaultUnwrapStackDepth = 64
	defaultStringBufCap     = 256
)

type unwrapReader struct {
	sources    []unwrapSource
	sourcesBuf [defaultUnwrapDepth]unwrapSource
	used       int
	validator  parser
}

type unwrapSource struct {
	scanner   scanner
	depthLeft int

	stack    []containerState
	stackBuf [defaultUnwrapStackDepth]containerState

	topValueSeen bool
	done         bool

	emit    []byte
	emitPos int

	decodedBuf []byte
	decodedArr [defaultStringBufCap]byte
	scratch    []byte
	scratchArr [defaultStringBufCap]byte
	rawBuf     []byte
	rawArr     [defaultStringBufCap]byte

	sliceReader bytes.Reader
}

type containerState struct {
	typ            byte
	objPhase       objPhase
	objCount       int
	arrExpectValue bool
	arrNeedComma   bool
	arrCount       int
}

type objPhase int

const (
	objExpectKey objPhase = iota
	objExpectColon
	objExpectValue
	objExpectComma
)

var unwrapReaderPool = sync.Pool{
	New: func() any {
		return &unwrapReader{}
	},
}

func acquireUnwrapReader(r io.Reader, depth int) *unwrapReader {
	u := unwrapReaderPool.Get().(*unwrapReader)
	u.reset(r, depth)
	return u
}

func releaseUnwrapReader(u *unwrapReader) {
	if u == nil {
		return
	}
	u.clear()
	unwrapReaderPool.Put(u)
}

func (u *unwrapReader) reset(r io.Reader, depth int) {
	if cap(u.sources) == 0 {
		u.sources = u.sourcesBuf[:1]
	} else {
		u.sources = u.sources[:1]
	}
	u.used = 1
	u.sources[0].resetFromReader(r, depth)
}

func (u *unwrapReader) clear() {
	if u.used > 0 && len(u.sources) < u.used {
		u.sources = u.sources[:u.used]
	}
	for i := 0; i < u.used; i++ {
		u.sources[i].clear()
	}
	u.sources = u.sources[:0]
	u.used = 0
	u.validator.scanner.Reset(nil)
	u.validator.fmt.clear()
	u.validator.formatter = nil
	u.validator.unwrapDepth = 0
	u.validator.silentErr = false
	u.validator.sliceReader.Reset(nil)
	if cap(u.validator.scratch) > maxScratchCap {
		u.validator.scratch = nil
	} else if u.validator.scratch != nil {
		u.validator.scratch = u.validator.scratch[:0]
	}
	if cap(u.validator.decodedBuf) > maxScratchCap {
		u.validator.decodedBuf = nil
	} else if u.validator.decodedBuf != nil {
		u.validator.decodedBuf = u.validator.decodedBuf[:0]
	}
}

func (u *unwrapReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := 0
	for n < len(p) {
		b, err := u.nextByte()
		if err != nil {
			if err == io.EOF {
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

func (u *unwrapReader) nextByte() (byte, error) {
	for {
		if len(u.sources) == 0 {
			return 0, io.EOF
		}
		src := &u.sources[len(u.sources)-1]
		b, err := src.nextByte(u)
		if err == errContinue {
			continue
		}
		if err == io.EOF {
			u.sources = u.sources[:len(u.sources)-1]
			continue
		}
		return b, err
	}
}

func (u *unwrapReader) pushSource(b []byte, depth int) {
	idx := len(u.sources)
	if idx < cap(u.sources) {
		u.sources = u.sources[:idx+1]
	} else {
		u.sources = append(u.sources, unwrapSource{})
	}
	if u.used < len(u.sources) {
		u.used = len(u.sources)
	}
	u.sources[idx].resetFromBytes(b, depth)
}

func (s *unwrapSource) resetFromReader(r io.Reader, depth int) {
	s.scanner.Reset(r)
	s.resetCommon(depth)
}

func (s *unwrapSource) resetFromBytes(b []byte, depth int) {
	s.sliceReader.Reset(b)
	s.scanner.Reset(&s.sliceReader)
	s.resetCommon(depth)
}

func (s *unwrapSource) resetCommon(depth int) {
	s.depthLeft = depth
	s.stack = s.stackBuf[:0]
	s.topValueSeen = false
	s.done = false
	s.emit = nil
	s.emitPos = 0
	if cap(s.decodedBuf) == 0 {
		s.decodedBuf = s.decodedArr[:0]
	} else {
		s.decodedBuf = s.decodedBuf[:0]
	}
	if cap(s.scratch) == 0 {
		s.scratch = s.scratchArr[:0]
	} else {
		s.scratch = s.scratch[:0]
	}
	if cap(s.rawBuf) == 0 {
		s.rawBuf = s.rawArr[:0]
	} else {
		s.rawBuf = s.rawBuf[:0]
	}
}

func (s *unwrapSource) clear() {
	s.scanner.Reset(nil)
	s.sliceReader.Reset(nil)
	s.depthLeft = 0
	s.stack = s.stack[:0]
	s.topValueSeen = false
	s.done = false
	s.emit = nil
	s.emitPos = 0
	if cap(s.decodedBuf) > maxScratchCap {
		s.decodedBuf = nil
	} else {
		s.decodedBuf = s.decodedBuf[:0]
	}
	if cap(s.scratch) > maxScratchCap {
		s.scratch = nil
	} else {
		s.scratch = s.scratch[:0]
	}
	if cap(s.rawBuf) > maxScratchCap {
		s.rawBuf = nil
	} else {
		s.rawBuf = s.rawBuf[:0]
	}
}

var errContinue = errors.New("continue")

var (
	litTrue  = [...]byte{'t', 'r', 'u', 'e'}
	litFalse = [...]byte{'f', 'a', 'l', 's', 'e'}
	litNull  = [...]byte{'n', 'u', 'l', 'l'}
)

func (s *unwrapSource) nextByte(u *unwrapReader) (byte, error) {
	if s.emitPos < len(s.emit) {
		b := s.emit[s.emitPos]
		s.emitPos++
		if s.emitPos >= len(s.emit) {
			s.emit = nil
			s.emitPos = 0
		}
		return b, nil
	}
	if s.done {
		return 0, io.EOF
	}

	b, err := s.scanner.readNonSpace()
	if err != nil {
		if err == io.EOF && s.topValueSeen {
			s.done = true
			return 0, io.EOF
		}
		return 0, err
	}

	if len(s.stack) == 0 {
		if s.topValueSeen {
			return 0, fmt.Errorf("json: multiple top-level values")
		}
		s.topValueSeen = true
		return s.handleValue(u, b)
	}

	frame := &s.stack[len(s.stack)-1]
	if frame.typ == '{' {
		switch frame.objPhase {
		case objExpectKey:
			if b == '}' {
				if frame.objCount != 0 {
					return 0, fmt.Errorf("json: expected object key")
				}
				s.stack = s.stack[:len(s.stack)-1]
				s.valueComplete()
				return '}', nil
			}
			if b != '"' {
				return 0, fmt.Errorf("json: expected object key")
			}
			token, err := s.readRawStringToken()
			if err != nil {
				return 0, err
			}
			frame.objPhase = objExpectColon
			s.emit = token
			return s.nextByte(u)
		case objExpectColon:
			if b != ':' {
				return 0, fmt.Errorf("json: expected ':' after object key")
			}
			frame.objPhase = objExpectValue
			return ':', nil
		case objExpectValue:
			return s.handleValue(u, b)
		case objExpectComma:
			if b == ',' {
				frame.objPhase = objExpectKey
				return ',', nil
			}
			if b == '}' {
				s.stack = s.stack[:len(s.stack)-1]
				s.valueComplete()
				return '}', nil
			}
			return 0, fmt.Errorf("json: expected ',' or '}'")
		}
	}

	// array
	if frame.arrExpectValue {
		if b == ']' {
			if frame.arrCount != 0 {
				return 0, fmt.Errorf("json: expected array value")
			}
			s.stack = s.stack[:len(s.stack)-1]
			s.valueComplete()
			return ']', nil
		}
		return s.handleValue(u, b)
	}

	if b == ',' {
		frame.arrExpectValue = true
		frame.arrNeedComma = false
		return ',', nil
	}
	if b == ']' {
		s.stack = s.stack[:len(s.stack)-1]
		s.valueComplete()
		return ']', nil
	}
	return 0, fmt.Errorf("json: expected ',' or ']'")
}

func (s *unwrapSource) handleValue(u *unwrapReader, first byte) (byte, error) {
	switch first {
	case '{':
		s.pushObject()
		return '{', nil
	case '[':
		s.pushArray()
		return '[', nil
	case '"':
		val, err := s.readStringValue()
		if err != nil {
			return 0, err
		}
		if s.depthLeft > 0 {
			trimmed := trimSpaceBytes(val)
			if looksLikeJSONBytes(trimmed) && u.validateJSONBytes(trimmed) {
				s.valueComplete()
				u.pushSource(trimmed, s.depthLeft-1)
				return 0, errContinue
			}
		}
		s.scratch = appendQuotedBytes(s.scratch[:0], val)
		s.emit = s.scratch
		s.valueComplete()
		return s.nextByte(u)
	case 't', 'f', 'n':
		lit, err := s.readLiteral(first)
		if err != nil {
			return 0, err
		}
		s.emit = lit
		s.valueComplete()
		return s.nextByte(u)
	default:
		if _, ok := numStartState(first); ok {
			num, err := s.readNumber(first)
			if err != nil {
				return 0, err
			}
			s.emit = num
			s.valueComplete()
			return s.nextByte(u)
		}
		return 0, fmt.Errorf("json: unexpected character %q", first)
	}
}

func (s *unwrapSource) pushObject() {
	if len(s.stack) < cap(s.stack) {
		s.stack = s.stack[:len(s.stack)+1]
	} else {
		s.stack = append(s.stack, containerState{})
	}
	frame := &s.stack[len(s.stack)-1]
	*frame = containerState{typ: '{', objPhase: objExpectKey}
}

func (s *unwrapSource) pushArray() {
	if len(s.stack) < cap(s.stack) {
		s.stack = s.stack[:len(s.stack)+1]
	} else {
		s.stack = append(s.stack, containerState{})
	}
	frame := &s.stack[len(s.stack)-1]
	*frame = containerState{typ: '[', arrExpectValue: true}
}

func (s *unwrapSource) valueComplete() {
	if len(s.stack) == 0 {
		s.done = true
		return
	}
	frame := &s.stack[len(s.stack)-1]
	if frame.typ == '{' {
		frame.objPhase = objExpectComma
		frame.objCount++
		return
	}
	frame.arrExpectValue = false
	frame.arrNeedComma = true
	frame.arrCount++
}

func (s *unwrapSource) readLiteral(first byte) ([]byte, error) {
	var lit []byte
	switch first {
	case 't':
		lit = litTrue[:]
	case 'f':
		lit = litFalse[:]
	case 'n':
		lit = litNull[:]
	default:
		return nil, fmt.Errorf("json: invalid literal")
	}
	for i := 1; i < len(lit); i++ {
		b, err := s.scanner.readByte()
		if err != nil {
			return nil, fmt.Errorf("json: unexpected end in literal")
		}
		if b != lit[i] {
			return nil, fmt.Errorf("json: invalid literal")
		}
	}
	return lit, nil
}

func (s *unwrapSource) readNumber(first byte) ([]byte, error) {
	state, ok := numStartState(first)
	if !ok {
		return nil, fmt.Errorf("json: invalid number")
	}
	s.scratch = s.scratch[:0]
	s.scratch = append(s.scratch, first)
	for {
		b, err := s.scanner.peekByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if isTerminator(b) {
			break
		}
		next, ok := numNextState(state, b)
		if !ok {
			return nil, fmt.Errorf("json: invalid number")
		}
		state = next
		_, _ = s.scanner.readByte()
		s.scratch = append(s.scratch, b)
	}
	if !numIsTerminal(state) {
		return nil, fmt.Errorf("json: invalid number")
	}
	return s.scratch, nil
}

func (s *unwrapSource) readRawStringToken() ([]byte, error) {
	s.rawBuf = s.rawBuf[:0]
	s.rawBuf = append(s.rawBuf, '"')
	for {
		b, err := s.scanner.readByte()
		if err != nil {
			return nil, fmt.Errorf("json: unterminated string")
		}
		if b < 0x20 {
			return nil, fmt.Errorf("json: invalid control character in string")
		}
		s.rawBuf = append(s.rawBuf, b)
		if b == '"' {
			return s.rawBuf, nil
		}
		if b == '\\' {
			esc, err := s.scanner.readByte()
			if err != nil {
				return nil, fmt.Errorf("json: unterminated escape sequence")
			}
			s.rawBuf = append(s.rawBuf, esc)
			switch esc {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				continue
			case 'u':
				for i := 0; i < 4; i++ {
					ch, err := s.scanner.readByte()
					if err != nil {
						return nil, fmt.Errorf("json: invalid unicode escape")
					}
					if !isHex(ch) {
						return nil, fmt.Errorf("json: invalid unicode escape")
					}
					s.rawBuf = append(s.rawBuf, ch)
				}
			default:
				return nil, fmt.Errorf("json: invalid escape sequence")
			}
		}
	}
}

func (s *unwrapSource) readStringValue() ([]byte, error) {
	s.decodedBuf = s.decodedBuf[:0]
	for {
		b, err := s.scanner.readByte()
		if err != nil {
			return nil, fmt.Errorf("json: unterminated string")
		}
		if b == '"' {
			return s.decodedBuf, nil
		}
		if b < 0x20 {
			return nil, fmt.Errorf("json: invalid control character in string")
		}
		if b != '\\' {
			s.decodedBuf = append(s.decodedBuf, b)
			continue
		}
		esc, err := s.scanner.readByte()
		if err != nil {
			return nil, fmt.Errorf("json: unterminated escape sequence")
		}
		switch esc {
		case '"', '\\', '/':
			s.decodedBuf = append(s.decodedBuf, esc)
		case 'b':
			s.decodedBuf = append(s.decodedBuf, '\b')
		case 'f':
			s.decodedBuf = append(s.decodedBuf, '\f')
		case 'n':
			s.decodedBuf = append(s.decodedBuf, '\n')
		case 'r':
			s.decodedBuf = append(s.decodedBuf, '\r')
		case 't':
			s.decodedBuf = append(s.decodedBuf, '\t')
		case 'u':
			r, err := s.readUnicodeEscape()
			if err != nil {
				return nil, err
			}
			s.decodedBuf = utf8.AppendRune(s.decodedBuf, r)
		default:
			return nil, fmt.Errorf("json: invalid escape sequence")
		}
	}
}

func (s *unwrapSource) readUnicodeEscape() (rune, error) {
	n1, err := s.readHex4()
	if err != nil {
		return 0, err
	}
	if n1 < 0xD800 || n1 > 0xDBFF {
		return n1, nil
	}
	b, err := s.scanner.readByte()
	if err != nil {
		return 0, fmt.Errorf("json: invalid surrogate pair")
	}
	if b != '\\' {
		return utf8.RuneError, fmt.Errorf("json: invalid surrogate pair")
	}
	b, err = s.scanner.readByte()
	if err != nil {
		return 0, fmt.Errorf("json: invalid surrogate pair")
	}
	if b != 'u' {
		return utf8.RuneError, fmt.Errorf("json: invalid surrogate pair")
	}
	n2, err := s.readHex4()
	if err != nil {
		return 0, err
	}
	if n2 < 0xDC00 || n2 > 0xDFFF {
		return utf8.RuneError, fmt.Errorf("json: invalid surrogate pair")
	}
	return utf16.DecodeRune(n1, n2), nil
}

func (s *unwrapSource) readHex4() (rune, error) {
	var val rune
	for i := 0; i < 4; i++ {
		b, err := s.scanner.readByte()
		if err != nil {
			return 0, fmt.Errorf("json: invalid unicode escape")
		}
		if !isHex(b) {
			return 0, fmt.Errorf("json: invalid unicode escape")
		}
		val = val<<4 | rune(fromHex(b))
	}
	return val, nil
}

func (u *unwrapReader) validateJSONBytes(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	p := &u.validator
	if cap(p.scratch) == 0 {
		p.scratch = make([]byte, 0, defaultStringBufCap)
	} else {
		p.scratch = p.scratch[:0]
	}
	if cap(p.decodedBuf) == 0 {
		p.decodedBuf = make([]byte, 0, defaultStringBufCap)
	} else {
		p.decodedBuf = p.decodedBuf[:0]
	}
	p.sliceReader.Reset(b)
	p.scanner.Reset(&p.sliceReader)
	p.formatter = &p.fmt
	p.fmt.reset(io.Discard, ColorPalette{}, nil, true)
	p.unwrapDepth = 0
	p.silentErr = true
	err := p.parseValue(0)
	if err == nil {
		err = p.scanner.skipSpace()
	}
	ok := err == io.EOF
	return ok
}

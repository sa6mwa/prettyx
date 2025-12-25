package prettyx

import (
	"io"
	"sync"
)

const maxScratchCap = 64 * 1024

var parserPool = sync.Pool{
	New: func() any {
		return &parser{}
	},
}

var valueReaderPool = sync.Pool{
	New: func() any {
		return &valueReader{}
	},
}

func acquireParser() *parser {
	return parserPool.Get().(*parser)
}

func releaseParser(p *parser) {
	if p == nil {
		return
	}
	p.scanner.Reset(nil)
	p.fmt.clear()
	p.formatter = nil
	p.unwrapDepth = 0
	p.silentErr = false
	p.sliceReader.Reset(nil)
	if cap(p.scratch) > maxScratchCap {
		p.scratch = nil
	} else {
		p.scratch = p.scratch[:0]
	}
	if cap(p.decodedBuf) > maxScratchCap {
		p.decodedBuf = nil
	} else {
		p.decodedBuf = p.decodedBuf[:0]
	}
	parserPool.Put(p)
}

func acquireValueReader(r io.Reader) *valueReader {
	v := valueReaderPool.Get().(*valueReader)
	v.scanner.Reset(r)
	v.Reset()
	return v
}

func releaseValueReader(v *valueReader) {
	if v == nil {
		return
	}
	v.scanner.Reset(nil)
	v.Reset()
	valueReaderPool.Put(v)
}

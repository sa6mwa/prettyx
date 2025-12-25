package prettyx

import (
	"bytes"
	"strings"
	"testing"
)

const benchDocString = `{
  "str": "hello \"world\" \\ / \b \f \n \r \t",
  "unicode": "snowman \u2603",
  "empty_obj": {},
  "empty_arr": [],
  "int": 123,
  "big": 1234567890,
  "neg": -45,
  "neg_zero": -0,
  "float": 3.14159,
  "exp": 1.23e+4,
  "exp_small": -2.5E-3,
  "bools": [true, false],
  "nil": null,
  "arr": [1, "two", {"three":3}, [4,5]],
  "obj": {"a":1, "b":{"c":[{"d":"e"}]}},
  "json_str_obj": "{\"x\":1,\"y\":[true,false,null],\"z\":{\"k\":\"v\"}}",
  "json_str_arr": " [1, 2, {\"a\":\"b\"}] ",
  "json_str_invalid": "{oops}",
  "json_str_deep": "{\"inner\":\"{\\\"deep\\\":[1,2]}\"}"
}`

var benchDocBytes = []byte(benchDocString)
var benchJSONL = buildBenchJSONL()

var benchPrettySink []byte
var benchCompactSink []byte

func warmPools() {
	p := acquireParser()
	releaseParser(p)
	v := acquireValueReader(bytes.NewReader(nil))
	releaseValueReader(v)
	u := acquireUnwrapReader(bytes.NewReader(nil), 1)
	releaseUnwrapReader(u)
}

func buildBenchJSONL() []byte {
	var b strings.Builder
	baseLines := []string{
		benchDocString,
		`["mixed",1,true,null,{"a":2}]`,
		`"just a string"`,
		`123`,
		`-45`,
		`3.14159`,
		`true`,
		`false`,
		`null`,
	}
	for _, line := range baseLines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	for i := 0; i < 8; i++ {
		b.WriteString(benchDocString)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func BenchmarkPretty_NoUnwrap(b *testing.B) {
	benchmarkPretty(b, false)
}

func BenchmarkPretty_Unwrap(b *testing.B) {
	benchmarkPretty(b, true)
}

func BenchmarkPrettyStream_NoUnwrap(b *testing.B) {
	benchmarkPrettyStream(b, false)
}

func BenchmarkPrettyStream_Unwrap(b *testing.B) {
	benchmarkPrettyStream(b, true)
}

func benchmarkPretty(b *testing.B, unwrap bool) {
	opts := *DefaultOptions
	opts.Unwrap = unwrap
	opts.Palette = "none"

	warmPools()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := Pretty(benchDocBytes, &opts)
		if err != nil {
			b.Fatal(err)
		}
		benchPrettySink = out
	}
}

func benchmarkPrettyStream(b *testing.B, unwrap bool) {
	opts := *DefaultOptions
	opts.Unwrap = unwrap
	opts.Palette = "none"

	var out bytes.Buffer
	reader := bytes.NewReader(benchDocBytes)

	warmPools()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out.Reset()
		reader.Reset(benchDocBytes)
		if err := PrettyStream(&out, reader, &opts); err != nil {
			b.Fatal(err)
		}
		benchPrettySink = out.Bytes()
	}
}

func BenchmarkCompact_NoUnwrap(b *testing.B) {
	benchmarkCompact(b, false)
}

func BenchmarkCompact_Unwrap(b *testing.B) {
	benchmarkCompact(b, true)
}

func benchmarkCompact(b *testing.B, unwrap bool) {
	opts := *DefaultOptions
	opts.Unwrap = unwrap

	var out bytes.Buffer
	reader := bytes.NewReader(benchJSONL)

	warmPools()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out.Reset()
		reader.Reset(benchJSONL)
		if err := CompactTo(&out, reader, &opts); err != nil {
			b.Fatal(err)
		}
		benchCompactSink = out.Bytes()
	}
}

package prettyx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
)

const fuzzMaxInput = 1 << 20

func FuzzPrettyStream(f *testing.F) {
	seeds := [][]byte{
		[]byte("null"),
		[]byte("true"),
		[]byte("123"),
		[]byte("\"hello\""),
		[]byte("[1,2,3]"),
		[]byte("{\"a\":1,\"b\":[true,false],\"c\":null}"),
		[]byte("  {\"a\":1}  "),
		[]byte("{\"payload\":\"{\\\"a\\\":1}\"}"),
		sampleJSON,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			return
		}

		opts := *DefaultOptions
		opts.Palette = "none"
		opts.Unwrap = false

		var buf bytes.Buffer
		err := PrettyStream(&buf, bytes.NewReader(data), &opts)
		if err != nil {
			if _, ok := decodeSingleJSON(data); ok {
				t.Fatalf("PrettyStream failed for valid JSON: %v", err)
			}
			return
		}

		if err := decodeJSONStream(buf.Bytes()); err != nil {
			t.Fatalf("PrettyStream output is not valid JSON: %v", err)
		}

		if inVal, ok := decodeSingleJSON(data); ok {
			outVal, ok := decodeSingleJSON(bytes.TrimSpace(buf.Bytes()))
			if !ok {
				t.Fatalf("expected single JSON output for single input")
			}
			if !reflect.DeepEqual(inVal, outVal) {
				t.Fatalf("PrettyStream output mismatch\ninput: %s\noutput: %s", data, buf.Bytes())
			}
		}
	})
}

func FuzzPrettyStreamUnwrap(f *testing.F) {
	seeds := [][]byte{
		[]byte("{\"payload\":\"{\\\"a\\\":1}\"}"),
		[]byte("{\"payload\":\"[1,2,3]\"}"),
		[]byte("{\"payload\":\"{bad}\"}"),
		sampleJSON,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			return
		}

		opts := *DefaultOptions
		opts.Palette = "none"
		opts.Unwrap = true

		var buf bytes.Buffer
		if err := PrettyStream(&buf, bytes.NewReader(data), &opts); err != nil {
			return
		}
		if err := decodeJSONStream(buf.Bytes()); err != nil {
			t.Fatalf("PrettyStream unwrap output invalid JSON: %v", err)
		}
	})
}

func FuzzCompactTo(f *testing.F) {
	seeds := [][]byte{
		[]byte("null"),
		[]byte("true"),
		[]byte("123"),
		[]byte("\"hello\""),
		[]byte("[1,2,3]"),
		[]byte("{\"a\":1,\"b\":[true,false],\"c\":null}"),
		[]byte("  {\"a\":1}  "),
		sampleJSON,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			return
		}

		opts := *DefaultOptions
		opts.Unwrap = false

		var buf bytes.Buffer
		err := CompactTo(&buf, bytes.NewReader(data), &opts)
		if err != nil {
			if _, ok := decodeSingleJSON(data); ok {
				t.Fatalf("CompactTo failed for valid JSON: %v", err)
			}
			return
		}

		if err := validateJSONLines(buf.Bytes()); err != nil {
			t.Fatalf("CompactTo output invalid JSON lines: %v", err)
		}

		if inVal, ok := decodeSingleJSON(data); ok {
			line, ok := firstNonEmptyLine(buf.Bytes())
			if !ok {
				t.Fatalf("expected compact output line for single input")
			}
			outVal, ok := decodeSingleJSON(line)
			if !ok {
				t.Fatalf("expected compact output line to be JSON")
			}
			if !reflect.DeepEqual(inVal, outVal) {
				t.Fatalf("CompactTo output mismatch\ninput: %s\noutput: %s", data, line)
			}
		}
	})
}

func FuzzCompactToUnwrap(f *testing.F) {
	seeds := [][]byte{
		[]byte("{\"payload\":\"{\\\"a\\\":1}\"}"),
		[]byte("{\"payload\":\"[1,2,3]\"}"),
		[]byte("{\"payload\":\"{bad}\"}"),
		sampleJSON,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			return
		}

		opts := *DefaultOptions
		opts.Unwrap = true

		var buf bytes.Buffer
		if err := CompactTo(&buf, bytes.NewReader(data), &opts); err != nil {
			return
		}
		if err := validateJSONLines(buf.Bytes()); err != nil {
			t.Fatalf("CompactTo unwrap output invalid JSON lines: %v", err)
		}
	})
}

func decodeSingleJSON(data []byte) (any, bool) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	return v, true
}

func decodeJSONStream(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	for {
		var v any
		if err := dec.Decode(&v); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func validateJSONLines(data []byte) error {
	lines := bytes.Split(data, []byte{'\n'})
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		if _, ok := decodeSingleJSON(line); !ok {
			return fmt.Errorf("invalid json line %d", i)
		}
	}
	return nil
}

func firstNonEmptyLine(data []byte) ([]byte, bool) {
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		return line, true
	}
	return nil, false
}

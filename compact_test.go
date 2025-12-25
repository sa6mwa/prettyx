package prettyx

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompact_UnwrapDisabled_MultiDoc(t *testing.T) {
	input := strings.NewReader("{\"a\": 1}\n{\"b\": [1, 2,3]}\n\"str\"\nnull\n")

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, DefaultOptions); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"a\":1}\n{\"b\":[1,2,3]}\n\"str\"\nnull\n"
	if buf.String() != expected {
		t.Fatalf("unexpected compact output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_Unwrap_RewritesStrings(t *testing.T) {
	input := strings.NewReader("{\"payload\":\"{\\\"a\\\":1,\\\"b\\\":[2,3]}\",\"raw\":\"hi\"}\n")
	opts := *DefaultOptions
	opts.Unwrap = true

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, &opts); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"payload\":{\"a\":1,\"b\":[2,3]},\"raw\":\"hi\"}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected unwrap output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_Unwrap_MultiDoc(t *testing.T) {
	input := strings.NewReader("\"{\\\"a\\\":1}\"\n{\"b\":\"[1,2]\"}\n")
	opts := *DefaultOptions
	opts.Unwrap = true

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, &opts); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"a\":1}\n{\"b\":[1,2]}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected multi-doc output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_UnwrapDepthOnce(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 0

	input := strings.NewReader("{\"payload\":\"{\\\"inner\\\":\\\"{\\\\\\\"x\\\\\\\":1}\\\"}\"}")
	opts := *DefaultOptions
	opts.Unwrap = true

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, &opts); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"payload\":{\"inner\":\"{\\\"x\\\":1}\"}}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected depth output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_UnwrapSkipsKeys(t *testing.T) {
	input := strings.NewReader("{\"{\\\"a\\\":1}\":\"{\\\"b\\\":2}\"}")
	opts := *DefaultOptions
	opts.Unwrap = true

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, &opts); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"{\\\"a\\\":1}\":{\"b\":2}}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected key output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_UnwrapInvalidJSONLeavesString(t *testing.T) {
	input := strings.NewReader("{\"payload\":\"{bad}\"}")
	opts := *DefaultOptions
	opts.Unwrap = true

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, &opts); err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	const expected = "{\"payload\":\"{bad}\"}\n"
	if buf.String() != expected {
		t.Fatalf("unexpected invalid output\nexpected:\n%q\nactual:\n%q", expected, buf.String())
	}
}

func TestCompact_InvalidJSONReturnsError(t *testing.T) {
	input := strings.NewReader("{\"a\":")

	var buf bytes.Buffer
	if err := CompactTo(&buf, input, DefaultOptions); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

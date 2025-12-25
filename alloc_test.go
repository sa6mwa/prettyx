package prettyx

import (
	"bytes"
	"testing"
)

type discardStringByteWriter struct{}

func (discardStringByteWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (discardStringByteWriter) WriteString(s string) (int, error) {
	return len(s), nil
}

func (discardStringByteWriter) WriteByte(_ byte) error {
	return nil
}

func TestPrettyStream_NoAlloc_Default(t *testing.T) {
	warmPools()

	input := []byte(`{"a":1,"b":[1,2,3],"c":{"d":"e"}}`)
	opts := *DefaultOptions
	opts.Unwrap = false
	opts.SemiCompact = false
	opts.Palette = "none"

	writer := discardStringByteWriter{}
	reader := bytes.NewReader(input)

	allocs := testing.AllocsPerRun(100, func() {
		reader.Reset(input)
		if err := PrettyStream(writer, reader, &opts); err != nil {
			t.Fatalf("PrettyStream failed: %v", err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected zero allocations, got %.2f", allocs)
	}
}

func TestPrettyStream_NoAlloc_SemiCompact(t *testing.T) {
	warmPools()

	input := []byte(`{"a":1,"b":[1,2,3],"c":{"d":"e"}}`)
	opts := *DefaultOptions
	opts.Unwrap = false
	opts.SemiCompact = true
	opts.Palette = "none"

	writer := discardStringByteWriter{}
	reader := bytes.NewReader(input)

	allocs := testing.AllocsPerRun(100, func() {
		reader.Reset(input)
		if err := PrettyStream(writer, reader, &opts); err != nil {
			t.Fatalf("PrettyStream failed: %v", err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected zero allocations, got %.2f", allocs)
	}
}

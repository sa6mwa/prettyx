package prettyx

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

var (
	sampleJSON       = []byte("{\"count\":2,\"message\":\"ok\",\"payload\":\"{\\\"bar\\\":{\\\"list\\\":\\\"[10,20]\\\",\\\"nested\\\":\\\"{\\\\\\\"inner\\\\\\\":true}\\\"},\\\"foo\\\":1}\"}")
	expectedNoColors = `{
  "count": 2,
  "message": "ok",
  "payload": {
    "bar": {
      "list": [
        10,
        20
      ],
      "nested": {
        "inner": true
      }
    },
    "foo": 1
  }
}
`
	expectedSemiCompactNoColors = `{
  "count": 2, "message": "ok", "payload": {
    "bar": {
      "list": [10, 20], "nested": {
        "inner": true
      }
    }, "foo": 1
  }
}
`
)

func TestPretty_UnwrapsNestedJSON(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.Unwrap = true
	optsValue.Palette = "none"

	out, err := Pretty(sampleJSON, &optsValue)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}

	if actual := string(out); actual != expectedNoColors {
		t.Fatalf("unexpected output\nexpected:\n%q\nactual:\n%q", expectedNoColors, actual)
	}

	if strings.ContainsRune(string(out), '\u001b') {
		t.Fatalf("expected ASCII output without color codes, found escape sequence: %q", out)
	}
}

func TestPretty_SemiCompactMatchesTidwallStyle(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.Unwrap = true
	optsValue.SemiCompact = true
	optsValue.Palette = "none"

	out, err := Pretty(sampleJSON, &optsValue)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}

	if actual := string(out); actual != expectedSemiCompactNoColors {
		t.Fatalf("unexpected semi-compact output\nexpected:\n%q\nactual:\n%q", expectedSemiCompactNoColors, actual)
	}
}

func TestPrettyTo_WritesOutput(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.Unwrap = true
	optsValue.Palette = "none"

	var buf bytes.Buffer
	if err := PrettyTo(&buf, sampleJSON, &optsValue); err != nil {
		t.Fatalf("PrettyTo failed: %v", err)
	}

	got := buf.String()
	if got != expectedNoColors {
		t.Fatalf("unexpected writer output\nexpected:\n%q\nactual:\n%q", expectedNoColors, got)
	}

	if strings.ContainsRune(got, '\u001b') {
		t.Fatalf("expected writer output without color codes, found escape sequence: %q", got)
	}
}

func TestPretty_UnwrapDisabledByDefault(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.Unwrap = false
	optsValue.Palette = "none"

	out, err := Pretty(sampleJSON, &optsValue)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}

	const expectedUnwrapDisabled = `{
  "count": 2,
  "message": "ok",
  "payload": "{\"bar\":{\"list\":\"[10,20]\",\"nested\":\"{\\\"inner\\\":true}\"},\"foo\":1}"
}
`

	if string(out) != expectedUnwrapDisabled {
		t.Fatalf("unexpected output with unwrap disabled\nexpected:\n%q\nactual:\n%q", expectedUnwrapDisabled, out)
	}
}

func TestPretty_PaletteNoneDisablesColor(t *testing.T) {
	optsValue := *DefaultOptions
	optsValue.Unwrap = true
	optsValue.Palette = "none"

	out, err := Pretty(sampleJSON, &optsValue)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}

	if string(out) != expectedNoColors {
		t.Fatalf("unexpected output for palette none\nexpected:\n%q\nactual:\n%q", expectedNoColors, out)
	}
	if strings.ContainsRune(string(out), '\u001b') {
		t.Fatalf("expected output without color codes, found escape sequence: %q", out)
	}
}

func TestPretty_UnknownPalette(t *testing.T) {
	optsValue := *DefaultOptions
	optsValue.Palette = "does-not-exist"

	if _, err := Pretty(sampleJSON, &optsValue); err == nil {
		t.Fatalf("expected error for unknown palette")
	}
}

func TestPrettyMatchesJQFromJSON(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed; skipping comparison")
	}

	optsValue := *DefaultOptions
	optsValue.Unwrap = true
	optsValue.Palette = "none"

	prettyOut, err := Pretty(sampleJSON, &optsValue)
	if err != nil {
		t.Fatalf("Pretty failed: %v", err)
	}

	const jqProgram = `def trim($s):
  $s | sub("^\\s+";"") | sub("\\s+$";"");
def looks_like_json($s):
  ($s | length > 1)
  and (
    (($s[0:1] == "{") and ($s[-1:] == "}"))
    or (($s[0:1] == "[") and ($s[-1:] == "]"))
  );
def unwrap:
  if type == "string" then
    (. as $original
     | trim($original) as $trimmed
     | if looks_like_json($trimmed) then
         (try ($trimmed | fromjson | unwrap) catch $original)
       else
         $original
       end)
  elif type == "array" then
    map(unwrap)
  elif type == "object" then
    with_entries(.value |= unwrap)
  else
    .
  end;
unwrap`

	cmd := exec.Command("jq", jqProgram)
	cmd.Stdin = bytes.NewReader(sampleJSON)
	var jqOut bytes.Buffer
	var jqErr bytes.Buffer
	cmd.Stdout = &jqOut
	cmd.Stderr = &jqErr
	if err := cmd.Run(); err != nil {
		t.Logf("warning: jq comparison skipped due to error: %v (stderr: %s)", err, jqErr.String())
		return
	}

	var jqJSON any
	if err := json.Unmarshal(bytes.TrimSpace(jqOut.Bytes()), &jqJSON); err != nil {
		t.Fatalf("failed to unmarshal jq output: %v", err)
	}

	var prettyJSON any
	if err := json.Unmarshal(bytes.TrimSpace(prettyOut), &prettyJSON); err != nil {
		t.Fatalf("failed to unmarshal prettyx output: %v", err)
	}

	if !reflect.DeepEqual(jqJSON, prettyJSON) {
		t.Errorf("warning: prettyx JSON differs from jq recursive unwrap\nprettyx: %q\njq: %q", string(prettyOut), jqOut.String())
	}
}

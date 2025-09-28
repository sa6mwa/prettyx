package prettyx

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	sampleJSON       = []byte("{\"count\":2,\"message\":\"ok\",\"payload\":\"{\\\"bar\\\":{\\\"list\\\":\\\"[10,20]\\\",\\\"nested\\\":\\\"{\\\\\\\"inner\\\\\\\":true}\\\"},\\\"foo\\\":1}\"}")
	expectedNoColors = `{
  "count": 2,
  "message": "ok",
  "payload": {
    "bar": {
      "list": [10, 20],
      "nested": {
        "inner": true
      }
    },
    "foo": 1
  }
}
`
)

func TestPrettyWithRenderer_UnwrapsNestedJSON(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.SortKeys = true
	optsValue.NoUnwrap = false

	var sink bytes.Buffer
	renderer := lipgloss.NewRenderer(&sink)
	renderer.SetColorProfile(termenv.Ascii)

	out, err := PrettyWithRenderer(sampleJSON, &optsValue, renderer, nil)
	if err != nil {
		t.Fatalf("PrettyWithRenderer failed: %v", err)
	}

	if actual := string(out); actual != expectedNoColors {
		t.Fatalf("unexpected output\nexpected:\n%q\nactual:\n%q", expectedNoColors, actual)
	}

	if strings.ContainsRune(string(out), '\u001b') {
		t.Fatalf("expected ASCII output without color codes, found escape sequence: %q", out)
	}
}

func TestPrettyTo_WritesOutput(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.SortKeys = true
	optsValue.NoUnwrap = false

	var buf bytes.Buffer
	if err := PrettyTo(&buf, sampleJSON, &optsValue, nil); err != nil {
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

func TestPrettyWithRenderer_NoUnwrapDisablesUnwrap(t *testing.T) {
	t.Cleanup(func() { MaxNestedJSONDepth = 10 })
	MaxNestedJSONDepth = 4

	optsValue := *DefaultOptions
	optsValue.SortKeys = true
	optsValue.NoUnwrap = true

	var sink bytes.Buffer
	renderer := lipgloss.NewRenderer(&sink)
	renderer.SetColorProfile(termenv.Ascii)

	out, err := PrettyWithRenderer(sampleJSON, &optsValue, renderer, nil)
	if err != nil {
		t.Fatalf("PrettyWithRenderer failed: %v", err)
	}

	const expectedNoUnwrap = `{
  "count": 2,
  "message": "ok",
  "payload": "{\"bar\":{\"list\":\"[10,20]\",\"nested\":\"{\\\"inner\\\":true}\"},\"foo\":1}"
}
`

	if string(out) != expectedNoUnwrap {
		t.Fatalf("unexpected output for no-unwrap\nexpected:\n%q\nactual:\n%q", expectedNoUnwrap, out)
	}
}

func TestPrettyMatchesJQFromJSON(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed; skipping comparison")
	}

	optsValue := *DefaultOptions
	optsValue.SortKeys = true
	optsValue.NoUnwrap = false

	var sink bytes.Buffer
	renderer := lipgloss.NewRenderer(&sink)
	renderer.SetColorProfile(termenv.Ascii)

	prettyOut, err := PrettyWithRenderer(sampleJSON, &optsValue, renderer, nil)
	if err != nil {
		t.Fatalf("PrettyWithRenderer failed: %v", err)
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

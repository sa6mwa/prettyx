# prettyx

prettyx formats JSON with deterministic indentation and optional syntax highlighting. It also unwraps JSON values that are themselves encoded as JSON strings, so nested JSON becomes readable without manual decoding (this behaviour is similar to how the `jq` tool renders JSON with the `fromjson` builtin).

## Install the CLI

```
go install github.com/sa6mwa/prettyx/cmd/prettyx@latest
```

Run `prettyx` with one or more JSON files (use `-` for stdin). Add `-no-color` to force plain output. Use `-no-unwrap` to skip decoding JSON appearing inside string values.

By default prettyx unwraps JSON strings recursively so nested documents become objects or arrays. This differs from `jq`, which leaves embedded JSON as strings unless you explicitly call `fromjson` (for example: `jq '.payload |= fromjson'`).

```
prettyx payload.json other.json
cat payload.json | prettyx -no-color -
```

## jq equivalent

While not recursive, this example renders identically with `jq` and `prettyx`:

```console
$ echo '{"value":"[{"inner": true},{"inner": true,"msg":"true aswell"}]"}' | jq '.value |= fromjson'
{
  "value": [
    {
      "inner": true
    },
    {
      "inner": true,
      "msg": "true aswell"
    }
  ]
}
```

## Use as a library

Import the package and call `Pretty` (or `PrettyWithRenderer` if you need custom lipgloss styling).

```go
package main

import (
    "fmt"
    "log"

    "github.com/sa6mwa/prettyx"
)

func main() {
    src := []byte(`{"foo":"{"nested":true}"}`)
    pretty, err := prettyx.Pretty(src, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(string(pretty))
}
```

`Pretty` respects the configurable `prettyx.MaxNestedJSONDepth`, and you can pass custom `Options` to tweak width, indentation, key sorting, and `NoUnwrap` when you need the jq-style behaviour of keeping embedded JSON strings untouched.

## Designed for recursive jq-style unwrapping

prettyx was designed to replicate the behaviour of the following jq program, which repeatedly calls `fromjson` on any string that looks like JSON and walks the entire structure:

```
jq 'def trim($s):
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
unwrap'
```

Example comparison with sample data:

```console
$ cat sample.json
{"count":2,"message":"ok","payload":"{\"bar\":{\"list\":\"[10,20]\",\"nested\":\"{\\\"inner\\\":true}\"},\"foo\":1}"}

$ jq 'def trim($s):
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
unwrap' sample.json
{
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

$ prettyx sample.json
{
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
```

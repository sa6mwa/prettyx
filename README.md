# prettyx

prettyx formats JSON with deterministic indentation and optional syntax highlighting. The default output is jq-ish (one key/value per line). It can also unwrap JSON values that are themselves encoded as JSON strings, so nested JSON becomes readable without manual decoding (this behaviour is similar to how the `jq` tool renders JSON with the `fromjson` builtin).

## Install the CLI

```
go install pkt.systems/prettyx/cmd/prettyx@latest
```

## Build from source

```
make            # builds bin/prettyx
make test       # run tests with -race
make fuzz       # run fuzzers (10s per fuzzer)
make bench      # run benchmarks
make install    # installs to /usr/local/bin by default
```

Use `PREFIX=/path` (and optionally `DESTDIR=/path`) to change the install location.

## Usage

Run `prettyx` with one or more JSON files (use `-` for stdin). Add `--no-color` (or `--palette none`) to force plain output, or `-C`/`--color-force` to force color on non-TTY output. Use `--palette <name>` to pick from the bundled themes (see `--list-palettes`). The default palette matches jq’s built-in colours. Use `-u`/`--unwrap` to decode JSON appearing inside string values. Use `--semi-compact` for tidwall-style semi-compact formatting with soft wrapping (`-w`/`--width` controls the wrap width). Use `-c`/`--compact` to emit one compacted JSON document per line. When reading from URLs, use `-k`/`--insecure` to skip TLS verification and `--accept-all` to send `Accept: */*`.

By default prettyx leaves JSON strings untouched, matching `jq`'s default behaviour. Both require an explicit `fromjson` (for example: `jq '.payload |= fromjson'`) or `--unwrap` to recursively decode JSON-looking strings.

prettyx originally borrowed the tidwall/pretty output style. The current formatter is a fully rewritten zero-alloc streaming implementation, and the old layout is now available via `--semi-compact`.

```
prettyx payload.json other.json
prettyx -u payload.json
prettyx --unwrap payload.json
prettyx --semi-compact payload.json
prettyx --semi-compact -w 120 payload.json
prettyx -c payload.json
prettyx https://example.com/data.json
prettyx --accept-all https://example.com/data
cat payload.json | prettyx --no-color
cat payload.json | prettyx -C | less -R
cat payload.json | prettyx --palette tokyo-night
prettyx --list-palettes

Bundled palettes: default/jq (jq colour scheme), catppuccin-mocha, doom-dracula, doom-gruvbox, doom-iosvkem, doom-nord, gruvbox-light, monokai-vibrant, one-dark-aurora, outrun-electric, solarized-nightfall, synthwave84, tokyo-night, pslog (classic pslog default), and none.
```

## jq equivalent

With `--unwrap`, this example renders identically with `jq` and `prettyx`:

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

Import the package and call `Pretty`.

```go
package main

import (
    "fmt"
    "log"

    "pkt.systems/prettyx"
)

func main() {
    src := []byte(`{"foo":"{\"nested\":true}"}`)
    pretty, err := prettyx.Pretty(src, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(string(pretty))
}
```

`Pretty` respects the configurable `prettyx.MaxNestedJSONDepth`, and you can pass custom `Options` to tweak width (when `SemiCompact` is enabled), indentation, and `Unwrap` when you want the jq-style behaviour of decoding embedded JSON strings.

### Allocations and streaming

prettyx focuses on streaming performance and minimizing heap allocations. In this context, “zero-alloc” means no heap allocations per document in steady state (after pools are warmed).

- `PrettyStream` and `PrettyReader` are the zero-alloc paths when `Unwrap` is false.
- `CompactTo` is likewise allocation-free in steady state for non-unwrap input.
- `Pretty` and `PrettyToBuffer` allocate because they build an in-memory buffer for the output.
- `CompactToBuffer` allocates for its output buffer.
- When `Unwrap` is true, additional work is required to decode escaped strings and re-parse embedded JSON; the streaming path reuses internal buffers, but allocations can still occur depending on input.

```go
// Stream from a reader into a writer.
if err := prettyx.PrettyStream(os.Stdout, os.Stdin, nil); err != nil {
    log.Fatal(err)
}

// Pipe-style usage (close to stop the goroutine).
r := prettyx.PrettyReader(os.Stdin, nil)
defer r.Close()
if _, err := io.Copy(os.Stdout, r); err != nil {
    log.Fatal(err)
}
```

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

$ prettyx -u sample.json
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
```

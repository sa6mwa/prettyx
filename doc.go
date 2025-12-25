// Package prettyx provides streaming JSON pretty printing with optional ANSI
// coloring and recursive unwrapping of JSON-looking strings.
//
// By default, output is jq-ish (one element or key per line). Set
// Options.SemiCompact to enable tidwall-style formatting with soft wrapping.
// Options.Width controls the wrap column in SemiCompact mode; when Width <= 0,
// output always breaks after commas. For maximum throughput, PrettyStream and
// PrettyReader operate without allocations when Options.Unwrap is false.
//
// Basic usage:
//
//	src := []byte(`{"foo":"{\"nested\":true}"}`)
//	opts := &prettyx.Options{Unwrap: true, Palette: "none"}
//	out, err := prettyx.Pretty(src, opts)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Print(string(out))
//
// Streaming:
//
//	opts := &prettyx.Options{Palette: "none"}
//	if err := prettyx.PrettyStream(os.Stdout, os.Stdin, opts); err != nil {
//		log.Fatal(err)
//	}
//
// Semi-compact formatting:
//
//	opts := &prettyx.Options{SemiCompact: true, Width: 80, Palette: "none"}
//	if err := prettyx.PrettyStream(os.Stdout, bytes.NewReader(src), opts); err != nil {
//		log.Fatal(err)
//	}
package prettyx

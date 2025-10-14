package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"pkt.systems/prettyx"
)

func main() {
	var forceColor bool
	flag.BoolVar(&forceColor, "C", false, "force colorized output even when writing to a non-TTY")
	flag.BoolVar(&forceColor, "color-force", false, "force colorized output even when writing to a non-TTY")
	noColor := flag.Bool("no-color", false, "disable colorized output, even when writing to a TTY")
	noUnwrap := flag.Bool("no-unwrap", false, "preserve JSON-looking strings without decoding")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [file...]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Reads from stdin when no files are provided.")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"-"}
	}
	renderer := prettyx.NewRenderer(os.Stdout, forceColor)
	var palette *prettyx.ColorPalette
	if *noColor {
		nc := prettyx.NoColorPalette(renderer)
		palette = &nc
	}
	opts := *prettyx.DefaultOptions
	if *noUnwrap {
		opts.NoUnwrap = true
	}
	if forceColor {
		opts.ForceColor = true
	}
	for _, path := range args {
		if err := streamJSON(path, renderer, palette, &opts); err != nil {
			fmt.Fprintf(os.Stderr, "prettyjson: %v\n", err)
			os.Exit(1)
		}
	}
}

func streamJSON(path string, renderer *lipgloss.Renderer, palette *prettyx.ColorPalette, opts *prettyx.Options) error {
	reader, closer, err := openInput(path)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}
	dec := json.NewDecoder(reader)
	dec.UseNumber()
	source := path
	if path == "-" {
		source = "<stdin>"
	}
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("%s: %w", source, err)
		}
		out, err := prettyx.PrettyWithRenderer(raw, opts, renderer, palette)
		if err != nil {
			return fmt.Errorf("%s: %w", source, err)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			return fmt.Errorf("write error: %w", err)
		}
	}
}

func openInput(path string) (io.Reader, io.Closer, error) {
	if path == "-" {
		return os.Stdin, nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	return file, file, nil
}

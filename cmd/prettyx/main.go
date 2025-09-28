package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/sa6mwa/prettyx"
)

func main() {
	noColor := flag.Bool("no-color", false, "disable colorized output, even when writing to a TTY")
	noUnwrap := flag.Bool("no-unwrap", false, "preserve JSON-looking strings without decoding")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] file [file...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	renderer := lipgloss.NewRenderer(os.Stdout)
	var palette *prettyx.ColorPalette
	if *noColor {
		nc := prettyx.NoColorPalette(renderer)
		palette = &nc
	}
	opts := *prettyx.DefaultOptions
	if *noUnwrap {
		opts.NoUnwrap = true
	}
	for _, path := range flag.Args() {
		data, err := readInput(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "prettyjson: %s\n", err)
			os.Exit(1)
		}
		out, err := prettyx.PrettyWithRenderer(data, &opts, renderer, palette)
		if err != nil {
			fmt.Fprintf(os.Stderr, "prettyjson: %s: %v\n", path, err)
			os.Exit(1)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			fmt.Fprintf(os.Stderr, "prettyjson: write error: %v\n", err)
			os.Exit(1)
		}
	}
}

func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return data, nil
}

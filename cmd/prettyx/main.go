package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"pkt.systems/prettyx"
)

func main() {
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var forceColor bool
	flags.BoolVarP(&forceColor, "color-force", "C", false, "force colorized output even when writing to a non-TTY")
	noColor := flags.Bool("no-color", false, "disable colorized output, even when writing to a TTY")
	unwrap := flags.BoolP("unwrap", "u", false, "decode JSON-looking strings recursively")
	compact := flags.BoolP("compact", "c", false, "compact output (one document per line, no color)")
	semiCompact := flags.Bool("semi-compact", false, "use tidwall-style semi-compact formatting (soft wraps to --width)")
	width := flags.IntP("width", "w", prettyx.DefaultOptions.Width, "soft wrap width for --semi-compact (<= 0 always wraps)")
	insecure := flags.BoolP("insecure", "k", false, "allow insecure HTTPS connections for URL inputs (skip TLS verification)")
	acceptAll := flags.Bool("accept-all", false, "send Accept: */* when fetching URLs (default sends JSON-focused Accept header)")
	paletteName := flags.String("palette", "default", "palette name (use --list-palettes to see options)")
	listPalettes := flags.Bool("list-palettes", false, "list available palette names and exit")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [flags] [file_or_url...]\n", os.Args[0])
		fmt.Fprintln(flags.Output(), "Reads from stdin when no files are provided.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "prettyx: %v\n", err)
		os.Exit(2)
	}

	if *listPalettes {
		for _, name := range prettyx.PaletteNames() {
			fmt.Fprintln(os.Stdout, name)
		}
		return
	}

	args := flags.Args()
	if len(args) == 0 {
		args = []string{"-"}
	}
	opts := *prettyx.DefaultOptions
	opts.Unwrap = *unwrap
	if forceColor {
		opts.ForceColor = true
	}
	opts.Palette = *paletteName
	if *noColor {
		opts.Palette = "none"
	}
	opts.SemiCompact = *semiCompact
	opts.Width = *width
	urlOpts := urlOptions{
		insecure:  *insecure,
		acceptAll: *acceptAll,
	}
	for _, path := range args {
		var err error
		if *compact {
			err = streamCompact(path, &opts, urlOpts)
		} else {
			err = streamPretty(path, &opts, urlOpts)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "prettyx: %v\n", err)
			os.Exit(1)
		}
	}
}

func streamPretty(path string, opts *prettyx.Options, urlOpts urlOptions) error {
	reader, closer, err := openInput(path, urlOpts)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}
	source := path
	if path == "-" {
		source = "<stdin>"
	}
	if err := prettyx.PrettyStream(os.Stdout, reader, opts); err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	return nil
}

func streamCompact(path string, opts *prettyx.Options, urlOpts urlOptions) error {
	reader, closer, err := openInput(path, urlOpts)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}
	if err := prettyx.CompactTo(os.Stdout, reader, opts); err != nil {
		source := path
		if path == "-" {
			source = "<stdin>"
		}
		return fmt.Errorf("%s: %w", source, err)
	}
	return nil
}

const defaultAcceptHeader = "application/json, application/*+json, text/json, application/x-ndjson"

type urlOptions struct {
	insecure  bool
	acceptAll bool
}

func openInput(path string, urlOpts urlOptions) (io.Reader, io.Closer, error) {
	if path == "-" {
		return os.Stdin, nil, nil
	}
	if parsedURL, isURL, err := parseHTTPURL(path); isURL {
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		reader, closer, err := openURL(parsedURL, urlOpts)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		return reader, closer, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	return file, file, nil
}

func parseHTTPURL(rawURL string) (*url.URL, bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, false, err
	}
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, false, nil
	}
	if parsedURL.Host == "" {
		return nil, true, fmt.Errorf("invalid URL: missing host")
	}
	return parsedURL, true, nil
}

func openURL(parsedURL *url.URL, opts urlOptions) (io.Reader, io.Closer, error) {
	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	if opts.acceptAll {
		req.Header.Set("Accept", "*/*")
	} else {
		req.Header.Set("Accept", defaultAcceptHeader)
	}
	client := &http.Client{}
	if opts.insecure && parsedURL.Scheme == "https" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		// InsecureSkipVerify is opt-in via -k for HTTPS URL inputs.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = transport
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices-1 {
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return resp.Body, resp.Body, nil
}

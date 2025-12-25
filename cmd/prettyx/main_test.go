package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenURLAcceptHeaderDefault(t *testing.T) {
	t.Parallel()

	acceptCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptCh <- r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	parsedURL, isURL, err := parseHTTPURL(server.URL)
	if err != nil {
		t.Fatalf("parseHTTPURL error: %v", err)
	}
	if !isURL {
		t.Fatalf("expected URL to be detected")
	}

	reader, closer, err := openURL(parsedURL, urlOptions{})
	if err != nil {
		t.Fatalf("openURL error: %v", err)
	}
	defer closer.Close()

	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("read response: %v", err)
	}

	select {
	case got := <-acceptCh:
		if got != defaultAcceptHeader {
			t.Fatalf("unexpected Accept header: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for Accept header")
	}
}

func TestOpenURLAcceptHeaderAll(t *testing.T) {
	t.Parallel()

	acceptCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptCh <- r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	parsedURL, isURL, err := parseHTTPURL(server.URL)
	if err != nil {
		t.Fatalf("parseHTTPURL error: %v", err)
	}
	if !isURL {
		t.Fatalf("expected URL to be detected")
	}

	reader, closer, err := openURL(parsedURL, urlOptions{acceptAll: true})
	if err != nil {
		t.Fatalf("openURL error: %v", err)
	}
	defer closer.Close()

	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("read response: %v", err)
	}

	select {
	case got := <-acceptCh:
		if got != "*/*" {
			t.Fatalf("unexpected Accept header: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for Accept header")
	}
}

func TestOpenURLInsecureHTTPS(t *testing.T) {
	t.Parallel()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()

	parsedURL, isURL, err := parseHTTPURL(server.URL)
	if err != nil {
		t.Fatalf("parseHTTPURL error: %v", err)
	}
	if !isURL {
		t.Fatalf("expected URL to be detected")
	}

	if _, _, err := openURL(parsedURL, urlOptions{}); err == nil {
		t.Fatalf("expected TLS error without -k")
	}

	reader, closer, err := openURL(parsedURL, urlOptions{insecure: true})
	if err != nil {
		t.Fatalf("openURL insecure error: %v", err)
	}
	defer closer.Close()

	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("read response: %v", err)
	}
}

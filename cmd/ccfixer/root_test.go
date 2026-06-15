package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestProxyRewritesSystemMessage(t *testing.T) {
	var got []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: ok\n\n")
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	front := httptest.NewServer(newProxy(target, &options{}, io.Discard))
	defer front.Close()

	body := `{"messages":[{"role":"user","content":"hi"},{"role":"system","content":"x"}]}`
	resp, err := http.Post(front.URL+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if string(respBody) != "data: ok\n\n" {
		t.Fatalf("response not passed through: %q", respBody)
	}
	if bytes.Contains(got, []byte(`"role":"system"`)) {
		t.Fatalf("upstream still received a system role: %s", got)
	}
	if !bytes.Contains(got, []byte("<system-reminder>x</system-reminder>")) {
		t.Fatalf("upstream body missing merged reminder: %s", got)
	}
}

func TestProxyPassesThroughNonMessagesPath(t *testing.T) {
	var got []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	front := httptest.NewServer(newProxy(target, &options{}, io.Discard))
	defer front.Close()

	body := `{"messages":[{"role":"system","content":"x"}]}`
	resp, err := http.Post(front.URL+"/v1/models", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if string(got) != body {
		t.Fatalf("non-messages path should pass body through unchanged, got: %s", got)
	}
}

func TestResolveBaseURL(t *testing.T) {
	cases := []struct {
		name       string
		listenAddr string
		port       int
		want       string
	}{
		{"empty host", ":0", 54321, "http://127.0.0.1:54321"},
		{"loopback", "127.0.0.1:0", 8080, "http://127.0.0.1:8080"},
		{"unspecified v4", "0.0.0.0:0", 9000, "http://127.0.0.1:9000"},
		{"unspecified v6", "[::]:0", 9100, "http://127.0.0.1:9100"},
		{"hostname", "localhost:0", 7000, "http://localhost:7000"},
		{"concrete ip", "192.168.1.100:0", 9200, "http://192.168.1.100:9200"},
		{"ipv6 loopback", "[::1]:0", 9300, "http://[::1]:9300"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveBaseURL(tc.listenAddr, tc.port)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveBaseURL(%q, %d) = %q, want %q", tc.listenAddr, tc.port, got, tc.want)
			}
		})
	}
}

func TestResolveBaseURLInvalid(t *testing.T) {
	if _, err := resolveBaseURL("bogus", 1234); err == nil {
		t.Fatal("want error for invalid listen address, got nil")
	}
}

func TestResolveBaseURLWithRealRandomPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if port <= 0 {
		t.Fatalf("expected a positive bound port, got %d", port)
	}
	got, err := resolveBaseURL("127.0.0.1:0", port)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://127.0.0.1:" + strconv.Itoa(port)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRunRootInvalidListen(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"-u", "https://example.com", "-l", "bogus"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Fatal("want error for invalid listen address, got nil")
	}
}

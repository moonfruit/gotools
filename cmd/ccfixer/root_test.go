package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

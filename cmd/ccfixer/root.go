package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moonfruit/gotools/internal/ccfixer"
)

type options struct {
	listen   string
	upstream string
	verbose  bool
}

func newRootCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "ccfixer",
		Short: "Reverse proxy that rewrites Claude Code's mid-conversation system messages",
		Long: `ccfixer is a transparent reverse proxy for Claude Code's Anthropic Messages
API traffic. Some upstream relays reject the mid-conversation role:"system"
messages that newer Claude Code emits. ccfixer merges each such message into an
adjacent user message (wrapped in <system-reminder> tags) before forwarding, and
passes everything else — headers, paths, and streaming responses — through
unchanged.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRoot(cmd, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.listen, "listen", "l", "127.0.0.1:8787", "listen address")
	cmd.Flags().StringVarP(&opts.upstream, "upstream", "u", "", "upstream base URL (required), e.g. https://relay.example.com")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "log how many system messages were merged per request")
	_ = cmd.MarkFlagRequired("upstream")
	return cmd
}

func runRoot(cmd *cobra.Command, opts *options) error {
	target, err := url.Parse(opts.upstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("upstream URL must include scheme and host, got %q", opts.upstream)
	}

	proxy := newProxy(target, opts, cmd.ErrOrStderr())
	fmt.Fprintf(cmd.OutOrStdout(), "ccfixer listening on %s, forwarding to %s\n", opts.listen, opts.upstream)

	server := &http.Server{Addr: opts.listen, Handler: proxy}
	return server.ListenAndServe()
}

// resolveBaseURL builds the base URL clients should use to reach the proxy,
// given the listen address the user requested and the actual bound port. An
// empty or unspecified host (e.g. ":0", "0.0.0.0", "::") is reported as
// 127.0.0.1 so the URL is directly usable.
func resolveBaseURL(listenAddr string, port int) (string, error) {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", listenAddr, err)
	}
	if host == "" {
		host = "127.0.0.1"
	} else if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)), nil
}

// newProxy builds a reverse proxy to target that rewrites qualifying request
// bodies via rewriteRequest before forwarding. It uses the modern Rewrite hook
// (Director is deprecated as of Go 1.26).
func newProxy(target *url.URL, opts *options, logw io.Writer) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)         // route to target scheme/host, joining the inbound path
			pr.Out.Host = target.Host // send the upstream host header
			rewriteRequest(pr.Out, opts, logw)
		},
	}
}

// rewriteRequest rewrites the body of POST requests whose path contains
// "/v1/messages". All other requests are left untouched. Fail-open: on any read
// problem the request is forwarded as-is.
func rewriteRequest(req *http.Request, opts *options, logw io.Writer) {
	if req.Method != http.MethodPost || req.Body == nil {
		return
	}
	if !strings.Contains(req.URL.Path, "/v1/messages") {
		return
	}
	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return
	}
	out, n, _ := ccfixer.Transform(body)
	req.Body = io.NopCloser(bytes.NewReader(out))
	req.ContentLength = int64(len(out))
	if opts.verbose && n > 0 {
		fmt.Fprintf(logw, "ccfixer: merged %d system message(s)\n", n)
	}
}

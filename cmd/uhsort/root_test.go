package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCmd(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(stdin))
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestStdinToStdout(t *testing.T) {
	stdin := "bob@example.com\nalice@1.1.1.1\nadmin@[::1]:22\n"
	out, _, err := runCmd(t, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "alice@1.1.1.1\nadmin@[::1]:22\nbob@example.com\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestPortParticipates(t *testing.T) {
	stdin := "a@h.com:80\nb@h.com\na@h.com:22\n"
	out, _, err := runCmd(t, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// host=h.com common; sort by port (0 < 22 < 80), then user.
	want := "b@h.com\na@h.com:22\na@h.com:80\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestCount(t *testing.T) {
	stdin := "a@h\na@h\nb@h\n"
	out, _, err := runCmd(t, stdin, "-c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "2\ta@h\n1\tb@h\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestUnique(t *testing.T) {
	stdin := "a@h\na@h\nb@h\n"
	out, _, err := runCmd(t, stdin, "-u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "a@h\nb@h\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestFileInputToStdout(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(in, []byte("b@x\na@x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCmd(t, "", in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "a@x\nb@x\n" {
		t.Errorf("got %q", out)
	}
}

func TestOutputFlag(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.txt")
	stdoutStr, _, err := runCmd(t, "b@x\na@x\n", "-o", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdoutStr != "" {
		t.Errorf("expected empty stdout, got %q", stdoutStr)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a@x\nb@x\n" {
		t.Errorf("output file = %q", string(got))
	}
}

func TestInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(path, []byte("b@x\na@x\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCmd(t, "", "-i", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a@x\nb@x\n" {
		t.Errorf("file content = %q", string(got))
	}
	// Mode should be preserved.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("mode = %o, want 0600", mode)
	}
}

func TestInPlaceRequiresFile(t *testing.T) {
	_, _, err := runCmd(t, "a@x\n", "-i")
	if err == nil {
		t.Fatal("expected error when -i is used without a positional file")
	}
	if !strings.Contains(err.Error(), "in-place") {
		t.Errorf("error should mention in-place: %v", err)
	}
}

func TestInPlaceAndOutputMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(in, []byte("a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.txt")
	_, _, err := runCmd(t, "", "-i", "-o", out, in)
	if err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
}

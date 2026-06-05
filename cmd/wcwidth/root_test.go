package main

import (
	"bytes"
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

func TestArgsEachOnOwnLine(t *testing.T) {
	out, _, err := runCmd(t, "", "你好", "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "4\n3\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestStdinLineByLine(t *testing.T) {
	stdin := "abc\n你好\n\n"
	out, _, err := runCmd(t, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 第三行为空行 => 0
	want := "3\n4\n0\n"
	if out != want {
		t.Errorf("\n got = %q\nwant = %q", out, want)
	}
}

func TestEastAsianFlag(t *testing.T) {
	out, _, err := runCmd(t, "", "-E", "±")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "2\n" {
		t.Errorf("got %q, want %q", out, "2\n")
	}
}

func TestNarrowFlag(t *testing.T) {
	out, _, err := runCmd(t, "", "-N", "±")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "1\n" {
		t.Errorf("got %q, want %q", out, "1\n")
	}
}

func TestEastAsianAndNarrowMutuallyExclusive(t *testing.T) {
	_, _, err := runCmd(t, "", "-E", "-N", "±")
	if err == nil {
		t.Fatal("expected error when -E and -N are combined, got nil")
	}
}

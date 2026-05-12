package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moonfruit/gotools/internal/uhsort"
)

type options struct {
	inPlace bool
	output  string
	unique  bool
	count   bool
}

func newRootCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "uhsort [file]",
		Short: "Sort user@host[:port] lists",
		Long: `Sort lines of the form "user@host[:port][ rest]".

Sort key precedence: host (IPv4 < IPv6 < domain), then port (0 = no port),
then user, then the remaining content. Lines without "user@" are treated as
host-only. Optionally deduplicate equivalent lines, or prefix with counts.

Reads stdin by default; pass a positional FILE to read from a file. Writes
stdout by default; use -o for a file, or -i to edit the input file in place.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.inPlace, "in-place", "i", false, "edit the input file in place (requires a positional file)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "write output to FILE (default: stdout)")
	cmd.Flags().BoolVarP(&opts.unique, "unique", "u", false, "deduplicate equivalent lines after sorting")
	cmd.Flags().BoolVarP(&opts.count, "count", "c", false, "deduplicate and prefix each line with its count (implies -u)")
	cmd.MarkFlagsMutuallyExclusive("in-place", "output")
	return cmd
}

func runRoot(cmd *cobra.Command, args []string, opts *options) error {
	var inputPath string
	if len(args) == 1 {
		inputPath = args[0]
	}
	if opts.inPlace && inputPath == "" {
		return errors.New("-i/--in-place requires a positional input file")
	}

	data, err := readInput(cmd, inputPath)
	if err != nil {
		return err
	}

	lines, err := scanLines(data)
	if err != nil {
		return err
	}

	out := process(lines, opts)

	return writeOutput(cmd, inputPath, opts, out)
}

func readInput(cmd *cobra.Command, inputPath string) ([]byte, error) {
	if inputPath != "" {
		b, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", inputPath, err)
		}
		return b, nil
	}
	b, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	return b, nil
}

func scanLines(data []byte) ([]uhsort.Line, error) {
	var lines []uhsort.Line
	s := bufio.NewScanner(strings.NewReader(string(data)))
	s.Buffer(make([]byte, 64*1024), 1<<30)
	for s.Scan() {
		lines = append(lines, uhsort.Parse(s.Text()))
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan input: %w", err)
	}
	return lines, nil
}

func process(lines []uhsort.Line, opts *options) []byte {
	var buf strings.Builder
	switch {
	case opts.count:
		for _, c := range uhsort.Count(lines) {
			fmt.Fprintf(&buf, "%d\t%s\n", c.N, c.Line.Raw)
		}
	case opts.unique:
		for _, l := range uhsort.Dedupe(lines) {
			buf.WriteString(l.Raw)
			buf.WriteByte('\n')
		}
	default:
		uhsort.Sort(lines)
		for _, l := range lines {
			buf.WriteString(l.Raw)
			buf.WriteByte('\n')
		}
	}
	return []byte(buf.String())
}

func writeOutput(cmd *cobra.Command, inputPath string, opts *options, data []byte) error {
	switch {
	case opts.inPlace:
		return writeInPlace(inputPath, data)
	case opts.output != "":
		if err := os.WriteFile(opts.output, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", opts.output, err)
		}
		return nil
	default:
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
}

// writeInPlace writes data to a sibling temp file then atomically renames over path.
// Preserves the original file's mode if available.
func writeInPlace(path string, data []byte) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	f, err := os.CreateTemp(dir, ".uhsort-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if info, err := os.Stat(abs); err == nil {
		_ = os.Chmod(tmp, info.Mode())
	}
	if err := os.Rename(tmp, abs); err != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tmp, abs, err)
	}
	return nil
}

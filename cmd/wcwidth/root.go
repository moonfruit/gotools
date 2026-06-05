package main

import (
	"bufio"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/moonfruit/gotools/internal/wcwidth"
)

type options struct {
	eastAsian bool
	narrow    bool
}

func newRootCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "wcwidth [string ...]",
		Short: "Print the terminal display width of UTF-8 text",
		Long: `Print the terminal display width of UTF-8 text.

Half-width characters (ASCII) count as 1, CJK and full-width characters as 2,
and combining marks as 0.

With positional arguments, print the width of each argument on its own line.
With no arguments, read stdin and print the width of each line (a trailing
newline is not counted; an empty line yields 0).

East Asian Ambiguous-width characters follow the runtime environment by
default; use -E to force width 2 or -N to force width 1.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.eastAsian, "east-asian", "E", false, "treat ambiguous-width characters as width 2")
	cmd.Flags().BoolVarP(&opts.narrow, "narrow", "N", false, "treat ambiguous-width characters as width 1")
	cmd.MarkFlagsMutuallyExclusive("east-asian", "narrow")
	return cmd
}

func runRoot(cmd *cobra.Command, args []string, opts *options) error {
	var eastAsian *bool
	switch {
	case opts.eastAsian:
		v := true
		eastAsian = &v
	case opts.narrow:
		v := false
		eastAsian = &v
	}

	out := cmd.OutOrStdout()

	if len(args) > 0 {
		for _, arg := range args {
			fmt.Fprintf(out, "%d\n", wcwidth.Width(arg, eastAsian))
		}
		return nil
	}

	s := bufio.NewScanner(cmd.InOrStdin())
	s.Buffer(make([]byte, 64*1024), 1<<30)
	for s.Scan() {
		fmt.Fprintf(out, "%d\n", wcwidth.Width(s.Text(), eastAsian))
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	return nil
}

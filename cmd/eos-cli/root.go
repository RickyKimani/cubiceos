package main

import (
	"github.com/rickykimani/cubiceos/internal/tui"
	"github.com/rickykimani/cubiceos/internal/web"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var httpMode bool

	cmd := &cobra.Command{
		Use:   "eos-cli",
		Short: "Interactive EOS solver",
		Long: `CubicEOS is a solver (in molar volume) for cubic equations of state.
By default, running 'eos-cli' launches the interactive terminal UI.

Use '--http' to start the web UI instead.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if httpMode {
				web.Run()
				return nil
			}
			return tui.Run()
		},
	}

	cmd.Flags().BoolVar(&httpMode, "http", false, "Launch the web UI instead of the TUI")

	return cmd
}

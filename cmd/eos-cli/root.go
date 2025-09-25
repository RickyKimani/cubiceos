package main

import (
	"github.com/rickykimani/cubiceos/internal/tui"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eos-cli",
		Short: "Interactive EOS solver",
		Long: `CubicEOS is a solver and explorer for cubic equations of state.
By default, running 'eos-cli' launches the interactive terminal UI.`, //add http later
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := tui.Run()
			return err

		},
	}
}

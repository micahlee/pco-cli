package cmd

import (
	"fmt"
	"os"

	"github.com/micahlee/pco-cli/internal/config"
	"github.com/micahlee/pco-cli/internal/output"
	"github.com/micahlee/pco-cli/internal/pco"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	svc        *pco.Service
	printer    output.Printer
)

var rootCmd = &cobra.Command{
	Use:   "pco",
	Short: "Planning Center Online CLI",
	Long:  "A command-line interface for interacting with the Planning Center Online API.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it.
		if cmd.Name() == "version" || cmd.Name() == "init" {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		svc = pco.NewService(cfg)
		printer = output.New(os.Stdout, jsonOutput)
		return nil
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

// Execute runs the root command.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

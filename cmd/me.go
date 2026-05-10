package cmd

import (
	"github.com/micahlee/pco-cli/internal/output"
	"github.com/spf13/cobra"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show my PCO profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		person, err := svc.Me(cmd.Context())
		if err != nil {
			return err
		}

		printer.Detail([]output.Field{
			{Label: "Name", Value: person.Attrs.Name},
			{Label: "Email", Value: person.Attrs.LoginIdentifier},
			{Label: "PCO ID", Value: person.ID},
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(meCmd)
}

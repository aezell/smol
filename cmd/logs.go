package cmd

import (
	"fmt"

	"github.com/smol-tools/smol/sprite"
	"github.com/spf13/cobra"
)

var logsLines int

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show web server logs for a site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		output, err := client.ServiceLogs(name, "web", logsLines)
		if err != nil {
			return fmt.Errorf("fetching logs: %w", err)
		}

		fmt.Print(output)
		return nil
	},
}

func init() {
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of log lines to show")
	rootCmd.AddCommand(logsCmd)
}

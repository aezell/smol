package cmd

import (
	"fmt"

	"github.com/smol-tools/smol/sprite"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open a site in your browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := siteURL(name)

		fmt.Printf("Opening %s\n", url)
		if err := sprite.OpenBrowser(url); err != nil {
			fmt.Printf("Visit: %s\n", url)
		}
		return nil
	},
}

func siteURL(name string) string {
	return fmt.Sprintf("https://%s.sprite.dev", name)
}

func init() {
	rootCmd.AddCommand(openCmd)
}

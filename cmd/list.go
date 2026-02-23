package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/smol-tools/smol/sprite"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	Aliases: []string{"ls"},
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		sprites, err := client.ListSprites()
		if err != nil {
			return fmt.Errorf("listing sites: %w", err)
		}

		if len(sprites) == 0 {
			fmt.Println("No sites yet. Create one with: smol create <name>")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS")
		for _, s := range sprites {
			status := s.Status
			if status == "" {
				status = "running"
			}
			fmt.Fprintf(w, "%s\t%s\n", s.Name, status)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

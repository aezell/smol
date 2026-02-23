package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/smol-tools/smol/sprite"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status for a site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		services, err := client.ListServices(name)
		if err != nil {
			return fmt.Errorf("listing services: %w", err)
		}

		fmt.Printf("Site: %s\n", name)
		fmt.Printf("URL:  %s\n\n", siteURL(name))

		if len(services) == 0 {
			fmt.Println("No services running. Deploy with: smol deploy ./folder --to " + name)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "SERVICE\tSTATUS")
		for _, s := range services {
			status := "unknown"
			if s.State != nil {
				status = s.State.Status
			}
			fmt.Fprintf(w, "%s\t%s\n", s.Name, status)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

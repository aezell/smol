package cmd

import (
	"fmt"

	"github.com/aezell/smol/sprite"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		fmt.Printf("Creating site %q...\n", name)
		if err := client.CreateSprite(name); err != nil {
			return fmt.Errorf("creating sprite: %w", err)
		}

		// Create the web root directory.
		if _, err := client.Exec(name, "mkdir -p /srv/www"); err != nil {
			return fmt.Errorf("creating web root: %w", err)
		}

		fmt.Printf("Site %q created.\n", name)
		fmt.Printf("Deploy files with: smol deploy ./your-folder --to %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}

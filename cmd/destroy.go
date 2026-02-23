package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/smol-tools/smol/sprite"
	"github.com/spf13/cobra"
)

var destroyForce bool

var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy a site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !destroyForce {
			fmt.Printf("Destroy site %q? This cannot be undone. [y/N] ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		fmt.Printf("Destroying site %q...\n", name)
		if err := client.DestroySprite(name); err != nil {
			return fmt.Errorf("destroying site: %w", err)
		}

		fmt.Printf("Site %q destroyed.\n", name)
		return nil
	},
}

func init() {
	destroyCmd.Flags().BoolVarP(&destroyForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(destroyCmd)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/aezell/smol/sprite"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove saved credentials",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := sprite.ConfigPath()
		if err != nil {
			return err
		}

		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not logged in.")
				return nil
			}
			return fmt.Errorf("removing config: %w", err)
		}

		fmt.Println("Logged out.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

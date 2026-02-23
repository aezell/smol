package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "smol",
	Short: "Publish static websites to Sprites",
	Long: `smol is a CLI for publishing small, static websites.

Point it at a folder of HTML, CSS, JS, and images, and it handles the rest:
creating a sprite, uploading your files, and serving them to the world.`,
	Example: `  smol create mysite
  smol deploy ./my-folder --to mysite
  smol list
  smol open mysite
  smol destroy mysite`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

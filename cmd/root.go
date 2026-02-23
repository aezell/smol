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

// Version is set at build time via ldflags.
var Version = "dev"

// Execute runs the root command.
func Execute() error {
	rootCmd.Version = Version
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

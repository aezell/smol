package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aezell/smol/sprite"
	"github.com/spf13/cobra"
)

var loginOrg string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with your Fly.io account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sprite.LoadConfig()
		if err != nil {
			return err
		}

		if cfg.IsLoggedIn() {
			fmt.Printf("Already logged in as %s (org: %s).\n", cfg.Email, cfg.Org)
			fmt.Println("Run 'smol logout' first to switch accounts.")
			return nil
		}

		// Step 1: Start a CLI auth session with Fly.io.
		fmt.Println("Opening browser to log in...")
		authURL, sessionID, err := sprite.StartLoginSession()
		if err != nil {
			return fmt.Errorf("starting login: %w", err)
		}

		fmt.Printf("\n  %s\n\n", authURL)

		if err := sprite.OpenBrowser(authURL); err != nil {
			fmt.Println("Could not open browser automatically. Visit the URL above.")
		}

		// Step 2: Wait for the user to complete login.
		fmt.Print("Waiting for authentication...")
		ctx := context.Background()
		flyToken, err := sprite.WaitForLogin(ctx, sessionID)
		if err != nil {
			fmt.Println(" failed.")
			return fmt.Errorf("authentication: %w", err)
		}
		fmt.Println(" done!")

		// Step 3: Get user info.
		user, err := sprite.GetCurrentUser(ctx, flyToken)
		if err != nil {
			return fmt.Errorf("getting user info: %w", err)
		}
		fmt.Printf("Logged in as %s\n\n", user.Email)

		// Step 4: Let user pick an organization.
		orgs, err := sprite.FetchOrganizations(ctx, flyToken)
		if err != nil {
			return fmt.Errorf("fetching organizations: %w", err)
		}

		var selectedOrg string
		if loginOrg != "" {
			// Validate the --org flag against available orgs.
			for _, org := range orgs {
				if org.Slug == loginOrg {
					selectedOrg = loginOrg
					break
				}
			}
			if selectedOrg == "" {
				return fmt.Errorf("organization %q not found in your account", loginOrg)
			}
			fmt.Printf("Using organization: %s\n", selectedOrg)
		} else if len(orgs) == 0 {
			return fmt.Errorf("no organizations found on your Fly.io account")
		} else if len(orgs) == 1 {
			selectedOrg = orgs[0].Slug
			fmt.Printf("Using organization: %s\n", selectedOrg)
		} else {
			fmt.Println("Select an organization:")
			for i, org := range orgs {
				label := org.Slug
				if org.Name != "" && org.Name != org.Slug {
					label = fmt.Sprintf("%s (%s)", org.Slug, org.Name)
				}
				fmt.Printf("  %d) %s\n", i+1, label)
			}
			fmt.Print("\nChoice: ")

			var input string
			fmt.Scanln(&input)
			choice, err := strconv.Atoi(input)
			if err != nil || choice < 1 || choice > len(orgs) {
				return fmt.Errorf("invalid selection")
			}
			selectedOrg = orgs[choice-1].Slug
		}

		// Step 5: Exchange Fly token for a Sprite API token.
		fmt.Printf("Creating API token for %s...", selectedOrg)

		apiURL := os.Getenv("SMOL_API_URL")
		if apiURL == "" {
			apiURL = "https://api.sprites.dev"
		}

		spriteToken, err := sprite.CreateSpriteToken(ctx, flyToken, selectedOrg, apiURL)
		if err != nil {
			fmt.Println(" failed.")
			return fmt.Errorf("creating API token: %w", err)
		}
		fmt.Println(" done!")

		// Step 6: Save config.
		cfg.Token = spriteToken
		cfg.APIURL = apiURL
		cfg.Org = selectedOrg
		cfg.Email = user.Email
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		cfgPath, _ := sprite.ConfigPath()
		fmt.Printf("\nYou're all set! Config saved to %s\n", cfgPath)
		fmt.Println("\nGet started:")
		fmt.Println("  smol create mysite")
		fmt.Println("  smol deploy ./my-folder --to mysite")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginOrg, "org", "", "organization slug (skips interactive selection)")
	rootCmd.AddCommand(loginCmd)
}

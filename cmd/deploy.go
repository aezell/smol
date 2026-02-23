package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aezell/smol/sprite"
	"github.com/spf13/cobra"
)

var deployTo string

var deployCmd = &cobra.Command{
	Use:   "deploy <directory>",
	Short: "Deploy a directory to a site",
	Long: `Deploy uploads files from a local directory to a site and ensures
the web server is running. If the site doesn't exist yet, it will be created.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if deployTo == "" {
			return fmt.Errorf("--to flag is required (site name)")
		}

		// Verify the source directory exists.
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("source directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}

		client, err := sprite.NewClient()
		if err != nil {
			return err
		}

		name := deployTo

		fmt.Printf("Preparing site %q...\n", name)

		// Check if the sprite exists; create it if not.
		needsSetup := false
		_, err = client.GetSprite(name)
		if err != nil {
			fmt.Printf("  Creating sprite...\n")
			if err := client.CreateSprite(name); err != nil {
				return fmt.Errorf("creating sprite: %w", err)
			}
			needsSetup = true
		}

		fmt.Printf("  Waiting for sprite...\n")
		if err := client.WaitReady(name); err != nil {
			return fmt.Errorf("waiting for sprite: %w", err)
		}

		// If the sprite existed, check if the web service is set up.
		if !needsSetup {
			services, _ := client.ListServices(name)
			hasWeb := false
			for _, s := range services {
				if s.Name == "web" {
					hasWeb = true
					break
				}
			}
			needsSetup = !hasWeb
		}

		// Upload files.
		if _, err := client.Exec(name, "mkdir -p /srv/www"); err != nil {
			return fmt.Errorf("creating web root: %w", err)
		}
		if !needsSetup {
			// Clean old files on update deploys.
			if _, err := client.Exec(name, "rm -rf /srv/www/*"); err != nil {
				return fmt.Errorf("cleaning web root: %w", err)
			}
		}
		if err := uploadFiles(client, name, dir); err != nil {
			return err
		}

		// Set up Caddy and make public if this is a first-time setup.
		if needsSetup {
			if err := setupWebServer(client, name); err != nil {
				return err
			}
		}

		// Get the sprite info for the actual URL.
		siteInfo, err := client.GetSprite(name)
		if err != nil {
			fmt.Printf("\nSite %q deployed.\n", name)
			fmt.Printf("Run 'smol open %s' to view it.\n", name)
			return nil
		}

		fmt.Printf("\nSite %q deployed.\n", name)
		fmt.Printf("Live at: %s\n", siteInfo.URL)
		return nil
	},
}

// setupWebServer installs Caddy, creates the service, and makes the site public.
func setupWebServer(client *sprite.Client, name string) error {
	fmt.Printf("  Installing Caddy...\n")
	installCmd := "curl -sL 'https://caddyserver.com/api/download?os=linux&arch=amd64' -o /usr/local/bin/caddy && chmod +x /usr/local/bin/caddy"
	if _, err := client.Exec(name, installCmd); err != nil {
		return fmt.Errorf("installing caddy: %w", err)
	}

	if err := client.CreateService(name, "web", "caddy", []string{"file-server", "--root", "/srv/www", "--listen", ":8080"}, 8080); err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	if err := client.StartService(name, "web"); err != nil {
		return fmt.Errorf("starting service: %w", err)
	}

	fmt.Printf("  Making site public...\n")
	if err := client.MakePublic(name); err != nil {
		return fmt.Errorf("making site public: %w", err)
	}

	return nil
}

// uploadFiles tars a local directory and uploads it to /srv/www on the sprite.
func uploadFiles(client *sprite.Client, name, dir string) error {
	fmt.Printf("  Uploading files...\n")
	tarBuf, fileCount, err := createTarGz(dir)
	if err != nil {
		return fmt.Errorf("creating archive: %w", err)
	}

	if err := client.UploadTar(name, tarBuf, "/srv/www"); err != nil {
		return fmt.Errorf("uploading files: %w", err)
	}
	fmt.Printf("  Uploaded %d files.\n", fileCount)
	return nil
}

func createTarGz(dir string) (*bytes.Buffer, int, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	fileCount := 0
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, 0, err
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories.
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") && path != absDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path.
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			fileCount++
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	if err := tw.Close(); err != nil {
		return nil, 0, err
	}
	if err := gz.Close(); err != nil {
		return nil, 0, err
	}

	return &buf, fileCount, nil
}

func init() {
	deployCmd.Flags().StringVar(&deployTo, "to", "", "target site name (required)")
	deployCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(deployCmd)
}

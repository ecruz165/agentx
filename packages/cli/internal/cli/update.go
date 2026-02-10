package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/config"
	"github.com/agentx-labs/agentx/internal/updater"
	"github.com/spf13/cobra"
)

var (
	updateCheck   bool
	updateForce   bool
	updateVersion string
)

func init() {
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Force update even if already on latest version")
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "Install a specific version (e.g., 1.2.0)")

	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"self-update"},
	Short:   "Update agentx to the latest version",
	Long: `Downloads and installs the latest version of agentx from GitHub releases
or a configured mirror.

  agentx update              # update to latest
  agentx update --check      # check only
  agentx update --version 1.2.0  # install specific version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve mirror from config or env var.
		config.Load()
		mirror := config.Get("mirror")
		if envMirror := os.Getenv(branding.EnvVar("MIRROR")); envMirror != "" {
			mirror = envMirror
		}

		var opts []updater.Option
		if mirror != "" {
			opts = append(opts, updater.WithMirror(mirror))
		}

		u := updater.New(buildVersion, opts...)

		// Fetch release.
		var release *updater.Release
		var err error
		if updateVersion != "" {
			fmt.Fprintf(os.Stderr, "Checking for version %s...\n", updateVersion)
			release, err = u.CheckSpecificVersion(updateVersion)
		} else {
			fmt.Fprintln(os.Stderr, "Checking for updates...")
			release, err = u.CheckLatestVersion()
		}
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		// Compare versions.
		available, err := updater.IsUpdateAvailable(buildVersion, release.Version)
		if err != nil {
			// If current version is "dev", treat as always updateable.
			if buildVersion == "dev" {
				available = true
			} else {
				return fmt.Errorf("comparing versions: %w", err)
			}
		}

		if updateCheck {
			if available {
				fmt.Printf("Update available: %s -> %s\n", buildVersion, release.Version)
			} else {
				fmt.Printf("You are on the latest version (%s)\n", buildVersion)
			}
			return nil
		}

		if !available && !updateForce {
			fmt.Printf("You are on the latest version (%s)\n", buildVersion)
			return nil
		}

		// Download.
		fmt.Fprintf(os.Stderr, "Downloading agentx %s for %s/%s...\n", release.Version, runtime.GOOS, runtime.GOARCH)

		tmpDir, err := os.MkdirTemp("", "agentx-update-*")
		if err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath, err := u.DownloadBinary(release, tmpDir)
		if err != nil {
			return fmt.Errorf("downloading binary: %w", err)
		}

		// Verify checksum.
		fmt.Fprintln(os.Stderr, "Verifying checksum...")
		if err := u.VerifyChecksum(release, archivePath); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		// Extract.
		binPath, err := updater.ExtractBinary(archivePath, tmpDir)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}

		// Replace.
		fmt.Fprintln(os.Stderr, "Installing...")
		currentBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding current binary: %w", err)
		}

		if err := updater.ReplaceBinary(binPath, currentBinary, release.Version); err != nil {
			return err
		}

		// Update cache.
		cache := &updater.VersionCache{
			LatestVersion:   release.Version,
			CurrentVersion:  release.Version,
			UpdateAvailable: false,
		}
		_ = updater.SaveCache(config.Dir(), cache)

		fmt.Printf("Successfully updated to %s\n", release.Version)
		return nil
	},
}

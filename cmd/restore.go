package cmd

import (
	"fmt" // Needed for Errorf
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner" // Added spinner
	"github.com/spf13/cobra"
	"github.com/tuannvm/mcpenetes/internal/config"
	"github.com/tuannvm/mcpenetes/internal/log" // Added log
	"github.com/tuannvm/mcpenetes/internal/util"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restores client configurations from the latest backups.",
	Long: `Restores the configuration files for all defined clients 
from the most recent backup found in the backup directory specified in config.yaml.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Executing restore command...")

		// 1. Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Error loading config.yaml: %v", err)
		}

		if len(cfg.Clients) == 0 {
			log.Warn("No clients defined in config.yaml. Nothing to restore.")
			return
		}

		backupDir, err := util.ExpandPath(cfg.Backups.Path)
		if err != nil {
			log.Fatal("Error expanding backup path '%s': %v", cfg.Backups.Path, err)
		}

		// 2. List backup files
		log.Detail("Reading backup directory: %s", backupDir)
		backupFiles, err := os.ReadDir(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Warn("Backup directory '%s' does not exist. Nothing to restore.", backupDir)
				return
			}
			log.Fatal("Error reading backup directory '%s': %v", backupDir, err)
		}

		// 3. Group backups by client name
		clientBackups := make(map[string][]string) // Map clientName -> list of backup filenames
		for _, entry := range backupFiles {
			if entry.IsDir() {
				continue // Skip directories
			}
			fileName := entry.Name()
			// Basic parsing: expect format like <clientName>-<timestamp>.<ext>
			parts := strings.SplitN(fileName, "-", 2)
			if len(parts) < 2 {
				log.Warn("Skipping unrecognized file in backup directory: %s", fileName)
				continue // Doesn't match expected format
			}
			clientName := parts[0]
			clientBackups[clientName] = append(clientBackups[clientName], fileName)
		}

		// 4. Iterate through configured clients and restore the latest backup
		log.Info("Restoring client configurations:")
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Restoring..."
		s.Start()

		successCount := 0
		failureCount := 0
		clientErrors := make(map[string]error) // Store errors to show after spinner
		clientSuccess := make(map[string]string) // Store success info (backup filename)
		clientSkipped := make(map[string]bool) // Store clients with no backups

		for clientName, clientConf := range cfg.Clients {
			backups, found := clientBackups[clientName]
			if !found || len(backups) == 0 {
				clientSkipped[clientName] = true
				continue
			}

			// Find the latest backup (sort filenames descending)
			sort.Sort(sort.Reverse(sort.StringSlice(backups)))
			latestBackupFileName := backups[0]
			latestBackupPath := filepath.Join(backupDir, latestBackupFileName)

			clientConfigPath, err := util.ExpandPath(clientConf.ConfigPath)
			if err != nil {
				clientErrors[clientName] = fmt.Errorf("error expanding client config path '%s': %w", clientConf.ConfigPath, err)
				failureCount++
				continue
			}

			// Perform the restore (copy backup to original location)
			err = copyFile(latestBackupPath, clientConfigPath)
			if err != nil {
				clientErrors[clientName] = fmt.Errorf("error restoring config from '%s': %w", latestBackupFileName, err)
				failureCount++
				continue
			}

			clientSuccess[clientName] = latestBackupFileName
			successCount++
		}

		s.Stop()

		// Log results after spinner stops
		for clientName := range cfg.Clients {
			if err, failed := clientErrors[clientName]; failed {
				log.Error("- %s: Failed restore - %v", clientName, err)
			} else if _, skipped := clientSkipped[clientName]; skipped {
				log.Warn("- %s: No backups found to restore.", clientName)
			} else if backupFile, success := clientSuccess[clientName]; success {
				log.Success("- %s: Successfully restored from %s", clientName, backupFile)
			} else {
				// Should not happen if logic is correct, but handle defensively
				log.Warn("- %s: No action taken (unexpected state).", clientName)
			}
		}

		log.Info("\nRestore finished.")
		log.Success("Successfully restored %d clients.", successCount)
		if failureCount > 0 {
			log.Error("Failed to restore %d clients.", failureCount)
			os.Exit(1) // Exit with error if any client failed
		}
	},
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := source.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0750); err != nil {
		return fmt.Errorf("failed to create destination directory '%s': %w", dstDir, err)
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destination.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(destination, source)
	return err
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

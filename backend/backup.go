package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Setting this to a path that maps to the host's /var/www/wiki/files via the Docker volume
const persistentDir = "/app/persistent"

// BackupWikiFiles copies all wiki text files to the persistent storage directory
func BackupWikiFiles() {
	// Create the persistent directory if it doesn't exist
	if err := os.MkdirAll(persistentDir, 0755); err != nil {
		log.Printf("Error creating persistent directory: %v", err)
		return
	}

	// Get all txt files in the current directory
	files, err := filepath.Glob("*.txt")
	if err != nil {
		log.Printf("Error finding wiki files: %v", err)
		return
	}

	// Copy each file to the persistent directory
	for _, file := range files {
		// Skip files.txt metafiles as they'll be recreated
		if strings.HasSuffix(file, ".files.txt") {
			continue
		}

		// Read the source file
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		// Write to the destination file
		destPath := filepath.Join(persistentDir, file)
		if err := os.WriteFile(destPath, content, 0600); err != nil {
			log.Printf("Error writing to persistent storage %s: %v", destPath, err)
		} else {
			log.Printf("Backed up %s to %s", file, destPath)
		}
	}
}

// SetupFileWatcher runs BackupWikiFiles on a regular interval
func SetupFileWatcher() {
	// Immediately backup files when starting
	BackupWikiFiles()

	// Set up a ticker to run backups every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			BackupWikiFiles()
		}
	}()
}

// RestoreWikiFile tries to load a file from the persistent directory if it doesn't exist in the app directory
func RestoreWikiFile(title string) error {
	filename := title + ".txt"
	
	// Check if the file exists in the app directory
	if _, err := os.Stat(filename); err == nil {
		// File exists in app directory, no need to restore
		return nil
	}
	
	// Check if the file exists in persistent directory
	persistentPath := filepath.Join(persistentDir, filename)
	if _, err := os.Stat(persistentPath); err != nil {
		// File doesn't exist in persistent directory either
		return err
	}
	
	// Read from persistent directory
	content, err := os.ReadFile(persistentPath)
	if err != nil {
		return err
	}
	
	// Write to app directory
	if err := os.WriteFile(filename, content, 0600); err != nil {
		return err
	}
	
	log.Printf("Restored %s from persistent storage", filename)
	return nil
} 
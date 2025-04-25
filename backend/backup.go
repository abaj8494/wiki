package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"io"
)

// persistentDir is declared in wiki.go
// init function to ensure variable values are coordinated
func init() {
	if persistentDir == "" {
		persistentDir = "/app/persistence"
	}
}

// BackupWikiFiles copies all wiki text files and uploaded files to the persistent storage directory
func BackupWikiFiles() {
	// Create the persistent directory if it doesn't exist
	if err := os.MkdirAll(persistentDir, 0755); err != nil {
		log.Printf("Error creating persistent directory: %v", err)
		return
	}

	// 1. Backup text files
	// Get all txt files in the current directory
	textFiles, err := filepath.Glob("*.txt")
	if err != nil {
		log.Printf("Error finding wiki text files: %v", err)
		return
	}

	// Copy each file to the persistent directory
	for _, file := range textFiles {
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

	// 2. Backup uploaded files
	if err := backupUploadedFiles(); err != nil {
		log.Printf("Error backing up uploaded files: %v", err)
	}
}

// backupUploadedFiles copies all uploaded files to the persistent storage
func backupUploadedFiles() error {
	// Get all directories in the filesDir (each directory is named after a page title)
	dirs, err := os.ReadDir(filesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// If the directory doesn't exist yet, there's nothing to backup
			return nil
		}
		return err
	}

	// Create the persistent files directory
	persistentFilesDir := filepath.Join(persistentDir, "files")
	if err := os.MkdirAll(persistentFilesDir, 0755); err != nil {
		return err
	}

	// For each page directory
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		pageName := dir.Name()
		pageDir := filepath.Join(filesDir, pageName)
		
		// Create corresponding directory in persistent storage
		persistentPageDir := filepath.Join(persistentFilesDir, pageName)
		if err := os.MkdirAll(persistentPageDir, 0755); err != nil {
			log.Printf("Error creating persistent directory for page %s: %v", pageName, err)
			continue
		}

		// Copy all files from this page directory
		files, err := os.ReadDir(pageDir)
		if err != nil {
			log.Printf("Error reading files for page %s: %v", pageName, err)
			continue
		}

		for _, fileInfo := range files {
			if fileInfo.IsDir() {
				continue // Skip subdirectories
			}

			fileName := fileInfo.Name()
			srcPath := filepath.Join(pageDir, fileName)
			destPath := filepath.Join(persistentPageDir, fileName)

			// Copy the file
			if err := copyFile(srcPath, destPath); err != nil {
				log.Printf("Error copying file %s: %v", srcPath, err)
			} else {
				log.Printf("Backed up attachment %s to %s", srcPath, destPath)
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Sync to ensure write is complete
	return destFile.Sync()
}

// RestoreAllFiles restores all wiki files and attachments from persistent storage
func RestoreAllFiles() {
	log.Printf("Starting restoration from %s", persistentDir)
	
	// Also restore .files.txt metafiles first
	metafiles, err := filepath.Glob(filepath.Join(persistentDir, "*.files.txt"))
	if err == nil {
		for _, metaFile := range metafiles {
			fileName := filepath.Base(metaFile)
			destPath := fileName
			
			// Read from persistent storage
			content, err := os.ReadFile(metaFile)
			if err == nil {
				if err := os.WriteFile(destPath, content, 0600); err != nil {
					log.Printf("Error restoring metafile %s: %v", fileName, err)
				} else {
					log.Printf("Restored metadata file %s", fileName)
				}
			}
		}
	}
	
	// Restore text files
	files, err := filepath.Glob(filepath.Join(persistentDir, "*.txt"))
	if err != nil {
		log.Printf("Error finding persistent text files: %v", err)
		return
	}

	for _, persistentFile := range files {
		// Skip .files.txt files as we already processed them
		if strings.HasSuffix(persistentFile, ".files.txt") {
			continue
		}
		
		// Get the filename without the path
		fileName := filepath.Base(persistentFile)
		
		// Create the destination path
		destPath := fileName
		
		// Read from persistent storage
		content, err := os.ReadFile(persistentFile)
		if err != nil {
			log.Printf("Error reading persistent file %s: %v", persistentFile, err)
			continue
		}
		
		// Write to app directory
		if err := os.WriteFile(destPath, content, 0600); err != nil {
			log.Printf("Error restoring file %s: %v", fileName, err)
		} else {
			log.Printf("Restored %s from persistent storage", fileName)
			
			// Extract title from filename (remove .txt extension)
			title := strings.TrimSuffix(fileName, ".txt")
			
			// Also restore any uploaded files for this page
			if err := RestoreUploadedFiles(title); err != nil {
				log.Printf("Error restoring uploaded files for %s: %v", title, err)
			}
		}
	}
	
	// Debug - list all files in persistence folder
	if files, err := filepath.Glob(filepath.Join(persistentDir, "*")); err == nil {
		log.Printf("Files in persistent directory: %v", files)
		
		// Check files folder too
		if files, err := filepath.Glob(filepath.Join(persistentDir, "files", "*")); err == nil {
			log.Printf("Files in persistent/files directory: %v", files)
		}
	}
}

// RestoreUploadedFiles restores all uploaded files for a specific page
func RestoreUploadedFiles(title string) error {
	// Source directory in persistent storage
	srcDir := filepath.Join(persistentDir, "files", title)
	
	// Check if the directory exists in persistent storage
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		// No backups exist for this page
		return nil
	}
	
	// Destination directory in app
	destDir := filepath.Join(filesDir, title)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	
	// Get all files in the persistent directory
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	
	// Copy each file to the app directory
	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue
		}
		
		fileName := fileInfo.Name()
		srcPath := filepath.Join(srcDir, fileName)
		destPath := filepath.Join(destDir, fileName)
		
		if err := copyFile(srcPath, destPath); err != nil {
			log.Printf("Error restoring file %s: %v", fileName, err)
		} else {
			log.Printf("Restored file %s for page %s", fileName, title)
		}
	}
	
	return nil
}

// SetupFileWatcher performs initial backup and restoration of wiki files at startup
func SetupFileWatcher() {
	// First restore all files from persistent storage
	RestoreAllFiles()
	log.Println("Initial restoration completed.")
	
	// Then backup any new files
	BackupWikiFiles()
	log.Println("Initial backup completed. Automatic backups will occur after file modifications.")
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
	
	// Also restore any uploaded files for this page
	if err := RestoreUploadedFiles(title); err != nil {
		log.Printf("Error restoring uploaded files for %s: %v", title, err)
	}
	
	log.Printf("Restored %s from persistent storage", filename)
	return nil
} 
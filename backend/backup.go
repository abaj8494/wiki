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

	// 1. Backup text files and .files.txt metadata files
	// Get all txt files in the current directory
	textFiles, err := filepath.Glob("*.txt")
	if err != nil {
		log.Printf("Error finding wiki text files: %v", err)
		return
	}

	// Copy each file to the persistent directory
	for _, file := range textFiles {
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
	
	// Check if persistent directory exists
	if _, err := os.Stat(persistentDir); os.IsNotExist(err) {
		log.Printf("Persistent directory %s does not exist, skipping restoration", persistentDir)
		return
	}
	
	// Get all files from persistent directory (both .txt and .files.txt)
	allFiles, err := filepath.Glob(filepath.Join(persistentDir, "*.txt"))
	if err != nil {
		log.Printf("Error finding persistent files: %v", err)
		return
	}
	
	// Keep track of page titles we've restored
	restoredPages := make(map[string]bool)
	
	// Process all files from persistent storage
	for _, persistentFile := range allFiles {
		fileName := filepath.Base(persistentFile)
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
			
			// For regular txt files (not .files.txt), also restore the attachments
			if !strings.HasSuffix(fileName, ".files.txt") {
				title := strings.TrimSuffix(fileName, ".txt")
				restoredPages[title] = true
				
				// Restore any uploaded files for this page
				if err := RestoreUploadedFiles(title); err != nil {
					log.Printf("Error restoring uploaded files for %s: %v", title, err)
				}
			}
		}
	}
	
	// Now check for pages with attachments but no .files.txt
	// This can happen if the files directory exists but the metadata file was lost
	regenerateAttachmentMetadata(restoredPages)
	
	// Debug - list all files in persistence folder
	if files, err := filepath.Glob(filepath.Join(persistentDir, "*")); err == nil {
		log.Printf("Files in persistent directory: %v", files)
		
		// Check files folder too
		if files, err := filepath.Glob(filepath.Join(persistentDir, "files", "*")); err == nil {
			log.Printf("Files in persistent/files directory: %v", files)
		}
	}
}

// regenerateAttachmentMetadata ensures all pages with attachments have proper .files.txt metadata files
func regenerateAttachmentMetadata(restoredPages map[string]bool) {
	// Check the persistent files directory
	persistentFilesDir := filepath.Join(persistentDir, "files")
	if _, err := os.Stat(persistentFilesDir); os.IsNotExist(err) {
		return
	}
	
	// Get all directories in the persistent files directory
	dirs, err := os.ReadDir(persistentFilesDir)
	if err != nil {
		log.Printf("Error reading persistent files directory: %v", err)
		return
	}
	
	// For each page directory
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		
		pageName := dir.Name()
		// If we already processed this page, skip it
		if restoredPages[pageName] {
			continue
		}
		
		// Check if we have attachments for this page
		pageDir := filepath.Join(persistentFilesDir, pageName)
		files, err := os.ReadDir(pageDir)
		if err != nil || len(files) == 0 {
			continue
		}
		
		// Create the page .txt file if it doesn't exist (empty content)
		pageFile := pageName + ".txt"
		if _, err := os.Stat(pageFile); os.IsNotExist(err) {
			// Look for it in persistent storage first
			persistentPageFile := filepath.Join(persistentDir, pageFile)
			if _, err := os.Stat(persistentPageFile); err == nil {
				// File exists in persistent storage, copy it
				content, err := os.ReadFile(persistentPageFile)
				if err == nil {
					if err := os.WriteFile(pageFile, content, 0600); err != nil {
						log.Printf("Error restoring page file %s: %v", pageFile, err)
					}
				}
			} else {
				// Create empty page file
				if err := os.WriteFile(pageFile, []byte{}, 0600); err != nil {
					log.Printf("Error creating empty page file %s: %v", pageFile, err)
					continue
				}
			}
		}
		
		// Build the list of attachment filenames
		var fileNames []string
		for _, file := range files {
			if !file.IsDir() {
				fileNames = append(fileNames, file.Name())
			}
		}
		
		if len(fileNames) > 0 {
			// Create or update the .files.txt metadata file
			filesListFilename := pageName + ".files.txt"
			filesContent := strings.Join(fileNames, "\n")
			if err := os.WriteFile(filesListFilename, []byte(filesContent), 0600); err != nil {
				log.Printf("Error creating metadata file %s: %v", filesListFilename, err)
			} else {
				log.Printf("Generated metadata file for %s with %d attachments", pageName, len(fileNames))
			}
			
			// Also restore the actual files to the app directory
			if err := RestoreUploadedFiles(pageName); err != nil {
				log.Printf("Error restoring generated attachment files for %s: %v", pageName, err)
			}
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
	filesCopied := false
	var fileNames []string
	
	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue
		}
		
		fileName := fileInfo.Name()
		fileNames = append(fileNames, fileName)
		srcPath := filepath.Join(srcDir, fileName)
		destPath := filepath.Join(destDir, fileName)
		
		if err := copyFile(srcPath, destPath); err != nil {
			log.Printf("Error restoring file %s: %v", fileName, err)
		} else {
			log.Printf("Restored file %s for page %s", fileName, title)
			filesCopied = true
		}
	}
	
	// If we successfully copied files, ensure the metadata file exists
	if filesCopied && len(fileNames) > 0 {
		filesListFilename := title + ".files.txt"
		// Check if the file exists first
		_, err := os.Stat(filesListFilename)
		if os.IsNotExist(err) {
			// File doesn't exist, create it
			filesContent := strings.Join(fileNames, "\n")
			if err := os.WriteFile(filesListFilename, []byte(filesContent), 0600); err != nil {
				log.Printf("Error creating missing metadata file %s: %v", filesListFilename, err)
			} else {
				log.Printf("Created missing metadata file for %s with %d attachments", title, len(fileNames))
			}
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
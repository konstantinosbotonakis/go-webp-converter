package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"errors"
	"imageconverter/internal/converter"
	"imageconverter/internal/filesystem"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// runApp encapsulates the core application logic.
// It returns a list of messages detailing operations and an error for critical issues.
func runApp(inputPath string, forceOverwrite bool) ([]string, error) {
	var messages []string

	// Check if path exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		// Return error immediately for path not existing, as it's a prerequisite.
		return messages, fmt.Errorf("path '%s' does not exist", inputPath)
	} else if err != nil {
		return messages, fmt.Errorf("error checking path '%s': %w", inputPath, err)
	}

	messages = append(messages, fmt.Sprintf("Input path: %s", inputPath))
	messages = append(messages, fmt.Sprintf("Force overwrite: %t", forceOverwrite))

	files, err := filesystem.FindFiles(inputPath)
	if err != nil {
		// Return error for issues during file searching.
		return messages, fmt.Errorf("error finding files: %w", err)
	}

	if len(files) == 0 {
		messages = append(messages, "No processable files found.")
		return messages, nil
	}

	messages = append(messages, "Processing files...")
	for _, fPath := range files {
		ext := strings.ToLower(filepath.Ext(fPath))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			baseName := strings.TrimSuffix(filepath.Base(fPath), filepath.Ext(fPath))
			outputFilePath := filepath.Join(filepath.Dir(fPath), baseName+".webp")

			errConv := converter.ConvertToWebP(fPath, outputFilePath, forceOverwrite)
			if errConv != nil {
				if strings.Contains(errConv.Error(), "already exists, use --force to overwrite") {
					messages = append(messages, fmt.Sprintf("Skipping conversion (file exists): %s", outputFilePath))
				} else {
					// For other conversion errors, add to messages but don't necessarily stop all processing.
					// Depending on desired strictness, could return error here.
					messages = append(messages, fmt.Sprintf("Failed to convert %s: %v", fPath, errConv))
				}
			} else {
				messages = append(messages, fmt.Sprintf("Successfully converted %s to %s", fPath, outputFilePath))
			}
		} else {
			messages = append(messages, fmt.Sprintf("Skipping non-image file: %s", fPath))
		}
	}
	return messages, nil
}

func main() {
	// Define flags
	path := flag.String("path", "", "Input file or directory path (required)")
	flag.StringVar(path, "p", "", "Input file or directory path (alias for -path)")
	force := flag.Bool("force", false, "Overwrite existing files")
	flag.BoolVar(force, "f", false, "Overwrite existing files (alias for -force)")

	flag.Parse()

	// Basic flag validation
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Error: Input path is required. Use --path or -p.")
		flag.Usage()
		os.Exit(1)
	}

	messages, err := runApp(*path, *force)

	for _, msg := range messages {
		// Simple routing: if "Failed" or "Skipping conversion (file exists)", could go to Stderr or just Stdout.
		// For this refactor, let's print all operational messages to Stdout.
		// Critical errors from runApp (like path not existing) are handled below.
		fmt.Println(msg)
	}

	if err != nil {
		// Handle critical errors returned by runApp
		fmt.Fprintf(os.Stderr, "Critical error: %v\n", err)
		// Determine exit code based on error type if needed
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "does not exist") {
			os.Exit(2) // Specific exit code for path not found
		} else if strings.Contains(err.Error(), "finding files") {
			os.Exit(3) // Specific exit code for file finding errors
		}
		os.Exit(1) // General error
	}
}

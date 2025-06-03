package main

import (
	"flag"
	"fmt"
	"os"
	"io"
	"net/http"
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
		return messages, fmt.Errorf("path '%s' does not exist", inputPath)
	} else if err != nil {
		return messages, fmt.Errorf("error checking path '%s': %w", inputPath, err)
	}

	messages = append(messages, fmt.Sprintf("INFO: Input path: %s", inputPath))
	messages = append(messages, fmt.Sprintf("INFO: Force overwrite: %t", forceOverwrite))

	files, err := filesystem.FindFiles(inputPath)
	if err != nil {
		return messages, fmt.Errorf("error finding files: %w", err)
	}

	if len(files) == 0 {
		messages = append(messages, "INFO: No processable files found.")
		return messages, nil
	}

	messages = append(messages, "INFO: Processing files...")
	for _, fPath := range files {
		file, openErr := os.Open(fPath)
		if openErr != nil {
			messages = append(messages, fmt.Sprintf("ERROR: Error opening file %s: %v. Skipping.", fPath, openErr))
			continue
		}

		buffer := make([]byte, 512)
		n, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			messages = append(messages, fmt.Sprintf("ERROR: Error reading file %s for content type detection: %v. Skipping.", fPath, readErr))
			file.Close()
			continue
		}
		mimeType := http.DetectContentType(buffer[:n])

		_, seekErr := file.Seek(0, 0)
		if seekErr != nil {
			messages = append(messages, fmt.Sprintf("ERROR: Error seeking in file %s: %v. Skipping.", fPath, seekErr))
			file.Close()
			continue
		}
		// It's important to close the file if we are done with it here,
		// or ensure ConvertToWebP handles an already open file (currently it reopens).
		// For simplicity, closing it here is fine since ConvertToWebP reopens.
		file.Close()

		messages = append(messages, fmt.Sprintf("INFO: File: %s, Detected MIME type: %s", fPath, mimeType))

		isSupportedMimeType := false
		switch mimeType {
		case "image/jpeg", "image/png", "image/gif":
			isSupportedMimeType = true
		}

		if isSupportedMimeType {
			baseName := strings.TrimSuffix(filepath.Base(fPath), filepath.Ext(fPath))
			outputFilePath := filepath.Join(filepath.Dir(fPath), baseName+".webp")

			errConv := converter.ConvertToWebP(fPath, outputFilePath, forceOverwrite)
			if errConv != nil {
				if strings.Contains(errConv.Error(), "already exists, use --force to overwrite") {
					// This specific error is more of a notice/skip condition if force is false.
					messages = append(messages, fmt.Sprintf("INFO: Skipping conversion (file exists, based on content type): %s", outputFilePath))
				} else {
					messages = append(messages, fmt.Sprintf("ERROR: Failed to convert %s (MIME: %s): %v", fPath, mimeType, errConv))
				}
			} else {
				messages = append(messages, fmt.Sprintf("INFO: Successfully converted %s (MIME: %s) to %s", fPath, mimeType, outputFilePath))
			}
		} else {
			messages = append(messages, fmt.Sprintf("INFO: Skipping file %s (detected MIME type: %s, not a supported image format).", fPath, mimeType))
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

	if *path == "" {
		fmt.Fprintln(os.Stderr, "Error: Input path is required. Use --path or -p.")
		flag.Usage()
		os.Exit(1)
	}

	messages, err := runApp(*path, *force)

	for _, msg := range messages {
		if strings.HasPrefix(msg, "ERROR:") {
			fmt.Fprintln(os.Stderr, msg)
		} else {
			// Default to Stdout for INFO and other messages
			fmt.Println(msg)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "CRITICAL: %v\n", err)
		// Determine exit code based on error type if needed
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "does not exist") {
			os.Exit(2) // Specific exit code for path not found
		} else if strings.Contains(err.Error(), "finding files") {
			os.Exit(3) // Specific exit code for file finding errors
		}
		os.Exit(1) // General error
	}
}

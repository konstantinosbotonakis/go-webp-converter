package converter

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/chai2010/webp"
)

// ConvertToWebP converts an image file (PNG or JPEG) to WebP format.
// If force is true, it will overwrite the outputFile if it already exists.
func ConvertToWebP(inputFile string, outputFile string, force bool) error {
	// Check if output file exists
	if _, err := os.Stat(outputFile); err == nil { // File exists
		if !force {
			return fmt.Errorf("output file %s already exists, use --force to overwrite", outputFile)
		}
		// If force is true, we can optionally print a message here or just proceed
		// fmt.Printf("Output file %s exists, overwriting due to --force flag.\n", outputFile)
	} else if !errors.Is(err, os.ErrNotExist) { // Another error occurred with os.Stat
		return fmt.Errorf("failed to check output file %s: %w", outputFile, err)
	}
	// If os.ErrNotExist, proceed to create the file

	// Open input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inputFile, err)
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		// It's useful to know which format failed, if image.Decode can provide it.
		// If format is empty, it means the decoder couldn't even determine the format.
		if format != "" {
			return fmt.Errorf("failed to decode image %s (format: %s): %w", inputFile, format, err)
		}
		return fmt.Errorf("failed to decode image %s (unknown format): %w", inputFile, err)
	}

	// Create output file
	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
	}
	defer output.Close()

	// Encode the image to WebP
	// Using github.com/chai2010/webp, a common way to encode is with options.
	// Let's use some default lossy options.
	options := &webp.Options{Lossless: false, Quality: 80.0}
	if err := webp.Encode(output, img, options); err != nil {
		return fmt.Errorf("failed to encode image %s to WebP (chai2010): %w", inputFile, err)
	}

	return nil
}

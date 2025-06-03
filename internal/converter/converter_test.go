package converter_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"github.com/chai2010/webp" // Changed from golang.org/x/image/webp

	"imageconverter/internal/converter"
)

// Helper function to create a dummy image file
func createDummyImage(t *testing.T, filename string, format string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255}) // A single red pixel

	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create dummy image file %s: %v", filename, err)
	}
	defer file.Close()

	switch format {
	case "png":
		if err := png.Encode(file, img); err != nil {
			t.Fatalf("Failed to encode dummy PNG %s: %v", filename, err)
		}
	case "jpeg":
		if err := jpeg.Encode(file, img, nil); err != nil {
			t.Fatalf("Failed to encode dummy JPEG %s: %v", filename, err)
		}
	default:
		t.Fatalf("Unsupported dummy image format: %s", format)
	}
}

func TestConvertToWebP_SuccessPNG(t *testing.T) {
	inputFile := "test_input.png"
	outputFile := "test_output.webp"
	createDummyImage(t, inputFile, "png")
	defer os.Remove(inputFile)
	defer os.Remove(outputFile) // Ensure cleanup even if test fails early

	err := converter.ConvertToWebP(inputFile, outputFile, false)
	if err != nil {
		t.Fatalf("ConvertToWebP failed for PNG: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output WebP file %s was not created", outputFile)
	}

	// Optional: Verify WebP content
	file, err := os.Open(outputFile)
	if err != nil {
		t.Fatalf("Failed to open output WebP file %s for verification: %v", outputFile, err)
	}
	defer file.Close()
	if _, err := webp.Decode(file); err != nil {
		t.Fatalf("Failed to decode output WebP file %s, it might be invalid: %v", outputFile, err)
	}
}

func TestConvertToWebP_SuccessJPEG(t *testing.T) {
	inputFile := "test_input.jpg"
	outputFile := "test_output.webp"
	createDummyImage(t, inputFile, "jpeg")
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)

	err := converter.ConvertToWebP(inputFile, outputFile, false)
	if err != nil {
		t.Fatalf("ConvertToWebP failed for JPEG: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output WebP file %s was not created", outputFile)
	}
	// Optional: Verify WebP content (as in PNG test)
	file, err := os.Open(outputFile)
	if err != nil {
		t.Fatalf("Failed to open output WebP file %s for verification: %v", outputFile, err)
	}
	defer file.Close()
	if _, err := webp.Decode(file); err != nil {
		t.Fatalf("Failed to decode output WebP file %s, it might be invalid: %v", outputFile, err)
	}
}

func TestConvertToWebP_InputFileNonExistent(t *testing.T) {
	inputFile := "nonexistent.jpg"
	outputFile := "output.webp"
	// Ensure the input file really doesn't exist, just in case
	_ = os.Remove(inputFile)
	defer os.Remove(outputFile)


	err := converter.ConvertToWebP(inputFile, outputFile, false)
	if err == nil {
		t.Fatalf("Expected ConvertToWebP to return an error for a non-existent input file, but got nil")
	}
}

func TestConvertToWebP_OutputFileExists_NoForce(t *testing.T) {
	inputFile := "input.png"
	outputFile := "existing_output.webp"
	createDummyImage(t, inputFile, "png")
	defer os.Remove(inputFile)

	// Create a dummy output file
	if err := os.WriteFile(outputFile, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to create dummy output file: %v", err)
	}
	defer os.Remove(outputFile)

	err := converter.ConvertToWebP(inputFile, outputFile, false)
	if err == nil {
		t.Fatalf("Expected ConvertToWebP to return an error when output file exists and force is false, but got nil")
	}
	expectedErrorMsg := "already exists, use --force to overwrite"
	if !bytes.Contains([]byte(err.Error()), []byte(expectedErrorMsg)) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestConvertToWebP_OutputFileExists_WithForce(t *testing.T) {
	inputFile := "input_force.png"
	outputFile := "output_force.webp"
	createDummyImage(t, inputFile, "png")
	defer os.Remove(inputFile)

	// Create an initial dummy output file
	initialContent := []byte("initial dummy content")
	if err := os.WriteFile(outputFile, initialContent, 0644); err != nil {
		t.Fatalf("Failed to create initial dummy output file: %v", err)
	}
	defer os.Remove(outputFile)

	initialStat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat initial output file: %v", err)
	}

	err = converter.ConvertToWebP(inputFile, outputFile, true)
	if err != nil {
		t.Fatalf("ConvertToWebP failed with force=true: %v", err)
	}

	finalStat, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat final output file: %v", err)
	}

	// Check if file was overwritten (content will be different, size might change, modtime will change)
	// A simple check is that the content is no longer the initialContent.
	// For a more robust check, compare mod times or sizes if they are expected to change significantly.
	finalContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read final output file: %v", err)
	}
	if bytes.Equal(finalContent, initialContent) {
		t.Errorf("Expected output file content to change after overwrite with force=true, but it remained the same. ModTime initial: %s, final: %s", initialStat.ModTime(), finalStat.ModTime())
	}
	if initialStat.ModTime() == finalStat.ModTime() && initialStat.Size() == finalStat.Size() {
         // If mod time and size are exactly the same, it's highly unlikely it was overwritten with new image data.
         // However, some file systems have low-resolution timestamps.
         // The content check above is more reliable.
         t.Logf("Warning: ModTime and Size of the output file did not change. Initial: %s, %d bytes. Final: %s, %d bytes. This might be okay if the dummy image is identical or due to filesystem timestamp resolution.", initialStat.ModTime(), initialStat.Size(), finalStat.ModTime(), finalStat.Size())
    }
}

func TestConvertToWebP_InvalidInputFormat(t *testing.T) {
	inputFile := "fake_image.png" // Looks like a PNG by extension, but isn't
	outputFile := "output_invalid.webp"

	// Create a text file instead of an image
	if err := os.WriteFile(inputFile, []byte("this is not an image"), 0644); err != nil {
		t.Fatalf("Failed to create fake image file: %v", err)
	}
	defer os.Remove(inputFile)
	defer os.Remove(outputFile)

	err := converter.ConvertToWebP(inputFile, outputFile, false)
	if err == nil {
		t.Fatalf("Expected ConvertToWebP to return an error for an invalid input image format, but got nil")
	}
	// We expect an error from image.Decode, which might say "unknown format" or specific format errors
	// For example: "failed to decode image fake_image.png (unknown format): image: unknown format"
	expectedErrorMsg := "failed to decode image"
	if !bytes.Contains([]byte(err.Error()), []byte(expectedErrorMsg)) {
		t.Errorf("Expected error message to indicate a decoding failure, got '%s'", err.Error())
	}
}

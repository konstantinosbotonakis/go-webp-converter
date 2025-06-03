package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create a dummy image file for integration tests
// This can be kept, or we can use a more generic file creator for some tests.
func createIntegrationTestImage(t *testing.T, dir string, filename string, format string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	var finalImg image.Image
	rect := image.Rect(0, 0, 1, 1)

	if strings.ToLower(format) == "gif" {
		// For GIF, create a paletted image
		palette := color.Palette([]color.Color{color.Transparent, color.RGBA{R: 255, A: 255}}) // Simple palette with red
		palettedImg := image.NewPaletted(rect, palette)
		palettedImg.SetColorIndex(0, 0, 1) // Set the pixel to the second color in the palette (red)
		finalImg = palettedImg
	} else {
		// For PNG/JPEG, create an RGBA image
		rgbaImg := image.NewRGBA(rect)
		rgbaImg.Set(0, 0, color.RGBA{R: 255, A: 255}) // Red pixel
		finalImg = rgbaImg
	}

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test image file %s: %v", filePath, err)
	}
	defer file.Close()

	switch strings.ToLower(format) {
	case "png":
		if err := png.Encode(file, finalImg); err != nil {
			t.Fatalf("Failed to encode test PNG %s: %v", filePath, err)
		}
	case "jpeg", "jpg":
		if err := jpeg.Encode(file, finalImg, nil); err != nil { // jpeg.Options can be nil for default
			t.Fatalf("Failed to encode test JPEG %s: %v", filePath, err)
		}
	case "gif":
		if err := gif.Encode(file, finalImg, &gif.Options{NumColors: 256}); err != nil {
			t.Fatalf("Failed to encode test GIF %s: %v", filePath, err)
		}
	default:
		t.Fatalf("Unsupported test image format: %s", format)
	}
	return filePath
}

// Helper to create a file with specific content
func createTestFile(t *testing.T, dir string, filename string, content []byte) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", filePath, err)
	}
	return filePath
}


func checkFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it does not", path)
	} else if err != nil {
		t.Errorf("Error checking file %s: %v", path, err)
	}
}

func checkFileDoesNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file %s to not exist, but it does", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("Error checking file %s (expected IsNotExist): %v", path, err)
	}
}

func findMessage(messages []string, substr string) bool {
	for _, msg := range messages {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

func TestIntegration_ConvertDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_integration_input_*")
	if err != nil {
		t.Fatalf("Failed to create temp input dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Standard images
	createIntegrationTestImage(t, tmpDir, "image1.png", "png")
	createIntegrationTestImage(t, tmpDir, "image2.jpg", "jpeg")
	createIntegrationTestImage(t, tmpDir, "image3.gif", "gif")

	// Text file with standard extension
	txtFilePath := createTestFile(t, tmpDir, "document.txt", []byte("this is a plain text document"))

	// Valid PNG image named image.txt
	// Create a minimal valid PNG byte slice
	var pngBytes []byte
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{0, 255, 0, 255}) // Green pixel
	buf := new(strings.Builder) // Use strings.Builder as a temporary io.Writer
	byteWriter := &byteWriter{buf}
	if err := png.Encode(byteWriter, img); err != nil {
		t.Fatalf("Failed to encode in-memory PNG: %v", err)
	}
	pngBytes = []byte(byteWriter.String()) // Get bytes from strings.Builder
	imageTxtPath := createTestFile(t, tmpDir, "image.txt", pngBytes)


	// Text file named document.jpg
	docJPEGPath := createTestFile(t, tmpDir, "document.jpg", []byte("this is plain text, not a jpeg"))


	messages, err := runApp(tmpDir, false)
	if err != nil {
		t.Fatalf("runApp failed: %v. Messages: %v", err, messages)
	}

	// Print messages for debugging
	// for _, msg := range messages {
	// 	fmt.Println(msg)
	// }

	// Check standard conversions
	checkFileExists(t, filepath.Join(tmpDir, "image1.webp"))
	checkFileExists(t, filepath.Join(tmpDir, "image2.webp"))
	checkFileExists(t, filepath.Join(tmpDir, "image3.webp"))

	// Check skipping of actual text file
	checkFileDoesNotExist(t, filepath.Join(tmpDir, "document.webp"))
	if !findMessage(messages, "INFO: Skipping file "+txtFilePath+" (detected MIME type: text/plain") {
		t.Errorf("Missing or incorrect skip message for document.txt. Messages: %v", messages)
	}

	// Check conversion of PNG file named image.txt
	checkFileExists(t, filepath.Join(tmpDir, "image.webp")) // Output should be image.webp not image.txt.webp
	if !findMessage(messages, "INFO: Successfully converted "+imageTxtPath) {
		t.Errorf("Missing success message for %s (PNG disguised as .txt). Messages: %v", imageTxtPath, messages)
	}
	if !findMessage(messages, "INFO: File: "+imageTxtPath+", Detected MIME type: image/png") {
		t.Errorf("Missing image/png MIME type detection message for %s. Messages: %v", imageTxtPath, messages)
	}


	// Check skipping of text file named document.jpg
	checkFileDoesNotExist(t, filepath.Join(tmpDir, "document.webp")) // Output should be document.webp
	if !findMessage(messages, "INFO: Skipping file "+docJPEGPath+" (detected MIME type: text/plain") {
		t.Errorf("Missing or incorrect skip message for %s (text disguised as .jpg). Messages: %v", docJPEGPath, messages)
	}
}

// byteWriter to satisfy io.Writer for png.Encode with a strings.Builder
type byteWriter struct {
	*strings.Builder
}

func (bw *byteWriter) Write(p []byte) (int, error) {
	return bw.WriteString(string(p))
}

func TestIntegration_ForceOverwrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_force_input_*")
	if err != nil {
		t.Fatalf("Failed to create temp input dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pngPath := createIntegrationTestImage(t, tmpDir, "image.png", "png")
	webpPath := filepath.Join(tmpDir, "image.webp")

	// First run, create .webp
	messages, errRun1 := runApp(tmpDir, false)
	if errRun1 != nil {
		t.Fatalf("runApp (1st run) failed: %v. Messages: %v", errRun1, messages)
	}
	checkFileExists(t, webpPath)
	if !findMessage(messages, "INFO: Successfully converted "+pngPath) {
		t.Error("Missing success message for image.png on 1st run")
	}
	stat1, _ := os.Stat(webpPath)
	time.Sleep(10 * time.Millisecond) // Ensure mod time can change if file is rewritten

	// Second run, no force, should skip
	messages, errRun2 := runApp(tmpDir, false)
	if errRun2 != nil {
		t.Fatalf("runApp (2nd run, no force) failed: %v. Messages: %v", errRun2, messages)
	}
	expectedSkipMessage := fmt.Sprintf("INFO: Skipping conversion (file exists, based on content type): %s", webpPath)
	if !findMessage(messages, expectedSkipMessage) {
		t.Errorf("Expected skip message '%s' on 2nd run (no force), got: %v", expectedSkipMessage, messages)
	}
	stat2, _ := os.Stat(webpPath)
	if stat1.ModTime() != stat2.ModTime() {
		// This can be flaky on some filesystems with low mtime resolution.
		// The message check is more reliable for skipping.
		t.Logf("Warning: ModTime changed on no-force run. Stat1: %s, Stat2: %s. This might be a filesystem artifact.", stat1.ModTime(), stat2.ModTime())
	}


	// Third run, with force, should overwrite
	messages, errRun3 := runApp(tmpDir, true)
	if errRun3 != nil {
		t.Fatalf("runApp (3rd run, with force) failed: %v. Messages: %v", errRun3, messages)
	}
	if !findMessage(messages, "INFO: Successfully converted "+pngPath) {
		t.Errorf("Missing success message for image.png on 3rd run (with force), got messages: %v", messages)
	}
	stat3, _ := os.Stat(webpPath)
	if stat1.ModTime() == stat3.ModTime() && stat1.Size() == stat3.Size() {
		 // If both modtime and size are same, it likely wasn't overwritten.
		 // Content check would be more robust but harder here.
		t.Errorf("Expected ModTime or Size to change on force overwrite. Stat1_Mod: %s, Stat3_Mod: %s. Stat1_Size: %d, Stat3_Size: %d",
			stat1.ModTime(), stat3.ModTime(), stat1.Size(), stat3.Size())
	}
}

func TestIntegration_SingleFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_single_input_*")
	if err != nil {
		t.Fatalf("Failed to create temp input dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pngPath := createIntegrationTestImage(t, tmpDir, "single.png", "png")
	expectedWebpPath := filepath.Join(tmpDir, "single.webp")

	messages, errRun := runApp(pngPath, false) // Pass the direct file path
	if errRun != nil {
		t.Fatalf("runApp failed for single file: %v. Messages: %v", errRun, messages)
	}

	// Print messages for debugging
	for _, msg := range messages {
		fmt.Println(msg)
	}

	checkFileExists(t, expectedWebpPath)
	if !findMessage(messages, "INFO: Successfully converted "+pngPath) {
		t.Error("Missing success message for single.png")
	}
}

func TestIntegration_PathNonExistent(t *testing.T) {
	nonExistentPath := filepath.Join("completely", "made", "up", "path", "image.png")
	// Ensure it really doesn't exist or make it unique
	_ = os.RemoveAll(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(nonExistentPath)))))


	messages, err := runApp(nonExistentPath, false)
	if err == nil {
		t.Fatalf("Expected runApp to return an error for non-existent path, got nil. Messages: %v", messages)
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error message to contain 'does not exist', got: %v", err.Error())
	}
}

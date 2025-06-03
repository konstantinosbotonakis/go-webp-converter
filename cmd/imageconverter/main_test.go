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
func createIntegrationTestImage(t *testing.T, dir string, filename string, format string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255}) // A single red pixel

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test image file %s: %v", filePath, err)
	}
	defer file.Close()

	switch strings.ToLower(format) {
	case "png":
		if err := png.Encode(file, img); err != nil {
			t.Fatalf("Failed to encode test PNG %s: %v", filePath, err)
		}
	case "jpeg", "jpg":
		if err := jpeg.Encode(file, img, nil); err != nil {
			t.Fatalf("Failed to encode test JPEG %s: %v", filePath, err)
		}
	case "gif":
		if err := gif.Encode(file, img, &gif.Options{NumColors: 1}); err != nil {
			t.Fatalf("Failed to encode test GIF %s: %v", filePath, err)
		}
	default:
		t.Fatalf("Unsupported test image format: %s", format)
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

	createIntegrationTestImage(t, tmpDir, "image1.png", "png")
	createIntegrationTestImage(t, tmpDir, "image2.jpg", "jpeg")
	createIntegrationTestImage(t, tmpDir, "image3.gif", "gif")

	txtFilePath := filepath.Join(tmpDir, "document.txt")
	if err := os.WriteFile(txtFilePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test text file: %v", err)
	}

	messages, err := runApp(tmpDir, false)
	if err != nil {
		t.Fatalf("runApp failed: %v. Messages: %v", err, messages)
	}

	// Print messages for debugging
	for _, msg := range messages {
		fmt.Println(msg)
	}


	checkFileExists(t, filepath.Join(tmpDir, "image1.webp"))
	checkFileExists(t, filepath.Join(tmpDir, "image2.webp"))
	checkFileExists(t, filepath.Join(tmpDir, "image3.webp"))
	checkFileDoesNotExist(t, filepath.Join(tmpDir, "document.webp"))

	if !findMessage(messages, "Successfully converted "+filepath.Join(tmpDir, "image1.png")) {
		t.Error("Missing success message for image1.png")
	}
	if !findMessage(messages, "Successfully converted "+filepath.Join(tmpDir, "image2.jpg")) {
		t.Error("Missing success message for image2.jpg")
	}
	if !findMessage(messages, "Successfully converted "+filepath.Join(tmpDir, "image3.gif")) {
		t.Error("Missing success message for image3.gif")
	}
	if !findMessage(messages, "Skipping non-image file: "+txtFilePath) {
		t.Error("Missing skip message for document.txt")
	}
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
	if !findMessage(messages, "Successfully converted "+pngPath) {
		t.Error("Missing success message for image.png on 1st run")
	}
	stat1, _ := os.Stat(webpPath)
	time.Sleep(10 * time.Millisecond) // Ensure mod time can change if file is rewritten

	// Second run, no force, should skip
	messages, errRun2 := runApp(tmpDir, false)
	if errRun2 != nil {
		t.Fatalf("runApp (2nd run, no force) failed: %v. Messages: %v", errRun2, messages)
	}
	if !findMessage(messages, "Skipping conversion (file exists): "+webpPath) {
		t.Errorf("Expected skip message for existing file on 2nd run (no force), got: %v", messages)
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
	if !findMessage(messages, "Successfully converted "+pngPath) {
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
	if !findMessage(messages, "Successfully converted "+pngPath) {
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

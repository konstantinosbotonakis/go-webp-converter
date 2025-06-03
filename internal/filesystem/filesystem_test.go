package filesystem_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"imageconverter/internal/filesystem"
)

func TestFindFiles_SingleFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	files, err := filesystem.FindFiles(tmpFile.Name())
	if err != nil {
		t.Fatalf("FindFiles returned an error for a single file: %v", err)
	}

	expected := []string{tmpFile.Name()}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("Expected FindFiles to return %v, got %v", expected, files)
	}
}

func TestFindFiles_Directory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testdir*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1Path := filepath.Join(tmpDir, "file1.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	file2Path := filepath.Join(subDir, "file2.txt")

	if err := os.WriteFile(file1Path, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(file2Path, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Create a symlink to a regular file
	symlinkPath := filepath.Join(tmpDir, "symlinkfile.txt")
	if err := os.Symlink(file1Path, symlinkPath); err != nil {
		// Symlinks might fail on some systems (e.g. Windows without developer mode)
		// We can skip this part of the test if symlink creation fails
		t.Logf("Skipping symlink test part: could not create symlink: %v", err)
	} else {
		defer os.Remove(symlinkPath) // Ensure cleanup if symlink was created
	}

	// Create a symlink to a directory - should not be followed by WalkDir by default
	symlinkDirPath := filepath.Join(tmpDir, "symlinkdir")
	if err := os.Symlink(subDir, symlinkDirPath); err != nil {
		t.Logf("Skipping symlink to dir test part: could not create symlink: %v", err)
	} else {
		defer os.Remove(symlinkDirPath)
	}


	files, err := filesystem.FindFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindFiles returned an error for a directory: %v", err)
	}

	expectedFiles := []string{file1Path, file2Path}
	// Check if symlink was created and add to expected if so
	// The FindFiles function should resolve and deduplicate symlinks.
	// So, if symlinkfile.txt points to file1.txt, only file1.txt (resolved path) should be in the results.
	// The initial expectedFiles already includes file1.txt.
	// No need to add symlinkPath or its resolved path again if it points to an already listed file.

	// Create a unique set of expected files
	expectedMap := make(map[string]struct{})
	for _, f := range expectedFiles {
		expectedMap[f] = struct{}{}
	}
	// If symlink was created and resolves to a new file not already in expectedFiles, add it.
	// However, our current symlink points to file1.txt which is already expected.
	// If we had another symlink, say symlink_to_new_file.txt -> new_actual_file.txt,
	// then new_actual_file.txt would be added.
	// For this test, since symlinkPath points to file1Path, no new unique path is added.

	// Rebuild expectedFiles from the map to ensure uniqueness, matching FindFiles behavior
	uniqueExpectedFiles := []string{}
	for f := range expectedMap {
		uniqueExpectedFiles = append(uniqueExpectedFiles, f)
	}


	sort.Strings(files)
	sort.Strings(uniqueExpectedFiles) // Sort the unique list

	if !reflect.DeepEqual(files, uniqueExpectedFiles) {
		t.Errorf("Expected FindFiles to return %v (unique, sorted), got %v (sorted)", uniqueExpectedFiles, files)
	}
}

func TestFindFiles_NonExistentPath(t *testing.T) {
	nonExistentPath := filepath.Join(" совершенно", "несуществующий", "путь", "file.txt")
	if _, err := os.Stat(nonExistentPath); !os.IsNotExist(err) {
		// If the path somehow exists, skip the test or ensure it's truly unique
		t.Skipf("Skipping test as non-existent path '%s' surprisingly exists or cannot be confirmed as non-existent.", nonExistentPath)
		return
	}

	_, err := filesystem.FindFiles(nonExistentPath)
	if err == nil {
		t.Errorf("Expected FindFiles to return an error for a non-existent path, but got nil")
	}
}

func TestFindFiles_SingleSymlinkToFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testsymtarget*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	symlinkPath := filepath.Join(filepath.Dir(tmpFile.Name()), "symlinktotestfile.txt")
	if err := os.Symlink(tmpFile.Name(), symlinkPath); err != nil {
		t.Skipf("Skipping symlink test: could not create symlink: %v", err)
		return
	}
	defer os.Remove(symlinkPath)

	files, err := filesystem.FindFiles(symlinkPath)
	if err != nil {
		t.Fatalf("FindFiles returned an error for a single symlink to file: %v", err)
	}

	expected := []string{tmpFile.Name()} // Expecting the resolved path
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("Expected FindFiles to return %v for symlink, got %v", expected, files)
	}
}

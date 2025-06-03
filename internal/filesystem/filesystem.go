package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FindFiles recursively finds all regular files in the given inputPath.
// If inputPath is a file, it returns a slice containing only that path.
// If inputPath is a directory, it walks the directory and returns paths to all regular files.
// Symbolic links to files are followed and their target paths are returned.
func FindFiles(inputPath string) ([]string, error) {
	info, err := os.Lstat(inputPath) // Use Lstat to get info about the link itself
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", inputPath, err)
	}

	// If inputPath is a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := filepath.EvalSymlinks(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve symlink %s: %w", inputPath, err)
		}
		// After resolving, get info about the target
		info, err = os.Stat(resolvedPath) // Stat the resolved path
		if err != nil {
			return nil, fmt.Errorf("failed to get file info for resolved symlink target %s (link: %s): %w", resolvedPath, inputPath, err)
		}
		if info.Mode().IsRegular() {
			return []string{resolvedPath}, nil
		}
		return []string{}, nil // Symlink does not point to a regular file
	}

	// If inputPath is a regular file (and not a symlink)
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return []string{inputPath}, nil
		}
		return []string{}, nil // Not a regular file
	}

	// If inputPath is a directory
	var files []string
	err = filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Skip files that cause errors (e.g. permission issues)
			// This error is from the function passed to WalkDir.
			// We want to collect files even if some paths are inaccessible.
			// So, we log it (or could add to a list of errors) and continue.
			// For now, just returning nil to continue the walk.
			// If we wanted to signal this up, we'd need a different mechanism.
			// fmt.Printf("Warning: error accessing %s: %v\n", path, walkErr) // Example logging
			return nil // Continue walking even if a path is problematic.
		}

		entryType := d.Type()

		if entryType.IsRegular() {
			files = append(files, path)
		} else if entryType&fs.ModeSymlink != 0 {
			resolvedPath, errEval := filepath.EvalSymlinks(path)
			if errEval != nil {
				// fmt.Printf("Warning: error evaluating symlink %s: %v\n", path, errEval)
				return nil // Skip broken or problematic symlinks
			}
			// Check if the resolved path points to a regular file
			resolvedInfo, errStat := os.Stat(resolvedPath)
			if errStat != nil {
				// fmt.Printf("Warning: error stating resolved symlink %s (target %s): %v\n", path, resolvedPath, errStat)
				return nil // Skip if cannot stat resolved path
			}
			if resolvedInfo.Mode().IsRegular() {
				// Check if the file already exists in the list to avoid duplicates
				// (e.g. if symlink points within the same directory being walked)
				alreadyFound := false
				for _, f := range files {
					if f == resolvedPath {
						alreadyFound = true
						break
					}
				}
				if !alreadyFound {
					files = append(files, resolvedPath)
				}
			}
		}
		return nil
	})

	// This 'err' variable here is from the assignment `err = filepath.WalkDir(...)`
	// It will be non-nil if the WalkDirFunc returns an error, thus aborting the walk.
	// If WalkDirFunc always returns nil (even on path errors it handles by skipping),
	// then this err will be nil.
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", inputPath, err)
	}

	return files, nil
}

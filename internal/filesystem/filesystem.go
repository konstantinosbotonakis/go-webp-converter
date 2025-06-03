package filesystem

import (
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
		return nil, err
	}

	// If inputPath is a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := filepath.EvalSymlinks(inputPath)
		if err != nil {
			return nil, err // Error resolving symlink
		}
		// After resolving, get info about the target
		info, err = os.Stat(resolvedPath) // Stat the resolved path
		if err != nil {
			return nil, err // Error stating resolved path
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
			// but continue walking the directory.
			// Log the error or handle it as needed. For now, just skip.
			// fmt.Fprintf(os.Stderr, "Error accessing path %q: %v\n", path, walkErr)
			return nil
		}

		entryType := d.Type()

		if entryType.IsRegular() {
			files = append(files, path)
		} else if entryType&fs.ModeSymlink != 0 {
			resolvedPath, symlinkErr := filepath.EvalSymlinks(path)
			if symlinkErr != nil {
				// Skip broken or problematic symlinks
				return nil
			}
			// Check if the resolved path points to a regular file
			resolvedInfo, statErr := os.Stat(resolvedPath)
			if statErr != nil {
				// Skip if cannot stat resolved path
				return nil
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

	if err != nil {
		return nil, err
	}

	return files, nil
}

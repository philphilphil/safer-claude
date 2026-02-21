package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// syncBack walks the temp directory and syncs edited/new files back to the
// original location. Returns true if the temp dir should be kept (conflicts).
func syncBack(tmpDir, originalBase string, manifest map[string]string) bool {
	var synced, skipped, newFiles, conflicts int
	keepTmp := false

	filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(tmpDir, path)
		if err != nil || rel == "." {
			return nil
		}

		// Skip hidden directories created by Claude (like .claude)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		// Skip junk files
		if skipFiles[d.Name()] {
			return nil
		}

		tmpHash, err := hashFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not hash temp file %s: %v\n", rel, err)
			return nil
		}

		origHash, inManifest := manifest[rel]
		origPath := filepath.Join(originalBase, rel)

		if inManifest {
			// Existing file — check if Claude edited it
			if tmpHash == origHash {
				skipped++
				return nil
			}

			// Claude edited this file — check for external conflicts
			currentOrigHash, err := hashFile(origPath)
			if err != nil {
				// Original was deleted externally
				fmt.Printf("  CONFLICT: %s was deleted externally but edited in session\n", rel)
				fmt.Printf("           Temp copy: %s\n", path)
				conflicts++
				keepTmp = true
				return nil
			}

			if currentOrigHash == origHash {
				// No external edits — safe to overwrite
				if err := os.MkdirAll(filepath.Dir(origPath), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "  ERROR: could not create dir for %s: %v\n", rel, err)
					return nil
				}
				if err := copyFile(path, origPath); err != nil {
					fmt.Fprintf(os.Stderr, "  ERROR: could not sync %s: %v\n", rel, err)
					return nil
				}
				fmt.Printf("  SYNCED: %s\n", rel)
				synced++
			} else {
				// External edits detected — conflict
				fmt.Printf("  CONFLICT: %s was modified both externally and in session\n", rel)
				fmt.Printf("           Temp copy: %s\n", path)
				conflicts++
				keepTmp = true
			}
		} else {
			// New file created by Claude
			if err := os.MkdirAll(filepath.Dir(origPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR: could not create dir for %s: %v\n", rel, err)
				return nil
			}
			if err := copyFile(path, origPath); err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR: could not sync new file %s: %v\n", rel, err)
				return nil
			}
			fmt.Printf("  NEW: %s\n", rel)
			newFiles++
		}

		return nil
	})

	// Check for deleted files
	for rel := range manifest {
		tmpPath := filepath.Join(tmpDir, rel)
		if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
			fmt.Printf("  DELETED in session (original kept): %s\n", rel)
		}
	}

	fmt.Printf("\nSync complete: %d synced, %d new, %d unchanged, %d conflicts\n", synced, newFiles, skipped, conflicts)
	if keepTmp {
		fmt.Printf("Temp directory kept due to conflicts: %s\n", tmpDir)
	}

	return keepTmp
}

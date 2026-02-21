package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skipFiles contains filenames to skip when copying.
var skipFiles = map[string]bool{
	".DS_Store":  true,
	"Thumbs.db":  true,
	"desktop.ini": true,
}

// hashFile computes the SHA-256 hex digest of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies a single file preserving permissions.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyToTemp copies the target (file or directory) into tmpDir and returns
// a manifest mapping relative paths to their SHA-256 hashes.
func copyToTemp(absTarget string, info fs.FileInfo, tmpDir string) (map[string]string, error) {
	manifest := make(map[string]string)

	if !info.IsDir() {
		// Single file: copy it into tmpDir with same name
		name := filepath.Base(absTarget)
		dst := filepath.Join(tmpDir, name)

		if err := copyFile(absTarget, dst); err != nil {
			return nil, err
		}

		hash, err := hashFile(absTarget)
		if err != nil {
			return nil, err
		}
		manifest[name] = hash
		return manifest, nil
	}

	// Directory: recursively copy
	return manifest, filepath.WalkDir(absTarget, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(absTarget, path)
		if err != nil {
			return err
		}

		// Skip hidden directories (like .git, .obsidian)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && rel != "." {
			return filepath.SkipDir
		}

		// Skip junk files
		if !d.IsDir() && skipFiles[d.Name()] {
			return nil
		}

		dst := filepath.Join(tmpDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		// Ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		if err := copyFile(path, dst); err != nil {
			return err
		}

		hash, err := hashFile(path)
		if err != nil {
			return err
		}
		manifest[rel] = hash
		return nil
	})
}

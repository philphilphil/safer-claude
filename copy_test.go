package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello world\n"), 0644)

	hash, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}

	// SHA-256 of "hello world\n"
	want := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	if hash != want {
		t.Errorf("got %s, want %s", hash, want)
	}
}

func TestHashFile_DifferentContent(t *testing.T) {
	tmp := t.TempDir()

	f1 := filepath.Join(tmp, "a.txt")
	f2 := filepath.Join(tmp, "b.txt")
	os.WriteFile(f1, []byte("aaa"), 0644)
	os.WriteFile(f2, []byte("bbb"), 0644)

	h1, _ := hashFile(f1)
	h2, _ := hashFile(f2)

	if h1 == h2 {
		t.Error("different files should produce different hashes")
	}
}

func TestHashFile_Nonexistent(t *testing.T) {
	_, err := hashFile("/nonexistent/file")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	content := []byte("copy me")
	os.WriteFile(src, content, 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.sh")
	dst := filepath.Join(tmp, "dst.sh")

	os.WriteFile(src, []byte("#!/bin/sh"), 0755)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	info, _ := os.Stat(dst)
	if info.Mode().Perm() != 0755 {
		t.Errorf("got perm %o, want 0755", info.Mode().Perm())
	}
}

func TestCopyToTemp_SingleFile(t *testing.T) {
	src := t.TempDir()
	tmp := t.TempDir()

	file := filepath.Join(src, "note.md")
	os.WriteFile(file, []byte("# Hello"), 0644)

	info, _ := os.Stat(file)
	manifest, err := copyToTemp(file, info, tmp)
	if err != nil {
		t.Fatalf("copyToTemp: %v", err)
	}

	if len(manifest) != 1 {
		t.Fatalf("expected 1 manifest entry, got %d", len(manifest))
	}

	if _, ok := manifest["note.md"]; !ok {
		t.Error("manifest missing 'note.md'")
	}

	// Verify file was copied
	got, _ := os.ReadFile(filepath.Join(tmp, "note.md"))
	if string(got) != "# Hello" {
		t.Errorf("got %q, want %q", got, "# Hello")
	}
}

func TestCopyToTemp_Directory(t *testing.T) {
	src := t.TempDir()
	tmp := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(src, "sub"), 0755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("bbb"), 0644)

	info, _ := os.Stat(src)
	manifest, err := copyToTemp(src, info, tmp)
	if err != nil {
		t.Fatalf("copyToTemp: %v", err)
	}

	if len(manifest) != 2 {
		t.Fatalf("expected 2 manifest entries, got %d", len(manifest))
	}

	if _, ok := manifest["a.txt"]; !ok {
		t.Error("manifest missing 'a.txt'")
	}
	if _, ok := manifest[filepath.Join("sub", "b.txt")]; !ok {
		t.Error("manifest missing 'sub/b.txt'")
	}

	// Verify files were copied
	got, _ := os.ReadFile(filepath.Join(tmp, "sub", "b.txt"))
	if string(got) != "bbb" {
		t.Errorf("got %q, want %q", got, "bbb")
	}
}

func TestCopyToTemp_SkipsHiddenDirs(t *testing.T) {
	src := t.TempDir()
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(src, ".git"), 0755)
	os.MkdirAll(filepath.Join(src, ".obsidian"), 0755)
	os.WriteFile(filepath.Join(src, ".git", "config"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(src, ".obsidian", "app.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(src, "visible.txt"), []byte("hi"), 0644)

	info, _ := os.Stat(src)
	manifest, err := copyToTemp(src, info, tmp)
	if err != nil {
		t.Fatalf("copyToTemp: %v", err)
	}

	if len(manifest) != 1 {
		t.Errorf("expected 1 entry (visible.txt), got %d", len(manifest))
	}

	if _, err := os.Stat(filepath.Join(tmp, ".git")); !os.IsNotExist(err) {
		t.Error(".git should not be copied")
	}
}

func TestCopyToTemp_SkipsJunkFiles(t *testing.T) {
	src := t.TempDir()
	tmp := t.TempDir()

	os.WriteFile(filepath.Join(src, ".DS_Store"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(src, "Thumbs.db"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(src, "desktop.ini"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(src, "real.txt"), []byte("keep"), 0644)

	info, _ := os.Stat(src)
	manifest, err := copyToTemp(src, info, tmp)
	if err != nil {
		t.Fatalf("copyToTemp: %v", err)
	}

	if len(manifest) != 1 {
		t.Errorf("expected 1 entry, got %d", len(manifest))
	}

	if _, ok := manifest["real.txt"]; !ok {
		t.Error("manifest missing 'real.txt'")
	}
}

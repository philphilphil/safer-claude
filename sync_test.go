package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupSyncTest(t *testing.T) (tmpDir, origDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	origDir = t.TempDir()
	return
}

func TestSyncBack_UnchangedFile(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	content := []byte("unchanged")
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), content, 0644)
	os.WriteFile(filepath.Join(origDir, "a.txt"), content, 0644)

	hash, _ := hashFile(filepath.Join(origDir, "a.txt"))
	manifest := map[string]string{"a.txt": hash}

	kept := syncBack(tmpDir, origDir, manifest)
	if kept {
		t.Error("should not keep tmp dir when no conflicts")
	}
}

func TestSyncBack_EditedFile(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	os.WriteFile(filepath.Join(origDir, "a.txt"), []byte("original"), 0644)
	hash, _ := hashFile(filepath.Join(origDir, "a.txt"))
	manifest := map[string]string{"a.txt": hash}

	// Simulate Claude editing the file in tmp
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("edited by claude"), 0644)

	kept := syncBack(tmpDir, origDir, manifest)
	if kept {
		t.Error("should not keep tmp dir when no conflicts")
	}

	// Original should now have the edited content
	got, _ := os.ReadFile(filepath.Join(origDir, "a.txt"))
	if string(got) != "edited by claude" {
		t.Errorf("got %q, want %q", got, "edited by claude")
	}
}

func TestSyncBack_NewFile(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	// Claude created a new file in tmp
	os.WriteFile(filepath.Join(tmpDir, "new.txt"), []byte("brand new"), 0644)
	manifest := map[string]string{} // empty manifest = no original files

	kept := syncBack(tmpDir, origDir, manifest)
	if kept {
		t.Error("should not keep tmp dir when no conflicts")
	}

	got, _ := os.ReadFile(filepath.Join(origDir, "new.txt"))
	if string(got) != "brand new" {
		t.Errorf("got %q, want %q", got, "brand new")
	}
}

func TestSyncBack_NewFileInSubdir(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "new.txt"), []byte("nested new"), 0644)
	manifest := map[string]string{}

	syncBack(tmpDir, origDir, manifest)

	got, _ := os.ReadFile(filepath.Join(origDir, "sub", "new.txt"))
	if string(got) != "nested new" {
		t.Errorf("got %q, want %q", got, "nested new")
	}
}

func TestSyncBack_ConflictBothEdited(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	// Original content at copy time
	os.WriteFile(filepath.Join(origDir, "a.txt"), []byte("original"), 0644)
	hash, _ := hashFile(filepath.Join(origDir, "a.txt"))
	manifest := map[string]string{"a.txt": hash}

	// Claude edits in tmp
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("claude edit"), 0644)

	// External edit to original (simulates user editing during session)
	os.WriteFile(filepath.Join(origDir, "a.txt"), []byte("external edit"), 0644)

	kept := syncBack(tmpDir, origDir, manifest)
	if !kept {
		t.Error("should keep tmp dir when conflicts exist")
	}

	// Original should keep external edit (not overwritten)
	got, _ := os.ReadFile(filepath.Join(origDir, "a.txt"))
	if string(got) != "external edit" {
		t.Errorf("original should not be overwritten, got %q", got)
	}
}

func TestSyncBack_ConflictDeletedExternally(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	// File existed at copy time
	origFile := filepath.Join(origDir, "a.txt")
	os.WriteFile(origFile, []byte("original"), 0644)
	hash, _ := hashFile(origFile)
	manifest := map[string]string{"a.txt": hash}

	// Claude edits in tmp
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("claude edit"), 0644)

	// File deleted externally
	os.Remove(origFile)

	kept := syncBack(tmpDir, origDir, manifest)
	if !kept {
		t.Error("should keep tmp dir when file was deleted externally but edited in session")
	}
}

func TestSyncBack_SkipsHiddenDirs(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	// Simulate .claude directory created by Claude
	os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".claude", "settings.json"), []byte("{}"), 0644)
	manifest := map[string]string{}

	syncBack(tmpDir, origDir, manifest)

	// .claude dir should NOT be synced to original
	if _, err := os.Stat(filepath.Join(origDir, ".claude")); !os.IsNotExist(err) {
		t.Error(".claude directory should not be synced back")
	}
}

func TestSyncBack_SkipsJunkFiles(t *testing.T) {
	tmpDir, origDir := setupSyncTest(t)

	os.WriteFile(filepath.Join(tmpDir, ".DS_Store"), []byte("x"), 0644)
	manifest := map[string]string{}

	syncBack(tmpDir, origDir, manifest)

	if _, err := os.Stat(filepath.Join(origDir, ".DS_Store")); !os.IsNotExist(err) {
		t.Error(".DS_Store should not be synced back")
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteClaudeSettings(t *testing.T) {
	tmp := t.TempDir()

	if err := writeClaudeSettings(tmp); err != nil {
		t.Fatalf("writeClaudeSettings: %v", err)
	}

	settingsPath := filepath.Join(tmp, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(settings.Permissions.Deny) == 0 {
		t.Error("expected deny rules, got none")
	}

	// Check that key deny rules are present
	denySet := make(map[string]bool)
	for _, d := range settings.Permissions.Deny {
		denySet[d] = true
	}

	mustDeny := []string{
		"Read(../)",
		"Read(../../**)",
		"Edit(../)",
		"Edit(../../**)",
	}
	for _, rule := range mustDeny {
		if !denySet[rule] {
			t.Errorf("missing deny rule: %s", rule)
		}
	}
}

func TestWriteClaudeSettings_CreatesDir(t *testing.T) {
	tmp := t.TempDir()
	writeClaudeSettings(tmp)

	info, err := os.Stat(filepath.Join(tmp, ".claude"))
	if err != nil {
		t.Fatalf("expected .claude dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error(".claude should be a directory")
	}
}

func TestWriteClaudeSettings_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	writeClaudeSettings(tmp)

	data, _ := os.ReadFile(filepath.Join(tmp, ".claude", "settings.json"))

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type claudePermissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type claudeSettings struct {
	Permissions claudePermissions `json:"permissions"`
}

// writeClaudeSettings creates .claude/settings.json in tmpDir that denies
// Claude access to anything outside the temp directory.
func writeClaudeSettings(tmpDir string) error {
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	settings := claudeSettings{
		Permissions: claudePermissions{
			Allow: []string{},
			Deny: []string{
				"Read(../)",
				"Read(../../**)",
				"Edit(../)",
				"Edit(../../**)",
				"Bash(cat *)",
				"Bash(head *)",
				"Bash(tail *)",
				"Bash(less *)",
				"Bash(more *)",
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)
}

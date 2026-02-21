package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: safer-claude <file-or-folder>\n")
		os.Exit(1)
	}

	target := os.Args[1]

	// Resolve to absolute path
	absTarget, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Verify target exists
	info, err := os.Stat(absTarget)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check that claude is in PATH
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: 'claude' not found in PATH.\nInstall Claude Code: https://docs.anthropic.com/en/docs/claude-code\n")
		os.Exit(1)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safer-claude-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp directory: %v\n", err)
		os.Exit(1)
	}

	// Copy files and build manifest
	manifest, err := copyToTemp(absTarget, info, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "Error copying files: %v\n", err)
		os.Exit(1)
	}

	// Write .claude/settings.json to restrict file access to the temp dir
	if err := writeClaudeSettings(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "Error writing Claude settings: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copied to %s\n", tmpDir)
	fmt.Printf("Launching Claude Code...\n\n")

	// Launch claude in the temp dir
	cmd := exec.Command(claudePath)
	cmd.Dir = tmpDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	claudeErr := cmd.Run()
	if claudeErr != nil {
		fmt.Fprintf(os.Stderr, "\nClaude exited with error: %v\n", claudeErr)
		fmt.Fprintf(os.Stderr, "Attempting to sync any changes...\n\n")
	}

	// Determine the base directory in the original for syncing back
	var originalBase string
	if info.IsDir() {
		originalBase = absTarget
	} else {
		originalBase = filepath.Dir(absTarget)
	}

	// Sync changes back
	kept := syncBack(tmpDir, originalBase, manifest)

	// Clean up temp dir if no conflicts left files there
	if !kept {
		os.RemoveAll(tmpDir)
	}
}

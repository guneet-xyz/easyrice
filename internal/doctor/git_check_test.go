package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckGitOnPath(t *testing.T) {
	if err := CheckGitOnPath(); err != nil {
		t.Fatalf("expected git on PATH, got error: %v", err)
	}
}

func TestCheckGitOnPath_GitPresent(t *testing.T) {
	// Create a fake git script in a temp directory
	tmpDir := t.TempDir()
	gitPath := filepath.Join(tmpDir, "git")

	// Write a simple shell script that exits 0
	err := os.WriteFile(gitPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to write fake git script: %v", err)
	}

	// Prepend tmpDir to PATH so exec.LookPath finds our fake git
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// CheckGitOnPath should succeed
	if err := CheckGitOnPath(); err != nil {
		t.Errorf("expected no error with git on PATH, got: %v", err)
	}
}

func TestCheckGitOnPath_GitMissing(t *testing.T) {
	// Set PATH to a temp directory with no git binary
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)

	// CheckGitOnPath should fail
	err := CheckGitOnPath()
	if err == nil {
		t.Error("expected error when git is not on PATH, got nil")
	}
}

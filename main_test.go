package main

import (
	"os/exec"
	"testing"
)

func TestMainRuns(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start main.go: %v", err)
	}
	// Optionally, wait a short time and kill the process
	// to avoid hanging the test suite
	cmd.Process.Kill()
}

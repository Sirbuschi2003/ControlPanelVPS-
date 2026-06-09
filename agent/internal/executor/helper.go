package executor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

// readFileSafe reads a file without path restriction checks (internal use only).
func readFileSafe(path string) ([]byte, error) {
	return os.ReadFile(filepath.Clean(path))
}

// writeFileSafe writes content to a file (creates parent dirs if needed).
func writeFileSafe(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), content, 0644)
}

// runCmdInputOutput runs a command with stdin input and returns combined output.
func runCmdInputOutput(stdin string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewBufferString(stdin)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

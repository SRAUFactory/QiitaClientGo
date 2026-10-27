package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestMainCommandExecution(t *testing.T) {
	// Build the binary in temporary location
	cmd := exec.Command("go", "build", "-o", "qiita-client-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	defer func() {
		_ = exec.Command("rm", "-f", "qiita-client-test").Run()
	}()

	// Test --version
	out, err := exec.Command("./qiita-client-test", "version").CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(string(out), "qiita-client version") {
		t.Errorf("unexpected version output: %s", string(out))
	}

	// Test help
	out, err = exec.Command("./qiita-client-test", "help").CombinedOutput()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	if !strings.Contains(string(out), "Available Commands:") {
		t.Errorf("unexpected help output: %s", string(out))
	}
}

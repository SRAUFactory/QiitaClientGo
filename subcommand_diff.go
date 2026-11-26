package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunDiffCommand executes the diff subcommand.
func RunDiffCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var dir string
	var base string
	fs.StringVar(&dir, "dir", "qiita", "Target directory to monitor (e.g. qiita)")
	fs.StringVar(&base, "base", "HEAD~1", "Base git reference to compare against (e.g. HEAD~1, origin/main)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	files, err := GetChangedMarkdownFiles(dir, base, runGitDiff)
	if err != nil {
		fmt.Fprintf(stderr, "Error detecting git diff: %v\n", err)
		return 1
	}

	for _, file := range files {
		fmt.Fprintln(stdout, file)
	}

	return 0
}

type gitDiffRunner func(base string) (string, error)

func runGitDiff(base string) (string, error) {
	cmd := exec.Command("git", "diff", "-z", "--name-only", "--diff-filter=ACMR", base, "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %v (stderr: %s)", err, stderr.String())
	}
	return stdout.String(), nil
}

// GetChangedMarkdownFiles filters git diff output for markdown files in targetDir.
func GetChangedMarkdownFiles(targetDir string, base string, runner gitDiffRunner) ([]string, error) {
	output, err := runner(base)
	if err != nil {
		return nil, err
	}

	cleanTargetDir := filepath.Clean(targetDir)
	var matchedFiles []string

	paths := strings.Split(output, "\x00")
	for _, rawPath := range paths {
		if rawPath == "" {
			continue
		}

		cleanPath := filepath.Clean(rawPath)

		if strings.HasSuffix(cleanPath, ".md") {
			rel, err := filepath.Rel(cleanTargetDir, cleanPath)
			if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				matchedFiles = append(matchedFiles, cleanPath)
			}
		}
	}

	return matchedFiles, nil
}

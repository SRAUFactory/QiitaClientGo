package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestGetChangedMarkdownFiles(t *testing.T) {
	mockFiles := []string{
		"qiita/article1.md",
		"qiita/other.txt",
		"qiita/日本語記事.md",
		"qiita/space in name.md",
		"src/main.go",
		"qiita/nested/article2.md",
		"qiita/..foo.md",
		"",
	}
	mockGitOutput := strings.Join(mockFiles, "\x00")

	mockRunner := func(base string) (string, error) {
		return mockGitOutput, nil
	}

	files, err := GetChangedMarkdownFiles("qiita", "HEAD~1", mockRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"qiita/article1.md",
		"qiita/日本語記事.md",
		"qiita/space in name.md",
		"qiita/nested/article2.md",
		"qiita/..foo.md",
	}

	if !reflect.DeepEqual(files, expected) {
		t.Errorf("GetChangedMarkdownFiles() = %#v, want %#v", files, expected)
	}
}

func TestRunDiffCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Invalid flags should fail
	exitCode := RunDiffCommand([]string{"-invalid"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for invalid flags, got 0")
	}
}

package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestGetChangedMarkdownFiles(t *testing.T) {
	mockGitOutput := `qiita/article1.md
qiita/other.txt
"qiita/\346\227\245\346\234\254\350\252\236\350\250\230\344\272\213.md"
src/main.go
qiita/nested/article2.md
`

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
		"qiita/nested/article2.md",
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

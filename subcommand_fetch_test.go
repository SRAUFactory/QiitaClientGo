package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Normal Title", "Normal Title"},
		{"日本語のタイトル", "日本語のタイトル"},
		{"Path/Traversal\\Attempt", "Path_Traversal_Attempt"},
		{"../../etc/passwd", "etc_passwd"},
		{"Special: Characters? <foo>*", "Special_ Characters_ _foo"},
		{"...   ", "untitled"},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsPathSafe(t *testing.T) {
	baseDir := "/tmp/qiita"
	if !IsPathSafe(baseDir, "/tmp/qiita/article.md") {
		t.Errorf("expected /tmp/qiita/article.md to be safe")
	}
	if IsPathSafe(baseDir, "/tmp/qiita/../etc/passwd") {
		t.Errorf("expected /tmp/etc/passwd to be unsafe")
	}
	if IsPathSafe(baseDir, "/tmp/qiita") {
		t.Errorf("expected baseDir itself to not be a file target")
	}
}

func TestExecuteFetch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fetch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Pre-create an existing file that matches by title
	existingFile := filepath.Join(tempDir, "Existing Title.md")
	existingContent := FormatMarkdown(Frontmatter{
		Title: "Existing Title",
		Tags:  []string{"OldTag"},
	}, "Old Body")
	if err := os.WriteFile(existingFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	// Mock server
	mockItems := []QiitaItem{
		{
			ID:        "item-1",
			Title:     "Existing Title",
			Body:      "Updated Body from Qiita",
			Private:   false,
			Tags:      []QiitaTag{{Name: "NewTag"}},
			UpdatedAt: "2026-10-27T12:00:00Z",
		},
		{
			ID:        "item-2",
			Title:     "Brand New Article",
			Body:      "Brand New Content",
			Private:   false,
			Tags:      []QiitaTag{{Name: "Go"}},
			UpdatedAt: "2026-10-27T12:00:00Z",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "1" {
			_ = json.NewEncoder(w).Encode(mockItems)
		} else {
			_ = json.NewEncoder(w).Encode([]QiitaItem{})
		}
	}))
	defer ts.Close()

	client := NewQiitaClient("dummy-token", ts.URL, 5*time.Second)
	var stdout, stderr bytes.Buffer

	updated, err := ExecuteFetch(client, FetchOptions{Dir: tempDir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("ExecuteFetch failed: %v", err)
	}

	if len(updated) != 2 {
		t.Fatalf("expected 2 updated files, got %d: %v", len(updated), updated)
	}

	// Verify existing file now has QiitaID written
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("failed to read updated existing file: %v", err)
	}
	fm, body, err := ParseMarkdown(string(content))
	if err != nil {
		t.Fatalf("failed to parse updated existing file: %v", err)
	}
	if fm.QiitaID != "item-1" {
		t.Errorf("expected QiitaID item-1, got %s", fm.QiitaID)
	}
	if body != "Updated Body from Qiita\n" {
		t.Errorf("expected updated body, got %q", body)
	}

	// Re-run fetch: both should be skipped since updated_at matches
	stdout.Reset()
	updatedSecond, err := ExecuteFetch(client, FetchOptions{Dir: tempDir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("ExecuteFetch 2nd run failed: %v", err)
	}
	if len(updatedSecond) != 0 {
		t.Errorf("expected 0 files updated on 2nd run, got %d", len(updatedSecond))
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPushSingleFile_NewArticle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "push-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "new_post.md")
	initialContent := FormatMarkdown(Frontmatter{
		Title: "初めてのQiita記事",
		Tags:  []string{"Go", "CLI"},
	}, "本文内容です。")
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req QiitaItemRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(QiitaItem{
			ID:    "generated-id-999",
			Title: req.Title,
		})
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	var stdout, stderr bytes.Buffer

	err = PushSingleFile(client, testFile, PushOptions{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("PushSingleFile failed: %v", err)
	}

	// Verify ID written back to testFile
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	fm, _, err := ParseMarkdown(string(content))
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}
	if fm.QiitaID != "generated-id-999" {
		t.Errorf("expected QiitaID 'generated-id-999', got %s", fm.QiitaID)
	}
}

func TestPushSingleFile_UpdateArticle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "push-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "existing_post.md")
	initialContent := FormatMarkdown(Frontmatter{
		Title:   "既存記事",
		QiitaID: "existing-id-123",
		Tags:    []string{"Go"},
	}, "更新前の本文")
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	patchCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/items/existing-id-123" {
			t.Errorf("expected /items/existing-id-123, got %s", r.URL.Path)
		}
		patchCalled = true
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(QiitaItem{ID: "existing-id-123"})
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	var stdout, stderr bytes.Buffer

	err = PushSingleFile(client, testFile, PushOptions{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("PushSingleFile failed: %v", err)
	}

	if !patchCalled {
		t.Errorf("expected PATCH to be called")
	}
}

func TestRunPushCommand_StdinPipe(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "push-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "piped.md")
	_ = os.WriteFile(testFile, []byte("---\ntitle: Piped\n---\nBody"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(testFile + "\n")

	// Dry run with stdin pipe
	exitCode := RunPushCommand([]string{"-dry-run"}, stdin, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d, stderr: %s", exitCode, stderr.String())
	}

	if !strings.Contains(stdout.String(), "dry-run") {
		t.Errorf("expected dry-run output, got: %s", stdout.String())
	}
}

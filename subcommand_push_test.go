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

	const validID = "0123456789abcdef0123"
	testFile := filepath.Join(tempDir, "existing_post.md")
	initialContent := FormatMarkdown(Frontmatter{
		Title:   "既存記事",
		QiitaID: validID,
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
		if r.URL.Path != "/items/"+validID {
			t.Errorf("expected /items/%s, got %s", validID, r.URL.Path)
		}
		patchCalled = true
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(QiitaItem{ID: validID})
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

func TestPushSingleFile_InvalidQiitaID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "push-invalid-id-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)

	invalidIDs := []string{
		"../authenticated_user",
		"abc/def",
		"INVALIDUPPERCASE1234",
		"short",
	}

	for _, badID := range invalidIDs {
		testFile := filepath.Join(tempDir, "bad.md")
		content := FormatMarkdown(Frontmatter{
			Title:   "Bad ID Test",
			QiitaID: badID,
		}, "Body")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var stdout, stderr bytes.Buffer
		err := PushSingleFile(client, testFile, PushOptions{}, &stdout, &stderr)
		if err == nil {
			t.Errorf("expected error for invalid qiita_id %q, got nil", badID)
		}
		if apiCalled {
			t.Errorf("API should not be called for invalid qiita_id %q", badID)
		}
	}
}

func TestPushSingleFile_ControlCharacters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "push-ctrl-char-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)

	cases := []struct {
		name string
		fm   Frontmatter
	}{
		{name: "newline in title", fm: Frontmatter{Title: "C:\new"}},
		{name: "tab in title", fm: Frontmatter{Title: "C:\table"}},
		{name: "bell in title", fm: Frontmatter{Title: "C:\app\x07"}},
		{name: "backspace in title", fm: Frontmatter{Title: "C:\backup"}},
		{name: "newline in tag", fm: Frontmatter{Title: "Valid Title", Tags: []string{"tag\nname"}}},
		{name: "tab in tag", fm: Frontmatter{Title: "Valid Title", Tags: []string{"tag\tname"}}},
		{name: "bell in tag", fm: Frontmatter{Title: "Valid Title", Tags: []string{"tag\x07name"}}},
		{name: "backspace in tag", fm: Frontmatter{Title: "Valid Title", Tags: []string{"tag\bname"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "ctrl.md")
			content := FormatMarkdown(tc.fm, "Body")
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			var stdout, stderr bytes.Buffer
			err := PushSingleFile(client, testFile, PushOptions{DryRun: true}, &stdout, &stderr)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
			err = PushSingleFile(client, testFile, PushOptions{DryRun: false}, &stdout, &stderr)
			if err == nil {
				t.Errorf("expected error for %s (non-dry-run), got nil", tc.name)
			}
			if apiCalled {
				t.Errorf("API should not be called for %s", tc.name)
			}
		})
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

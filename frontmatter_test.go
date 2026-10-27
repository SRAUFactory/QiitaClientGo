package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantFm       Frontmatter
		wantBody     string
		expectErrMsg string
	}{
		{
			name: "Standard frontmatter with multiline tags",
			content: `---
private: false
qiita_id: e59fda709e5ed65eb489
tags:
  - CSS
  - HTML5
  - AMP
title: "AMPについてのリンク集"
updated_at: '2026-07-31T13:53:00+09:00'
---

数年前にあるサイトをAMP対応した際のリンク集。
`,
			wantFm: Frontmatter{
				Title:     "AMPについてのリンク集",
				Tags:      []string{"CSS", "HTML5", "AMP"},
				QiitaID:   "e59fda709e5ed65eb489",
				Private:   false,
				UpdatedAt: "2026-07-31T13:53:00+09:00",
			},
			wantBody: "数年前にあるサイトをAMP対応した際のリンク集。\n",
		},
		{
			name: "Inline tags format",
			content: `---
title: Go言語入門
tags: [Go, backend, CLI]
private: true
qiita_id: abc123def456
updated_at: 2026-10-01T00:00:00+09:00
---

# 本文
これはテストです。`,
			wantFm: Frontmatter{
				Title:     "Go言語入門",
				Tags:      []string{"Go", "backend", "CLI"},
				QiitaID:   "abc123def456",
				Private:   true,
				UpdatedAt: "2026-10-01T00:00:00+09:00",
			},
			wantBody: "# 本文\nこれはテストです。",
		},
		{
			name: "No frontmatter",
			content: `# タイトルのみ
本文です。`,
			wantFm:   Frontmatter{},
			wantBody: "# タイトルのみ\n本文です。",
		},
		{
			name: "Frontmatter with colons in title",
			content: `---
title: "Kubernetes: 入門と実践: 2026年版"
tags:
  - Kubernetes
  - DevOps
---

本文
`,
			wantFm: Frontmatter{
				Title: "Kubernetes: 入門と実践: 2026年版",
				Tags:  []string{"Kubernetes", "DevOps"},
			},
			wantBody: "本文\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := ParseMarkdown(tt.content)
			if tt.expectErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrMsg) {
					t.Fatalf("expected error containing %q, got %v", tt.expectErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(fm, tt.wantFm) {
				t.Errorf("ParseMarkdown() fm = %#v, want %#v", fm, tt.wantFm)
			}
			if body != tt.wantBody {
				t.Errorf("ParseMarkdown() body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestFormatMarkdown(t *testing.T) {
	fm := Frontmatter{
		Title:     "テスト記事: コロンあり",
		Tags:      []string{"Go", "Testing"},
		QiitaID:   "1234567890abcdef",
		Private:   false,
		UpdatedAt: "2026-10-27T10:00:00+09:00",
	}
	body := "## 記事の本文\n内容です。\n"

	formatted := FormatMarkdown(fm, body)

	// Round-trip parse test
	parsedFm, parsedBody, err := ParseMarkdown(formatted)
	if err != nil {
		t.Fatalf("failed to parse formatted markdown: %v", err)
	}

	if !reflect.DeepEqual(parsedFm, fm) {
		t.Errorf("Roundtrip failed: got %#v, want %#v", parsedFm, fm)
	}
	if parsedBody != body {
		t.Errorf("Body changed: got %q, want %q", parsedBody, body)
	}
}

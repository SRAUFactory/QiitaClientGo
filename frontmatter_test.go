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

func TestFormatMarkdown_RoundTripEscaping(t *testing.T) {
	tests := []struct {
		name string
		fm   Frontmatter
		body string
	}{
		{
			name: "Quotes in title",
			fm: Frontmatter{
				Title: `He said "hi"`,
				Tags:  []string{"quote"},
			},
			body: "Body text\n",
		},
		{
			name: "Newline in title",
			fm: Frontmatter{
				Title: "line1\nline2",
				Tags:  []string{"multiline"},
			},
			body: "Body text\n",
		},
		{
			name: "Backslash in title",
			fm: Frontmatter{
				Title: `backslash\test`,
				Tags:  []string{`tag\with\backslash`},
			},
			body: "Body text\n",
		},
		{
			name: "Special characters in tags",
			fm: Frontmatter{
				Title: "Tag escaping test",
				Tags:  []string{`"quoted"`, "colon: in tag", "line1\nline2"},
			},
			body: "Body text\n",
		},
		{
			name: "Colon and special words",
			fm: Frontmatter{
				Title:   "a: b",
				Tags:    []string{"true", "[draft] memo"},
				Private: true,
			},
			body: "Body text\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatMarkdown(tt.fm, tt.body)
			parsedFm, parsedBody, err := ParseMarkdown(formatted)
			if err != nil {
				t.Fatalf("ParseMarkdown failed on formatted markdown:\n%s\nError: %v", formatted, err)
			}
			if !reflect.DeepEqual(parsedFm, tt.fm) {
				t.Errorf("Roundtrip Frontmatter mismatch:\ngot:  %#v\nwant: %#v\nFormatted was:\n%s", parsedFm, tt.fm, formatted)
			}
			if parsedBody != tt.body {
				t.Errorf("Roundtrip Body mismatch:\ngot:  %q\nwant: %q", parsedBody, tt.body)
			}
		})
	}
}

func TestParseMarkdown_LegacyV100Compatibility(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Frontmatter
	}{
		{
			name: "Legacy invalid JSON escape in title (Path: D:\\users)",
			content: `---
title: "Path: D:\users"
tags:
  - "Tag: C:\work"
---
body`,
			want: Frontmatter{
				Title: `Path: D:\users`,
				Tags:  []string{`Tag: C:\work`},
			},
		},
		{
			name: "Legacy escaped quotes with backslash in title",
			content: `---
title: "He said \"hi\" from D:\dir"
tags:
  - "a\c"
---
body`,
			want: Frontmatter{
				Title: `He said "hi" from D:\dir`,
				Tags:  []string{`a\c`},
			},
		},
		{
			name: "Legacy valid-looking JSON escapes prioritize new format (\\n and \\t)",
			content: `---
title: "Notes: C:\new\table"
tags:
  - "ok"
---
body`,
			want: Frontmatter{
				Title: "Notes: C:\new\table",
				Tags:  []string{"ok"},
			},
		},
		{
			name: "Legacy Go-specific escapes remain literal (C:\\app, C:\\vendor, D:\\apache\\bin)",
			content: `---
title: "Dir: C:\app"
tags:
  - "Dir: C:\vendor"
  - "Dir: D:\apache\bin"
---
body`,
			want: Frontmatter{
				Title: `Dir: C:\app`,
				Tags:  []string{`Dir: C:\vendor`, `Dir: D:\apache\bin`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := ParseMarkdown(tt.content)
			if err != nil {
				t.Fatalf("ParseMarkdown failed: %v", err)
			}
			if !reflect.DeepEqual(fm, tt.want) {
				t.Errorf("ParseMarkdown() = %#v, want %#v", fm, tt.want)
			}
		})
	}

	t.Run("Legacy backslash in inline tags", func(t *testing.T) {
		inline := "---\ntitle: \"Path: D:\\users\"\ntags: [Windows, \"D:\\data\"]\n---\nbody\n"
		fmInline, _, err := ParseMarkdown(inline)
		if err != nil {
			t.Fatalf("ParseMarkdown inline failed: %v", err)
		}
		if !reflect.DeepEqual(fmInline.Tags, []string{"Windows", `D:\data`}) {
			t.Errorf("Inline Tags = %#v, want %#v", fmInline.Tags, []string{"Windows", `D:\data`})
		}
	})
}

func TestFormatMarkdown_NoHTMLEscape(t *testing.T) {
	tests := []struct {
		name      string
		fm        Frontmatter
		wantLines []string
	}{
		{
			name: "Ampersand and angle brackets in title and tags",
			fm: Frontmatter{
				Title: "A & B: List<T> の使い方",
				Tags:  []string{"C++", "A&B", "List<T>", "x>y"},
			},
			wantLines: []string{
				`title: "A & B: List<T> の使い方"`,
				`  - "A&B"`,
				`  - "List<T>"`,
				`  - "x>y"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatMarkdown(tt.fm, "Body\n")
			for _, wantLine := range tt.wantLines {
				if !strings.Contains(formatted, wantLine) {
					t.Errorf("formatted markdown does not contain %q\nGot:\n%s", wantLine, formatted)
				}
			}
			if strings.Contains(formatted, `\u0026`) || strings.Contains(formatted, `\u003c`) || strings.Contains(formatted, `\u003e`) {
				t.Errorf("formatted markdown should not contain HTML unicode escapes:\n%s", formatted)
			}

			parsedFm, _, err := ParseMarkdown(formatted)
			if err != nil {
				t.Fatalf("ParseMarkdown failed: %v", err)
			}
			if !reflect.DeepEqual(parsedFm, tt.fm) {
				t.Errorf("Roundtrip mismatch: got %#v, want %#v", parsedFm, tt.fm)
			}
		})
	}
}

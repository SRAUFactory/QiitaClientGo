package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Frontmatter represents metadata stored in Qiita Markdown files.
type Frontmatter struct {
	Title     string   `json:"title"`
	Tags      []string `json:"tags"`
	QiitaID   string   `json:"qiita_id"`
	Private   bool     `json:"private"`
	UpdatedAt string   `json:"updated_at"`
}

// ParseMarkdown separates frontmatter and body from Markdown text.
// If frontmatter is missing, it returns an empty Frontmatter and the full text as body.
func ParseMarkdown(content string) (Frontmatter, string, error) {
	trimmed := strings.TrimLeft(content, "\r\n ")
	if !strings.HasPrefix(trimmed, "---") {
		return Frontmatter{}, content, nil
	}

	// Remove leading ---
	afterFirst := strings.TrimPrefix(trimmed, "---")
	idx := strings.Index(afterFirst, "\n---")
	if idx == -1 {
		// Not closed properly
		return Frontmatter{}, content, nil
	}

	yamlBlock := afterFirst[:idx]
	// The rest is body
	rest := afterFirst[idx+4:] // len("\n---") == 4
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}

	fm, err := parseYAMLFrontmatter(yamlBlock)
	if err != nil {
		return Frontmatter{}, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return fm, strings.TrimLeft(rest, "\r\n"), nil
}

func parseYAMLFrontmatter(block string) (Frontmatter, error) {
	var fm Frontmatter
	scanner := bufio.NewScanner(strings.NewReader(block))

	currentKey := ""
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check for list items under a key (e.g. tags)
		if strings.HasPrefix(trimmedLine, "- ") {
			if currentKey == "tags" {
				tagVal := cleanValue(strings.TrimPrefix(trimmedLine, "- "))
				if tagVal != "" {
					fm.Tags = append(fm.Tags, tagVal)
				}
			}
			continue
		}

		// Key-value pair
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		currentKey = key
		cleanVal := cleanValue(val)

		switch key {
		case "title":
			fm.Title = cleanVal
		case "qiita_id":
			fm.QiitaID = cleanVal
		case "private":
			fm.Private = strings.ToLower(cleanVal) == "true"
		case "updated_at":
			fm.UpdatedAt = cleanVal
		case "tags":
			if cleanVal != "" {
				// Inline list support: [tag1, tag2]
				if strings.HasPrefix(cleanVal, "[") && strings.HasSuffix(cleanVal, "]") {
					items := parseInlineList(cleanVal)
					fm.Tags = append(fm.Tags, items...)
				} else {
					fm.Tags = append(fm.Tags, cleanVal)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fm, err
	}
	return fm, nil
}

func cleanValue(val string) string {
	v := strings.TrimSpace(val)
	if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") && len(v) >= 2 {
		var out string
		if err := json.Unmarshal([]byte(v), &out); err == nil {
			return out
		}
		if u, err := strconv.Unquote(v); err == nil {
			return u
		}
	}
	if strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") && len(v) >= 2 {
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return strings.TrimSpace(v)
}

func parseInlineList(val string) []string {
	content := strings.TrimSpace(val)
	if !strings.HasPrefix(content, "[") || !strings.HasSuffix(content, "]") {
		return nil
	}
	inner := strings.TrimSpace(content[1 : len(content)-1])
	if inner == "" {
		return []string{}
	}

	var items []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(inner); i++ {
		ch := inner[i]

		if inDoubleQuote {
			current.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inSingleQuote {
			current.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(inner) && inner[i+1] == '\'' {
					current.WriteByte('\'')
					i++
				} else {
					inSingleQuote = false
				}
			}
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
			current.WriteByte(ch)
		case '\'':
			inSingleQuote = true
			current.WriteByte(ch)
		case ',':
			v := cleanValue(current.String())
			if v != "" {
				items = append(items, v)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		v := cleanValue(current.String())
		if v != "" {
			items = append(items, v)
		}
	}

	return items
}

func formatYAMLString(s string) string {
	// If the string contains special characters, quotes, or control characters,
	// escape it using JSON encoding (standard YAML compatible double-quoted scalar).
	if strings.ContainsAny(s, ":#\"'\\\n\r\t[]{},&*?|<>=!%@`") ||
		strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") ||
		strings.HasPrefix(s, "-") || s == "" {
		b, err := json.Marshal(s)
		if err == nil {
			return string(b)
		}
		return strconv.Quote(s)
	}
	return s
}

// FormatMarkdown serializes the frontmatter and body into Markdown content.
func FormatMarkdown(fm Frontmatter, body string) string {
	var b strings.Builder
	b.WriteString("---\n")

	b.WriteString(fmt.Sprintf("title: %s\n", formatYAMLString(fm.Title)))

	b.WriteString("tags:\n")
	for _, tag := range fm.Tags {
		b.WriteString(fmt.Sprintf("  - %s\n", formatYAMLString(tag)))
	}

	if fm.QiitaID != "" {
		b.WriteString(fmt.Sprintf("qiita_id: %s\n", fm.QiitaID))
	}

	if fm.Private {
		b.WriteString("private: true\n")
	} else {
		b.WriteString("private: false\n")
	}

	if fm.UpdatedAt != "" {
		b.WriteString(fmt.Sprintf("updated_at: '%s'\n", strings.ReplaceAll(fm.UpdatedAt, "'", "''")))
	}

	b.WriteString("---\n\n")
	b.WriteString(strings.TrimLeft(body, "\r\n"))
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}

	return b.String()
}

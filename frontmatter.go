package main

import (
	"bufio"
	"fmt"
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
					items := strings.Split(cleanVal[1:len(cleanVal)-1], ",")
					for _, item := range items {
						v := cleanValue(item)
						if v != "" {
							fm.Tags = append(fm.Tags, v)
						}
					}
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
	if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
		(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
		if len(v) >= 2 {
			v = v[1 : len(v)-1]
		}
	}
	return strings.TrimSpace(v)
}

// FormatMarkdown serializes the frontmatter and body into Markdown content.
func FormatMarkdown(fm Frontmatter, body string) string {
	var b strings.Builder
	b.WriteString("---\n")

	// Escape or quote title if it contains colons, quotes or special chars
	title := fm.Title
	if strings.ContainsAny(title, ":#\"'\n\r\t") || strings.HasPrefix(title, " ") || strings.HasSuffix(title, " ") {
		escaped := strings.ReplaceAll(title, "\"", "\\\"")
		b.WriteString(fmt.Sprintf("title: \"%s\"\n", escaped))
	} else {
		b.WriteString(fmt.Sprintf("title: %s\n", title))
	}

	b.WriteString("tags:\n")
	for _, tag := range fm.Tags {
		if strings.ContainsAny(tag, ":#\"'\n\r\t") {
			b.WriteString(fmt.Sprintf("  - \"%s\"\n", strings.ReplaceAll(tag, "\"", "\\\"")))
		} else {
			b.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
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
		b.WriteString(fmt.Sprintf("updated_at: '%s'\n", fm.UpdatedAt))
	}

	b.WriteString("---\n\n")
	b.WriteString(strings.TrimLeft(body, "\r\n"))
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}

	return b.String()
}

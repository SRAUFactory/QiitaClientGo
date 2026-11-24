package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var invalidFilenameRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// SanitizeFilename converts an article title into a safe filename, preventing path traversal.
func SanitizeFilename(title string) string {
	safe := invalidFilenameRegex.ReplaceAllString(title, "_")
	safe = strings.ReplaceAll(safe, "..", "_")
	safe = strings.Trim(safe, ". _")
	if safe == "" {
		safe = "untitled"
	}
	return safe
}

// IsPathSafe verifies that candidatePath is strictly within baseDir.
func IsPathSafe(baseDir string, candidatePath string) bool {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	absCandidate, err := filepath.Abs(candidatePath)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return false
	}

	// Must not escape baseDir with ".." or be baseDir itself
	if strings.HasPrefix(rel, "..") || rel == "." {
		return false
	}
	return true
}

// FetchOptions configures the fetch subcommand.
type FetchOptions struct {
	Dir     string
	Token   string
	BaseURL string
	Timeout time.Duration
}

// resolveToken resolves the Qiita API token from the flag value or the QIITA_TOKEN environment variable.
func resolveToken(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("QIITA_TOKEN")
}

// RunFetchCommand executes the fetch subcommand.
func RunFetchCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var dir string
	var token string
	fs.StringVar(&dir, "dir", "qiita", "Target directory to save Qiita markdown files")
	fs.StringVar(&token, "token", "", "Qiita API token (defaults to QIITA_TOKEN environment variable)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	resolvedToken := resolveToken(token)
	if resolvedToken == "" {
		fmt.Fprintln(stderr, "Error: Qiita API token is required. Set QIITA_TOKEN environment variable or use -token.")
		return 1
	}

	client := NewQiitaClient(resolvedToken, "", DefaultTimeout)
	opts := FetchOptions{
		Dir: dir,
	}

	updatedFiles, err := ExecuteFetch(client, opts, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Error executing fetch: %v\n", err)
		return 1
	}

	if len(updatedFiles) == 0 {
		fmt.Fprintln(stdout, "No articles updated.")
	} else {
		fmt.Fprintf(stdout, "\nSuccessfully synced %d files.\n", len(updatedFiles))
	}

	return 0
}

// ExecuteFetch performs the fetch and sync logic against the local directory.
func ExecuteFetch(client *QiitaClient, opts FetchOptions, stdout io.Writer, stderr io.Writer) ([]string, error) {
	fmt.Fprintln(stdout, "Fetching articles from Qiita API...")
	items, err := client.FetchAllAuthenticatedUserItems()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch articles: %w", err)
	}
	fmt.Fprintf(stdout, "  → Retrieved %d articles\n", len(items))

	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", opts.Dir, err)
	}

	type localFileInfo struct {
		path      string
		updatedAt string
	}

	// Map existing local files
	idToFile := make(map[string]localFileInfo)
	titleToFile := make(map[string]string)
	occupiedPaths := make(map[string]string) // path -> qiita_id

	err = filepath.WalkDir(opts.Dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			fmt.Fprintf(stderr, "  WARNING: Failed to access %s: %v\n", path, walkErr)
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		cleanPath := filepath.Clean(path)

		content, readErr := os.ReadFile(cleanPath)
		if readErr != nil {
			fmt.Fprintf(stderr, "  WARNING: Failed to read %s: %v\n", cleanPath, readErr)
			return nil
		}

		fm, _, parseErr := ParseMarkdown(string(content))
		if parseErr != nil {
			fmt.Fprintf(stderr, "  WARNING: Failed to parse frontmatter in %s: %v\n", cleanPath, parseErr)
			return nil
		}

		occupiedPaths[cleanPath] = fm.QiitaID

		if fm.QiitaID != "" {
			idToFile[fm.QiitaID] = localFileInfo{
				path:      cleanPath,
				updatedAt: fm.UpdatedAt,
			}
		} else {
			stem := strings.TrimSuffix(filepath.Base(cleanPath), ".md")
			titleToFile[stem] = cleanPath
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %s: %w", opts.Dir, err)
	}

	var updatedFiles []string

	for _, item := range items {
		itemTags := make([]string, len(item.Tags))
		for i, t := range item.Tags {
			itemTags[i] = t.Name
		}

		fm := Frontmatter{
			Title:     item.Title,
			Tags:      itemTags,
			QiitaID:   item.ID,
			Private:   item.Private,
			UpdatedAt: item.UpdatedAt,
		}

		var targetPath string
		if localInfo, ok := idToFile[item.ID]; ok {
			// Check updated_at
			if localInfo.updatedAt == item.UpdatedAt {
				fmt.Fprintf(stdout, "  Skip (no change): %s\n", item.Title)
				continue
			}
			targetPath = localInfo.path
			fmt.Fprintf(stdout, "  Update: %s\n", item.Title)
		} else if existingPath, ok := titleToFile[item.Title]; ok {
			targetPath = existingPath
			fmt.Fprintf(stdout, "  Initial link by title: %s\n", item.Title)
			delete(titleToFile, item.Title)
		} else {
			safeTitle := SanitizeFilename(item.Title)
			candidate := filepath.Clean(filepath.Join(opts.Dir, safeTitle+".md"))

			// Check collision with already occupied paths by other articles
			if existingOwner, occupied := occupiedPaths[candidate]; occupied && existingOwner != item.ID {
				// Collision detected: fallback to <safeTitle>-<qiita_id>.md
				suffixCandidate := filepath.Clean(filepath.Join(opts.Dir, fmt.Sprintf("%s-%s.md", safeTitle, item.ID)))
				fmt.Fprintf(stdout, "  Filename collision for %s; using %s\n", candidate, suffixCandidate)
				candidate = suffixCandidate
			}

			if !IsPathSafe(opts.Dir, candidate) {
				fmt.Fprintf(stderr, "  WARNING: Skipping unsafe path for article: %s\n", item.Title)
				continue
			}
			targetPath = candidate
			fmt.Fprintf(stdout, "  New file: %s\n", targetPath)
		}

		occupiedPaths[targetPath] = item.ID
		idToFile[item.ID] = localFileInfo{
			path:      targetPath,
			updatedAt: item.UpdatedAt,
		}

		formatted := FormatMarkdown(fm, item.Body)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, []byte(formatted), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
		updatedFiles = append(updatedFiles, targetPath)
	}

	return updatedFiles, nil
}

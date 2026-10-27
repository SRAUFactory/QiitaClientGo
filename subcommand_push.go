package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PushOptions holds options for the push subcommand.
type PushOptions struct {
	Token  string
	DryRun bool
	Files  []string
}

// RunPushCommand executes the push subcommand.
func RunPushCommand(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var token string
	var dryRun bool
	fs.StringVar(&token, "token", "", "Qiita API token (defaults to QIITA_TOKEN environment variable)")
	fs.BoolVar(&dryRun, "dry-run", false, "Simulate push without modifying Qiita or local files")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if token == "" {
		token = os.Getenv("QIITA_TOKEN")
	}
	if token == "" && !dryRun {
		fmt.Fprintln(stderr, "Error: Qiita API token is required. Set QIITA_TOKEN environment variable or use -token.")
		return 1
	}

	var targetFiles []string
	posArgs := fs.Args()

	for _, arg := range posArgs {
		if arg == "-" {
			// Read from stdin
			stdinFiles := readLinesFromReader(stdin)
			targetFiles = append(targetFiles, stdinFiles...)
		} else {
			targetFiles = append(targetFiles, arg)
		}
	}

	// If no positional arguments given, try reading from stdin (pipe mode)
	if len(posArgs) == 0 {
		stdinFiles := readLinesFromReader(stdin)
		targetFiles = append(targetFiles, stdinFiles...)
	}

	if len(targetFiles) == 0 {
		fmt.Fprintln(stdout, "No files specified to push.")
		return 0
	}

	client := NewQiitaClient(token, "", DefaultTimeout)
	opts := PushOptions{
		Token:  token,
		DryRun: dryRun,
		Files:  targetFiles,
	}

	hasError := false
	for _, file := range targetFiles {
		cleanPath := filepath.Clean(strings.TrimSpace(file))
		if cleanPath == "" {
			continue
		}

		if err := PushSingleFile(client, cleanPath, opts, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "Error pushing %s: %v\n", cleanPath, err)
			hasError = true
		}
	}

	if hasError {
		return 1
	}
	return 0
}

func readLinesFromReader(r io.Reader) []string {
	if r == nil {
		return nil
	}
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// PushSingleFile handles posting or updating a single Markdown file to Qiita.
func PushSingleFile(client *QiitaClient, filePath string, opts PushOptions, stdout io.Writer, stderr io.Writer) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fm, body, err := ParseMarkdown(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	title := fm.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(filePath), ".md")
		if title == "" {
			title = "無題"
		}
	}

	tags := make([]QiitaTag, len(fm.Tags))
	for i, t := range fm.Tags {
		tags[i] = QiitaTag{Name: t}
	}

	reqData := QiitaItemRequest{
		Title:   title,
		Body:    body,
		Private: fm.Private,
		Tags:    tags,
	}

	if fm.QiitaID == "" {
		// New post
		fmt.Fprintf(stdout, "Creating new article: %s (from %s)\n", title, filePath)
		if opts.DryRun {
			fmt.Fprintf(stdout, "  [dry-run] Would create article and write back ID to %s\n", filePath)
			return nil
		}

		item, err := client.CreateItem(reqData)
		if err != nil {
			return fmt.Errorf("failed to create Qiita item: %w", err)
		}

		fmt.Fprintf(stdout, "  → Created successfully (qiita_id: %s)\n", item.ID)

		// Write back qiita_id into frontmatter
		fm.QiitaID = item.ID
		fm.Title = title
		newContent := FormatMarkdown(fm, body)
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write back qiita_id to %s: %w", filePath, err)
		}
		fmt.Fprintf(stdout, "  → Saved qiita_id back to %s\n", filePath)
	} else {
		// Update existing post
		fmt.Fprintf(stdout, "Updating article: %s (qiita_id: %s, file: %s)\n", title, fm.QiitaID, filePath)
		if opts.DryRun {
			fmt.Fprintf(stdout, "  [dry-run] Would update article %s\n", fm.QiitaID)
			return nil
		}

		_, err := client.UpdateItem(fm.QiitaID, reqData)
		if err != nil {
			return fmt.Errorf("failed to update Qiita item %s: %w", fm.QiitaID, err)
		}
		fmt.Fprintf(stdout, "  → Updated successfully\n")
	}

	return nil
}

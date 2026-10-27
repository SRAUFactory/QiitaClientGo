package main

import (
	"fmt"
	"os"
)

var (
	// Version is set at build time or defaults to 1.0.0
	Version = "1.0.0"
)

func printUsage() {
	usage := `qiita-client is a CLI tool for syncing Markdown articles with Qiita.

Usage:
  qiita-client <command> [options] [arguments]

Available Commands:
  fetch       Fetch articles from Qiita and save as Markdown files with YAML frontmatter
  push        Publish or update Markdown files to Qiita
  diff        Detect changed Markdown files using git diff
  version     Show version information
  help        Show this help message

Authentication:
  Set the QIITA_TOKEN environment variable with your personal access token.

Examples:
  # Fetch all articles into ./qiita directory
  export QIITA_TOKEN="your_token_here"
  qiita-client fetch -dir qiita

  # Detect changed files and push them to Qiita
  qiita-client diff -dir qiita | qiita-client push

  # Push specific markdown files
  qiita-client push qiita/my-article.md

Run 'qiita-client <command> -h' for more information on a command.
`
	fmt.Print(usage)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "fetch":
		os.Exit(RunFetchCommand(args, os.Stdout, os.Stderr))
	case "push":
		os.Exit(RunPushCommand(args, os.Stdin, os.Stdout, os.Stderr))
	case "diff":
		os.Exit(RunDiffCommand(args, os.Stdout, os.Stderr))
	case "version", "-v", "--version":
		fmt.Printf("qiita-client version %s\n", Version)
		os.Exit(0)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

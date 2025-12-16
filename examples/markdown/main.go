// Example: Markdown translation extraction
//
// This example demonstrates extracting translatable strings from markdown files.
// Run with: go run . -content ./content -locale en
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cespare/xxhash/v2"
	"github.com/romshark/toki/internal/markdown"
)

func main() {
	contentDir := flag.String("content", "./content", "directory containing markdown files")
	locale := flag.String("locale", "en", "default locale")
	verbose := flag.Bool("v", false, "verbose output")
	jsonOutput := flag.Bool("json", false, "output ARB-compatible JSON")
	flag.Parse()

	parser := markdown.NewParser()
	hasher := xxhash.New()

	results, err := parser.ParseDir(*contentDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing markdown: %v\n", err)
		os.Exit(1)
	}

	// Generate hashes for all texts
	results.GenerateHashes(hasher)

	if *jsonOutput {
		// Output ARB-compatible JSON
		arbOutput := map[string]any{
			"@@locale": *locale,
		}
		messages := results.ToARBMessages(hasher)
		for _, msg := range messages {
			arbOutput[msg.ID] = msg.ICUMessage
			arbOutput["@"+msg.ID] = map[string]string{
				"description": msg.Description,
				"context":     msg.Context,
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(arbOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("Scanned %d markdown files for locale %q\n\n", len(results.Files), *locale)

	for _, file := range results.Files {
		fmt.Printf("File: %s\n", file.Path)
		if file.Frontmatter != nil {
			fmt.Printf("  Title: %s\n", file.Frontmatter.Title)
			if file.Frontmatter.Description != "" {
				fmt.Printf("  Description: %s\n", file.Frontmatter.Description)
			}
		}
		fmt.Printf("  Texts: %d\n", len(file.Texts))

		if *verbose {
			for i, text := range file.Texts {
				fmt.Printf("    [%d] %s Line %d (%s): %q\n",
					i+1, text.IDHash, text.Line, text.Type, truncate(text.Content, 50))
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total translatable texts: %d\n", results.TotalTexts())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

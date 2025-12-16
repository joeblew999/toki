package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseSimple(t *testing.T) {
	content := `# Hello World

This is a paragraph.

## Features

- Item one
- Item two
`
	file := writeTestFile(t, "test.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	assert.Nil(t, result.Frontmatter)
	assert.Len(t, result.Texts, 5)

	// Check heading
	assert.Equal(t, "Hello World", result.Texts[0].Content)
	assert.Equal(t, TextTypeHeading, result.Texts[0].Type)
	assert.Equal(t, 1, result.Texts[0].Line)

	// Check paragraph
	assert.Equal(t, "This is a paragraph.", result.Texts[1].Content)
	assert.Equal(t, TextTypeParagraph, result.Texts[1].Type)

	// Check second heading
	assert.Equal(t, "Features", result.Texts[2].Content)
	assert.Equal(t, TextTypeHeading, result.Texts[2].Type)

	// Check list items
	assert.Equal(t, "Item one", result.Texts[3].Content)
	assert.Equal(t, TextTypeListItem, result.Texts[3].Type)
	assert.Equal(t, "Item two", result.Texts[4].Content)
	assert.Equal(t, TextTypeListItem, result.Texts[4].Type)
}

func TestParser_Frontmatter(t *testing.T) {
	content := `---
title: Test Page
description: A test description
author: Test Author
---

# Main Content

Some text here.
`
	file := writeTestFile(t, "frontmatter.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	require.NotNil(t, result.Frontmatter)
	assert.Equal(t, "Test Page", result.Frontmatter.Title)
	assert.Equal(t, "A test description", result.Frontmatter.Description)
	assert.Equal(t, "Test Author", result.Frontmatter.Author)

	// Frontmatter title + description + heading + paragraph = 4 texts
	assert.Len(t, result.Texts, 4)

	// First two should be from frontmatter
	assert.Equal(t, TextTypeFrontmatter, result.Texts[0].Type)
	assert.Equal(t, "Test Page", result.Texts[0].Content)
	assert.Equal(t, TextTypeFrontmatter, result.Texts[1].Type)
	assert.Equal(t, "A test description", result.Texts[1].Content)
}

func TestParser_CodeBlocksSkipped(t *testing.T) {
	content := "# Title\n\n```go\nfunc main() {}\n```\n\nParagraph after code.\n"

	file := writeTestFile(t, "code.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	// Should have title and paragraph, but NOT the code
	assert.Len(t, result.Texts, 2)
	assert.Equal(t, "Title", result.Texts[0].Content)
	assert.Equal(t, "Paragraph after code.", result.Texts[1].Content)
}

func TestParser_Blockquotes(t *testing.T) {
	content := "# Quote Section\n\n> This is a quote.\n> It spans multiple lines.\n"

	file := writeTestFile(t, "quote.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	assert.Len(t, result.Texts, 2)
	assert.Equal(t, TextTypeHeading, result.Texts[0].Type)
	assert.Equal(t, TextTypeBlockquote, result.Texts[1].Type)
	assert.Contains(t, result.Texts[1].Content, "This is a quote.")
}

func TestParser_ImageAlt(t *testing.T) {
	content := "# Images\n\n![Alt text for image](image.png \"Title\")\n"

	file := writeTestFile(t, "images.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	// Find image text
	var imgText *Text
	for i := range result.Texts {
		if result.Texts[i].Type == TextTypeImage {
			imgText = &result.Texts[i]
			break
		}
	}
	require.NotNil(t, imgText)
	assert.Equal(t, "Title", imgText.Content) // Title takes precedence
}

func TestParser_Emphasis(t *testing.T) {
	content := "# Test\n\nThis has **bold** and *italic* text.\n"

	file := writeTestFile(t, "emphasis.md", content)
	parser := NewParser()

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	assert.Len(t, result.Texts, 2)
	// Emphasis markers should be preserved
	assert.Contains(t, result.Texts[1].Content, "**bold**")
	assert.Contains(t, result.Texts[1].Content, "*italic*")
}

func TestParser_Options(t *testing.T) {
	content := "# Heading\n\nParagraph.\n\n- List item\n"

	file := writeTestFile(t, "options.md", content)

	// Disable list item extraction
	opts := DefaultOptions()
	opts.ExtractListItems = false
	parser := NewParserWithOptions(opts)

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	// Should have heading and paragraph, but NOT list item
	assert.Len(t, result.Texts, 2)
	for _, text := range result.Texts {
		assert.NotEqual(t, TextTypeListItem, text.Type)
	}
}

func TestParser_MinTextLength(t *testing.T) {
	content := "# OK\n\nShort\n\nThis is a longer paragraph that should pass.\n"

	file := writeTestFile(t, "minlen.md", content)

	opts := DefaultOptions()
	opts.MinTextLength = 10
	parser := NewParserWithOptions(opts)

	result, err := parser.ParseFile(file)
	require.NoError(t, err)

	// "OK" (2 chars) and "Short" (5 chars) should be excluded
	// Only the longer paragraph should remain
	assert.Len(t, result.Texts, 1)
	assert.Contains(t, result.Texts[0].Content, "longer paragraph")
}

func TestHashText(t *testing.T) {
	hasher := xxhash.New()

	hash1 := HashText(hasher, "Hello World")
	hash2 := HashText(hasher, "Hello World")
	hash3 := HashText(hasher, "Different text")

	// Same content = same hash
	assert.Equal(t, hash1, hash2)
	// Different content = different hash
	assert.NotEqual(t, hash1, hash3)
	// Hash format
	assert.Regexp(t, `^msg[a-f0-9]+$`, hash1)
}

func TestToARBMessage(t *testing.T) {
	hasher := xxhash.New()

	text := Text{
		Content: "Hello World",
		Type:    TextTypeHeading,
		Line:    5,
		Context: "h1 heading",
	}

	msg := text.ToARBMessage(hasher)

	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "Hello World", msg.ICUMessage)
	assert.Equal(t, "h1 heading", msg.Description)
}

func TestParseDir(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "blog"), 0o755))
	writeTestFileAt(t, filepath.Join(dir, "index.md"), "# Home\n\nWelcome.\n")
	writeTestFileAt(t, filepath.Join(dir, "blog", "post.md"), "# Post\n\nContent.\n")

	parser := NewParser()
	result, err := parser.ParseDir(dir)
	require.NoError(t, err)

	assert.Len(t, result.Files, 2)
	assert.Equal(t, 4, result.TotalTexts()) // 2 headings + 2 paragraphs
}

// Helper functions

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	return writeTestFileAt(t, filepath.Join(dir, name), content)
}

func writeTestFileAt(t *testing.T, path, content string) string {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

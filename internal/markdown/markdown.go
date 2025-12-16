// Package markdown extracts translatable text from markdown files.
package markdown

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// TextType identifies the kind of markdown element a text was extracted from.
type TextType string

const (
	TextTypeHeading     TextType = "heading"
	TextTypeParagraph   TextType = "paragraph"
	TextTypeListItem    TextType = "list_item"
	TextTypeBlockquote  TextType = "blockquote"
	TextTypeLink        TextType = "link"
	TextTypeImage       TextType = "image"
	TextTypeFrontmatter TextType = "frontmatter"
)

// Text represents an extracted translatable text from markdown.
type Text struct {
	Content  string   // The actual text content
	Type     TextType // Type of markdown element
	Line     int      // Line number in source file (1-indexed)
	IDHash   string   // Unique hash for this text (for ARB mapping)
	Context  string   // Optional context hint for translators
	Comments []string // Any HTML comments preceding this element
}

// Frontmatter holds parsed YAML frontmatter from markdown files.
type Frontmatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Author      string   `yaml:"author"`
	Date        string   `yaml:"date"`
	Tags        []string `yaml:"tags"`
	// Extra holds any additional frontmatter fields
	Extra map[string]any `yaml:",inline"`
}

// File represents a parsed markdown file with its extracted texts.
type File struct {
	Path        string       // File path (relative to content root)
	AbsPath     string       // Absolute file path
	Frontmatter *Frontmatter // Parsed frontmatter (nil if none)
	Texts       []Text       // Extracted translatable texts
}

// ScanResult holds the results of parsing a directory of markdown files.
type ScanResult struct {
	Files []*File
}

// TotalTexts returns the total count of translatable texts across all files.
func (r *ScanResult) TotalTexts() int {
	count := 0
	for _, f := range r.Files {
		count += len(f.Texts)
	}
	return count
}

// Parser extracts translatable text from markdown files.
type Parser struct {
	md      goldmark.Markdown
	options ParserOptions
}

// ParserOptions configures what to extract from markdown.
type ParserOptions struct {
	// ExtractHeadings includes heading text (default: true)
	ExtractHeadings bool
	// ExtractParagraphs includes paragraph text (default: true)
	ExtractParagraphs bool
	// ExtractListItems includes list item text (default: true)
	ExtractListItems bool
	// ExtractBlockquotes includes blockquote text (default: true)
	ExtractBlockquotes bool
	// ExtractImageAlt includes image alt text (default: true)
	ExtractImageAlt bool
	// ExtractLinkText includes link text (default: true)
	ExtractLinkText bool
	// ExtractFrontmatter includes frontmatter title/description (default: true)
	ExtractFrontmatter bool
	// MinTextLength ignores text shorter than this (default: 0)
	MinTextLength int
}

// DefaultOptions returns parser options with all extraction enabled.
func DefaultOptions() ParserOptions {
	return ParserOptions{
		ExtractHeadings:    true,
		ExtractParagraphs:  true,
		ExtractListItems:   true,
		ExtractBlockquotes: true,
		ExtractImageAlt:    true,
		ExtractLinkText:    true,
		ExtractFrontmatter: true,
		MinTextLength:      0,
	}
}

// NewParser creates a markdown parser with default options.
func NewParser() *Parser {
	return NewParserWithOptions(DefaultOptions())
}

// NewParserWithOptions creates a markdown parser with custom options.
func NewParserWithOptions(opts ParserOptions) *Parser {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	return &Parser{
		md:      md,
		options: opts,
	}
}

// ParseDir recursively parses all markdown files in a directory.
func (p *Parser) ParseDir(dir string) (*ScanResult, error) {
	result := &ScanResult{
		Files: make([]*File, 0),
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving directory path: %w", err)
	}

	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		file, err := p.ParseFile(path)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		// Make path relative to content directory
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			relPath = path
		}
		file.Path = relPath

		result.Files = append(result.Files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ParseFile parses a single markdown file.
func (p *Parser) ParseFile(path string) (*File, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	file := &File{
		Path:    path,
		AbsPath: absPath,
		Texts:   make([]Text, 0),
	}

	// Extract and remove frontmatter
	mdContent, frontmatter := p.extractFrontmatter(content)
	file.Frontmatter = frontmatter

	// Add frontmatter texts if enabled
	if p.options.ExtractFrontmatter && frontmatter != nil {
		if frontmatter.Title != "" {
			file.Texts = append(file.Texts, Text{
				Content: frontmatter.Title,
				Type:    TextTypeFrontmatter,
				Line:    1,
				Context: "page title",
			})
		}
		if frontmatter.Description != "" {
			file.Texts = append(file.Texts, Text{
				Content: frontmatter.Description,
				Type:    TextTypeFrontmatter,
				Line:    1,
				Context: "page description/meta",
			})
		}
	}

	// Parse markdown AST
	reader := text.NewReader(mdContent)
	doc := p.md.Parser().Parse(reader)

	// Walk AST and extract texts
	p.walkNode(doc, mdContent, file)

	return file, nil
}

// extractFrontmatter separates YAML frontmatter from markdown content.
func (p *Parser) extractFrontmatter(content []byte) ([]byte, *Frontmatter) {
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return content, nil
	}

	// Find closing ---
	rest := content[4:] // Skip opening ---\n
	endIndex := bytes.Index(rest, []byte("\n---"))
	if endIndex == -1 {
		return content, nil
	}

	yamlContent := rest[:endIndex]
	mdContent := rest[endIndex+4:] // Skip \n---

	// Skip newline after closing ---
	if len(mdContent) > 0 && mdContent[0] == '\n' {
		mdContent = mdContent[1:]
	} else if len(mdContent) > 1 && mdContent[0] == '\r' && mdContent[1] == '\n' {
		mdContent = mdContent[2:]
	}

	var fm Frontmatter
	if err := yaml.Unmarshal(yamlContent, &fm); err != nil {
		// Invalid YAML, return original content
		return content, nil
	}

	return mdContent, &fm
}

// walkNode recursively extracts text from AST nodes.
func (p *Parser) walkNode(node ast.Node, source []byte, file *File) {
	switch n := node.(type) {
	case *ast.Heading:
		if p.options.ExtractHeadings {
			text := p.extractText(n, source)
			if p.shouldInclude(text) {
				file.Texts = append(file.Texts, Text{
					Content: text,
					Type:    TextTypeHeading,
					Line:    p.getLineNumber(n, source),
					Context: fmt.Sprintf("h%d heading", n.Level),
				})
			}
		}

	case *ast.Paragraph:
		// Skip paragraphs inside blockquotes (handled separately)
		if _, ok := n.Parent().(*ast.Blockquote); ok {
			break
		}
		if p.options.ExtractParagraphs {
			text := p.extractText(n, source)
			if p.shouldInclude(text) {
				file.Texts = append(file.Texts, Text{
					Content: text,
					Type:    TextTypeParagraph,
					Line:    p.getLineNumber(n, source),
				})
			}
		}

	case *ast.ListItem:
		if p.options.ExtractListItems {
			text := p.extractText(n, source)
			if p.shouldInclude(text) {
				file.Texts = append(file.Texts, Text{
					Content: text,
					Type:    TextTypeListItem,
					Line:    p.getLineNumber(n, source),
				})
			}
		}
		// Don't recurse into list items, we've extracted the text
		return

	case *ast.Blockquote:
		if p.options.ExtractBlockquotes {
			text := p.extractText(n, source)
			if p.shouldInclude(text) {
				file.Texts = append(file.Texts, Text{
					Content: text,
					Type:    TextTypeBlockquote,
					Line:    p.getLineNumber(n, source),
				})
			}
		}
		// Don't recurse into blockquotes, we've extracted the text
		return

	case *ast.Image:
		if p.options.ExtractImageAlt {
			alt := string(n.Title)
			if alt == "" {
				// Try alt text from children
				alt = p.extractText(n, source)
			}
			if p.shouldInclude(alt) {
				file.Texts = append(file.Texts, Text{
					Content: alt,
					Type:    TextTypeImage,
					Line:    p.getLineNumber(n, source),
					Context: "image alt text",
				})
			}
		}

	case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.CodeSpan:
		// Skip code blocks - don't translate code
		return

	case *ast.HTMLBlock, *ast.RawHTML:
		// Skip raw HTML
		return
	}

	// Recurse into children
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		p.walkNode(child, source, file)
	}
}

// extractText gets the text content from a node and its children.
func (p *Parser) extractText(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	p.collectText(node, source, &buf)
	return strings.TrimSpace(buf.String())
}

// collectText recursively collects text from nodes.
func (p *Parser) collectText(node ast.Node, source []byte, buf *bytes.Buffer) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(source))
		if n.HardLineBreak() || n.SoftLineBreak() {
			buf.WriteByte(' ')
		}
	case *ast.String:
		buf.Write(n.Value)
	case *ast.CodeSpan:
		// Include code spans in text but mark with backticks
		buf.WriteByte('`')
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			p.collectText(child, source, buf)
		}
		buf.WriteByte('`')
		return
	case *ast.Emphasis:
		// Preserve emphasis markers for context
		marker := "*"
		if n.Level == 2 {
			marker = "**"
		}
		buf.WriteString(marker)
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			p.collectText(child, source, buf)
		}
		buf.WriteString(marker)
		return
	case *ast.Link:
		// Extract link text, skip URL
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			p.collectText(child, source, buf)
		}
		return
	case *ast.AutoLink:
		// Skip auto-links (URLs)
		return
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		p.collectText(child, source, buf)
	}
}

// shouldInclude checks if text meets minimum length requirements.
func (p *Parser) shouldInclude(text string) bool {
	if text == "" {
		return false
	}
	if p.options.MinTextLength > 0 && len(text) < p.options.MinTextLength {
		return false
	}
	return true
}

// getLineNumber returns the line number for a node.
func (p *Parser) getLineNumber(node ast.Node, source []byte) int {
	// Try to get line info from the node itself
	pos := p.getNodePosition(node, source)
	if pos > 0 {
		return p.posToLine(pos, source)
	}

	// For inline nodes or nodes without line info, walk up to parent
	parent := node.Parent()
	if parent != nil {
		return p.getLineNumber(parent, source)
	}

	return 0
}

// getNodePosition tries to get the byte position of a node.
func (p *Parser) getNodePosition(node ast.Node, source []byte) int {
	// Check if node has Lines() with data
	if !isInlineNode(node) {
		lines := node.Lines()
		if lines.Len() > 0 {
			return lines.At(0).Start
		}
	}

	// For nodes with children, try first child's text segment
	if first := node.FirstChild(); first != nil {
		if txt, ok := first.(*ast.Text); ok {
			return txt.Segment.Start
		}
		// Recurse into first child
		return p.getNodePosition(first, source)
	}

	return -1
}

// posToLine converts a byte position to a line number.
func (p *Parser) posToLine(pos int, source []byte) int {
	lineNum := 1
	for i := 0; i < pos && i < len(source); i++ {
		if source[i] == '\n' {
			lineNum++
		}
	}
	return lineNum
}

// isInlineNode returns true if the node is an inline element.
func isInlineNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.Text, *ast.String, *ast.CodeSpan, *ast.Emphasis,
		*ast.Link, *ast.Image, *ast.AutoLink, *ast.RawHTML:
		return true
	}
	return false
}

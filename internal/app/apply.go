package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/romshark/icumsg"
	"github.com/romshark/tik/tik-go"

	"github.com/romshark/toki/internal/arb"
	"github.com/romshark/toki/internal/config"
	"github.com/romshark/toki/internal/log"
	"github.com/romshark/toki/internal/markdown"

	"golang.org/x/text/language"
)

// linkPatterns for prefixing internal links with language code
var (
	// Markdown links: [text](/path) or [text](/path "title")
	mdLinkPattern = regexp.MustCompile(`(\]\()(/[^)"'\s]+)`)
	// YAML link fields: link: "/path" or link: /path
	yamlLinkPattern = regexp.MustCompile(`(link:\s*["']?)(/[^"'\s\n]+)(["']?)`)
	// Hugo relref: {{< relref "/path" >}} or {{% relref "/path" %}}
	relrefPattern = regexp.MustCompile(`(\{\{[<%]\s*relref\s+["'])(/[^"']+)(["']\s*[>%]\}\})`)
)

var (
	ErrMissingSourcePath = errors.New("missing source markdown path (-md)")
	ErrMissingTargetPath = errors.New("missing target output path (-out)")
	ErrMissingTargetLang = errors.New("missing target language (-t)")
	ErrNoTranslations    = errors.New("no translations found in catalog")
	ErrCatalogNotFound   = errors.New("catalog file not found")
)

// Apply implements the `toki apply` command.
// It applies translations from ARB catalogs back to markdown files.
type Apply struct {
	hasher           *xxhash.Digest
	icuTokenizer     *icumsg.Tokenizer
	tikParser        *tik.Parser
	tikICUTranslator *tik.ICUTranslator
}

// ApplyResult holds the results of an apply operation.
type ApplyResult struct {
	Start        time.Time
	FilesWritten int
	TextsApplied int
	TextsSkipped int // Empty translations
	Err          error
}

func (r *ApplyResult) Print() {
	if r.Err != nil {
		log.Error("apply failed", r.Err)
		return
	}
	log.Info("apply complete",
		"files", r.FilesWritten,
		"texts_applied", r.TextsApplied,
		"texts_skipped", r.TextsSkipped,
		"duration", time.Since(r.Start).String())
}

func (a *Apply) Run(
	osArgs, env []string, stderr io.Writer, now time.Time,
) (result ApplyResult) {
	result.Start = now

	conf, err := config.ParseCLIArgsApply(osArgs)
	if err != nil {
		result.Err = fmt.Errorf("%w: %w", ErrInvalidCLIArgs, err)
		return result
	}

	log.SetWriter(stderr, false)
	switch {
	case conf.QuietMode:
		log.SetMode(log.ModeQuiet)
	case conf.VerboseMode:
		log.SetMode(log.ModeVerbose)
	}

	// Validate required parameters
	if conf.MarkdownPath == "" {
		result.Err = ErrMissingSourcePath
		return result
	}
	if conf.OutputPath == "" {
		result.Err = ErrMissingTargetPath
		return result
	}
	if conf.TargetLocale == language.Und {
		result.Err = ErrMissingTargetLang
		return result
	}

	// Load source (English) ARB catalog
	sourceARBPath := filepath.Join(conf.BundlePath, fmt.Sprintf("catalog_%s.arb", conf.SourceLocale))
	sourceARB, err := a.loadARB(sourceARBPath)
	if err != nil {
		result.Err = fmt.Errorf("loading source catalog: %w", err)
		return result
	}

	// Load target (translated) ARB catalog
	targetARBPath := filepath.Join(conf.BundlePath, fmt.Sprintf("catalog_%s.arb", conf.TargetLocale))
	targetARB, err := a.loadARB(targetARBPath)
	if err != nil {
		result.Err = fmt.Errorf("loading target catalog: %w", err)
		return result
	}

	// Build translation map: source text -> translated text
	translations := make(map[string]string)
	for id, targetMsg := range targetARB.Messages {
		if targetMsg.ICUMessage == "" {
			result.TextsSkipped++
			continue
		}
		sourceMsg, ok := sourceARB.Messages[id]
		if !ok {
			continue
		}
		// Unescape ICU message format back to plain text
		sourceText := unescapeICUMessage(sourceMsg.ICUMessage)
		targetText := unescapeICUMessage(targetMsg.ICUMessage)
		translations[sourceText] = targetText
	}

	if len(translations) == 0 {
		result.Err = ErrNoTranslations
		return result
	}

	log.Info("loaded translations", "count", len(translations))

	// Parse source markdown files
	mdParser := markdown.NewParser()
	scanResult, err := mdParser.ParseDir(conf.MarkdownPath)
	if err != nil {
		result.Err = fmt.Errorf("parsing markdown: %w", err)
		return result
	}

	// Process each file
	for _, file := range scanResult.Files {
		outPath := filepath.Join(conf.OutputPath, file.Path)

		// Read original file
		content, err := os.ReadFile(file.AbsPath)
		if err != nil {
			log.Error("reading file", err, "path", file.AbsPath)
			continue
		}

		// Apply translations
		translated := a.applyTranslations(string(content), file, translations)

		// Prefix internal links with language code if enabled
		if conf.PrefixLinks {
			translated = a.prefixLinks(translated, conf.TargetLocale)
		}

		// Ensure output directory exists
		outDir := filepath.Dir(outPath)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			log.Error("creating directory", err, "path", outDir)
			continue
		}

		// Write translated file
		if err := os.WriteFile(outPath, []byte(translated), 0o644); err != nil {
			log.Error("writing file", err, "path", outPath)
			continue
		}

		result.FilesWritten++
		log.Verbose("wrote file", "path", outPath)
	}

	result.TextsApplied = len(translations) - result.TextsSkipped
	return result
}

func (a *Apply) loadARB(path string) (*arb.File, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrCatalogNotFound, path)
		}
		return nil, err
	}
	defer f.Close()

	decoder := arb.NewDecoder()
	return decoder.Decode(f)
}

// applyTranslations replaces source texts with translations in the content.
// It uses line-based matching to preserve markdown structure.
func (a *Apply) applyTranslations(content string, file *markdown.File, translations map[string]string) string {
	result := content

	// Sort texts by line number in reverse order so we can replace from bottom up
	// This prevents offset issues when replacing text
	texts := make([]markdown.Text, len(file.Texts))
	copy(texts, file.Texts)

	// Sort descending by line number
	for i := 0; i < len(texts)-1; i++ {
		for j := i + 1; j < len(texts); j++ {
			if texts[i].Line < texts[j].Line {
				texts[i], texts[j] = texts[j], texts[i]
			}
		}
	}

	for _, text := range texts {
		translation, ok := translations[text.Content]
		if !ok || translation == "" {
			continue
		}

		// Handle frontmatter specially
		if text.Type == markdown.TextTypeFrontmatter {
			result = a.replaceFrontmatterField(result, text.Content, translation)
			continue
		}

		// For body text, do a simple string replace
		// This works because the text was extracted from the original content
		result = strings.Replace(result, text.Content, translation, 1)
	}

	return result
}

// replaceFrontmatterField replaces a field value in YAML frontmatter.
func (a *Apply) replaceFrontmatterField(content, oldValue, newValue string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// End of frontmatter
				break
			}
		}

		if !inFrontmatter {
			continue
		}

		// Check if this line contains the old value
		if strings.Contains(line, oldValue) {
			// Replace the value, preserving quotes if present
			if strings.Contains(line, `"`) {
				// Quoted value - escape quotes in new value
				escaped := strings.ReplaceAll(newValue, `"`, `\"`)
				lines[i] = strings.Replace(line, oldValue, escaped, 1)
			} else {
				lines[i] = strings.Replace(line, oldValue, newValue, 1)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// unescapeICUMessage converts ICU message format back to plain text.
// Reverses the escaping done during extraction.
func unescapeICUMessage(s string) string {
	// Remove wrapping quotes if present (for special characters)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	// Unescape double single quotes
	s = strings.ReplaceAll(s, "''", "'")
	return s
}

// prefixLinks adds language prefix to internal links in the content.
// For example, /platform becomes /vi/platform for Vietnamese.
func (a *Apply) prefixLinks(content string, locale language.Tag) string {
	base, _ := locale.Base()
	langCode := base.String()

	// Skip if it's English (typically the default, no prefix needed)
	if langCode == "en" {
		return content
	}

	result := content

	// Prefix markdown links: [text](/path) -> [text](/vi/path)
	result = mdLinkPattern.ReplaceAllStringFunc(result, func(match string) string {
		return prefixLinkMatch(match, mdLinkPattern, langCode, 1, 2, "")
	})

	// Prefix YAML link fields: link: "/path" -> link: "/vi/path"
	result = yamlLinkPattern.ReplaceAllStringFunc(result, func(match string) string {
		return prefixLinkMatch(match, yamlLinkPattern, langCode, 1, 2, 3)
	})

	// Prefix Hugo relref: {{< relref "/path" >}} -> {{< relref "/vi/path" >}}
	result = relrefPattern.ReplaceAllStringFunc(result, func(match string) string {
		return prefixLinkMatch(match, relrefPattern, langCode, 1, 2, 3)
	})

	return result
}

// prefixLinkMatch handles prefixing a single link match.
// prefixIdx is the submatch index of the prefix, pathIdx is the path, suffixIdx is the suffix (or "" for none).
func prefixLinkMatch(match string, pattern *regexp.Regexp, langCode string, prefixIdx, pathIdx int, suffixIdx any) string {
	submatches := pattern.FindStringSubmatch(match)
	if len(submatches) <= pathIdx {
		return match
	}

	path := submatches[pathIdx]

	// Skip external links (shouldn't match our patterns, but be safe)
	if strings.HasPrefix(path, "http") {
		return match
	}

	// Skip already-prefixed links (e.g., /vi/platform)
	// Check if path starts with a known language code
	if isAlreadyPrefixed(path) {
		return match
	}

	// Skip static assets
	if strings.HasPrefix(path, "/images/") || strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/css/") || strings.HasPrefix(path, "/js/") ||
		strings.HasPrefix(path, "/fonts/") {
		return match
	}

	// Build new path with language prefix
	newPath := "/" + langCode + path

	prefix := submatches[prefixIdx]
	suffix := ""
	if idx, ok := suffixIdx.(int); ok && idx < len(submatches) {
		suffix = submatches[idx]
	}

	return prefix + newPath + suffix
}

// isAlreadyPrefixed checks if a path already has a language prefix.
// Matches common 2-letter and some 3-letter language codes.
func isAlreadyPrefixed(path string) bool {
	// Path should be like /xx/... where xx is a language code
	if len(path) < 4 || path[0] != '/' {
		return false
	}

	// Find the second slash
	secondSlash := strings.Index(path[1:], "/")
	if secondSlash == -1 {
		return false
	}

	potentialCode := path[1 : secondSlash+1]

	// Common language codes (2-3 chars)
	commonCodes := map[string]bool{
		"en": true, "de": true, "fr": true, "es": true, "it": true,
		"pt": true, "ru": true, "zh": true, "ja": true, "ko": true,
		"ar": true, "hi": true, "th": true, "vi": true, "id": true,
		"nl": true, "pl": true, "tr": true, "sv": true, "da": true,
		"fi": true, "no": true, "cs": true, "el": true, "he": true,
		"hu": true, "ro": true, "uk": true, "bg": true, "hr": true,
		"sk": true, "sl": true, "et": true, "lv": true, "lt": true,
		"ms": true, "fil": true, "bn": true, "ta": true, "te": true,
	}

	return commonCodes[potentialCode]
}

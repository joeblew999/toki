package config

import (
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/text/language"
)

type ConfigWebedit struct {
	Host          string
	BundlePkgPath string
	DontOpen      bool
}

type ConfigGenerate struct {
	Locale          language.Tag
	Translations    []language.Tag
	ModPath         string
	TrimPath        bool
	JSON            bool
	QuietMode       bool
	VerboseMode     bool
	BundlePkgPath   string
	RequireComplete bool
	// Markdown scanning options
	MarkdownPath       string   // Path to markdown content directory
	MarkdownExtensions []string // Which elements to extract (headings, paragraphs, etc.)
	MarkdownOnly       bool     // Only process markdown, skip Go source scanning
}

var ErrLocaleNotBCP47 = errors.New("must be a valid non-und BCP 47 locale")

func ParseCLIArgsWebedit(osArgs []string) (*ConfigWebedit, error) {
	c := &ConfigWebedit{}

	cli := flag.NewFlagSet(osArgs[0], flag.ExitOnError)
	cli.BoolVar(&c.DontOpen, "dontopen", false, "disables automatic browser opening")
	cli.StringVar(&c.Host, "host", "localhost:52000",
		"HTTP server host address")
	cli.StringVar(&c.BundlePkgPath, "b", "tokibundle",
		"path to generated Go bundle package")

	if err := cli.Parse(osArgs[2:]); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	return c, nil
}

// ParseCLIArgsGenerate parses CLI arguments for command "generate"
func ParseCLIArgsGenerate(osArgs []string) (*ConfigGenerate, error) {
	c := &ConfigGenerate{}

	var locale string
	var translations strArray
	var mdExtensions strArray

	cli := flag.NewFlagSet(osArgs[0], flag.ExitOnError)
	cli.StringVar(&locale, "l", "",
		"default locale of the original source code texts in non-und BCP 47")
	cli.Var(&translations, "t",
		"translation locale in non-und BCP 47 "+
			"(multiple are accepted and duplicates are ignored). "+
			"Creates new catalogs for locales for which no catalogs exist yet.")
	cli.StringVar(&c.ModPath, "m", ".", "path to Go module")
	cli.BoolVar(&c.TrimPath, "trimpath", true, "enable source code path trimming")
	cli.BoolVar(&c.JSON, "json", false, "enables JSON output")
	cli.BoolVar(&c.QuietMode, "q", false, "disable all console logging")
	cli.BoolVar(&c.VerboseMode, "v", false, "enables verbose console logging")
	cli.StringVar(&c.BundlePkgPath, "b", "tokibundle",
		"path to generated Go bundle package relative to module path (-m)")
	cli.BoolVar(&c.RequireComplete, "require-complete", false,
		"fails the command if any active catalog has a completeness < 1.0 (under 100%)")
	// Markdown options
	cli.StringVar(&c.MarkdownPath, "md", "",
		"path to markdown content directory (enables markdown scanning)")
	cli.Var(&mdExtensions, "md-ext",
		"markdown elements to extract: headings, paragraphs, lists, blockquotes, images, frontmatter "+
			"(default: all). Multiple -md-ext flags can be used.")
	cli.BoolVar(&c.MarkdownOnly, "md-only", false,
		"only process markdown files, skip Go source code scanning (for non-Go projects like Hugo)")

	if err := cli.Parse(osArgs[2:]); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	if locale != "" {
		var err error
		c.Locale, err = language.Parse(locale)
		if err != nil {
			return nil, fmt.Errorf(
				"argument l=%q: %w: %w", locale, ErrLocaleNotBCP47, err,
			)
		}
		if c.Locale == language.Und {
			return nil, fmt.Errorf(
				"argument l=%q: %w: is und", locale, ErrLocaleNotBCP47,
			)
		}
	}

	slices.Sort(translations)
	translations = slices.Compact(translations)
	// Ignore any duplicate of locale in translations.
	// It will be filtered out later during missing catalog detection.
	// We don't do it here because -l is optional when the bundle package exists.
	c.Translations = make([]language.Tag, len(translations))
	for i, s := range translations {
		var err error
		c.Translations[i], err = language.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("argument t=%q: %w: %w", s, ErrLocaleNotBCP47, err)
		}
		if c.Translations[i] == language.Und {
			return nil, fmt.Errorf("argument t=%q: %w: is und", s, ErrLocaleNotBCP47)
		}
	}

	// Process markdown extensions
	c.MarkdownExtensions = mdExtensions

	return c, nil
}

// ConfigApply holds configuration for the apply command.
type ConfigApply struct {
	SourceLocale language.Tag // Source language (default: en)
	TargetLocale language.Tag // Target language to apply
	MarkdownPath string       // Path to source markdown content
	OutputPath   string       // Path to write translated markdown
	BundlePath   string       // Path to tokibundle directory with ARB files
	QuietMode    bool
	VerboseMode  bool
}

// ParseCLIArgsApply parses CLI arguments for command "apply"
func ParseCLIArgsApply(osArgs []string) (*ConfigApply, error) {
	c := &ConfigApply{
		SourceLocale: language.English,
	}

	var sourceLang, targetLang string

	cli := flag.NewFlagSet(osArgs[0], flag.ExitOnError)
	cli.StringVar(&sourceLang, "l", "en",
		"source locale (default: en)")
	cli.StringVar(&targetLang, "t", "",
		"target locale to apply translations for (required)")
	cli.StringVar(&c.MarkdownPath, "md", "",
		"path to source markdown content directory (required)")
	cli.StringVar(&c.OutputPath, "out", "",
		"path to write translated markdown files (required)")
	cli.StringVar(&c.BundlePath, "b", "tokibundle",
		"path to bundle directory containing ARB files")
	cli.BoolVar(&c.QuietMode, "q", false, "disable all console logging")
	cli.BoolVar(&c.VerboseMode, "v", false, "enables verbose console logging")

	if err := cli.Parse(osArgs[2:]); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	// Parse source locale
	if sourceLang != "" {
		var err error
		c.SourceLocale, err = language.Parse(sourceLang)
		if err != nil {
			return nil, fmt.Errorf("argument l=%q: %w: %w", sourceLang, ErrLocaleNotBCP47, err)
		}
	}

	// Parse target locale (required)
	if targetLang != "" {
		var err error
		c.TargetLocale, err = language.Parse(targetLang)
		if err != nil {
			return nil, fmt.Errorf("argument t=%q: %w: %w", targetLang, ErrLocaleNotBCP47, err)
		}
	}

	return c, nil
}

type strArray []string

func (l *strArray) String() string {
	return strings.Join(*l, ",")
}

func (l *strArray) Set(value string) error {
	*l = append(*l, value)
	return nil
}

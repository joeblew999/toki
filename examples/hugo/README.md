# Hugo Multilingual Example

This example demonstrates using Toki to manage translations for a Hugo static site with 3 languages:
- English (en) - default
- German (de)
- Chinese (zh)

## Directory Structure

```
examples/hugo/
├── content/
│   ├── en/           # English (source language)
│   │   ├── _index.md
│   │   └── about.md
│   ├── de/           # German translations
│   │   ├── _index.md
│   │   └── about.md
│   └── zh/           # Chinese translations
│       ├── _index.md
│       └── about.md
└── tokibundle/       # Generated ARB files (git-ignored)
    ├── catalog_en.arb
    ├── catalog_de.arb
    └── catalog_zh.arb
```

## Quick Start

```bash
# From toki root directory:

# Generate ARB for English source only
task hugo:generate

# Generate ARB files for all locales (en, de, zh)
task hugo:generate:all

# Clean generated files
task hugo:clean
```

## Manual Usage

```bash
# From this directory (examples/hugo):

# Extract translatable text from English source (markdown-only mode)
toki generate -l en -md ./content/en -md-only -v

# Generate ARB files for all locales
toki generate -l en -md ./content/en -md-only -t de -t zh -v
```

## Translation Workflow

1. **Author content in English** in `content/en/`
2. **Extract translatable strings**: `task hugo:generate:all`
3. **Translate**: Edit the generated `.arb` files in `tokibundle/`
   - `catalog_de.arb` - German translations
   - `catalog_zh.arb` - Chinese translations
4. **Verify completeness**: Check the completeness percentage in command output

## Key Flags

| Flag | Description |
|------|-------------|
| `-md` | Path to markdown content directory |
| `-md-only` | Skip Go source scanning (for non-Go projects) |
| `-l` | Source locale (e.g., `en`) |
| `-t` | Target translation locale (can be repeated) |
| `-v` | Verbose output |

## What Gets Extracted

- Frontmatter `title` and `description`
- Headings (h1-h6)
- Paragraphs
- List items (bulleted and numbered)
- Blockquotes
- Image alt text

## What Gets Skipped

- Code blocks (fenced and indented)
- URLs and paths
- HTML comments

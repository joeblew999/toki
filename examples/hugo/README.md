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
│   │   ├── about.md
│   │   └── blog/
│   │       ├── _index.md
│   │       └── company-update.md  # Blog post with dates
│   ├── de/           # German translations
│   │   ├── _index.md
│   │   ├── about.md
│   │   └── blog/
│   │       ├── _index.md
│   │       └── company-update.md
│   └── zh/           # Chinese translations
│       ├── _index.md
│       ├── about.md
│       └── blog/
│           ├── _index.md
│           └── company-update.md
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

## Date Handling (Issue #18)

The blog post example (`blog/company-update.md`) demonstrates date handling challenges:

### Date Locations

| Location | Format | Example |
|----------|--------|---------|
| Front matter `date` | RFC3339/ISO8601 | `2024-12-15T09:00:00+00:00` |
| Front matter `lastmod` | RFC3339/ISO8601 | `2024-12-16T14:30:00+00:00` |
| Inline text | Human-readable | "December 1, 2024" |
| Footer text | Human-readable | "Published: December 15, 2024" |

### Timezone Considerations

The example shows timezone offsets in front matter:
- English: `+00:00` (UTC)
- German: `+01:00` (CET)
- Chinese: `+08:00` (CST)

### Translation Challenges

1. **Front matter dates**: Should NOT be translated (Hugo handles display formatting)
2. **Inline dates**: Need locale-aware translation
   - EN: "December 1, 2024"
   - DE: "1. Dezember 2024"
   - ZH: "2024年12月1日"
3. **Times with zones**: Need conversion, not just translation
   - EN: "3:00 PM UTC"
   - DE: "16:00 Uhr MEZ"
   - ZH: "北京时间23:00"

### Current Status

- Front matter dates are preserved correctly (not extracted for translation)
- Inline dates are extracted and translated by translation services
- Time zone conversion is not automated (see issue #18)

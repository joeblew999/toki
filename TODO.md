# Toki Fork Status

## Completed

- [x] **Markdown support** (issue #13) - `feat/markdown-support` branch
  - Added `-md` flag for markdown content directory
  - Added `-md-only` flag for non-Go projects (Hugo)
  - Parses frontmatter (title, description)
  - Extracts headings, paragraphs, lists, blockquotes
  - Skips code blocks and URLs
  - ICU message escaping for apostrophes and curly braces
  - Handles Hugo shortcodes like `{{< relref >}}` in content
  - Hugo example with 3 languages (en, de, zh)
  - Tested on ubuntu-website: 2853 unique texts extracted

- [x] **Taskfile improvements**
  - All examples work via `task verify`
  - Easy run tasks: `task simple:run`, `task complex:run`, `task simpleweb:run`

- [x] **Apply command** - `toki apply`
  - Applies translations from ARB catalogs back to markdown files
  - Usage: `toki apply -md content/english -out content/vietnamese -t vi`
  - Reads source ARB (en) and target ARB (vi), matches by message ID
  - Replaces text in markdown files with translations
  - Preserves markdown structure and frontmatter
  - `-prefix-links` flag to add language prefix to internal links
  - Skips Hugo relref shortcodes (Hugo resolves language automatically)

- [x] **Nested YAML front matter extraction**
  - Extracts translatable text from nested structures: `banner.title`, `features[].content`, etc.
  - Smart field detection: knows `title`, `content`, `label` are translatable
  - Skips non-translatable fields: `link`, `image`, `enable`, `date`, etc.
  - Context path preserved: `features[0].button.label` for translator reference

- [x] **Timezone conversion** (issue #18) - `feat/timezone-conversion` branch
  - `MatchWithOptions()` for timezone-aware time formatting
  - `LocaleToTimezone()` returns default timezone for 30+ locales
  - Converts UTC times to local time before formatting
  - Tests verify: UTC 15:00 → NYC 10:00, Berlin 16:00, Tokyo 00:00

## Upstream Issues

| Issue | Title | Status |
|-------|-------|--------|
| #13 | Translate markdown | **Done** - our PR |
| #16 | webedit doesn't exit on port conflict | **PR #21 submitted** |
| #17 | webedit ports can get stuck | Upstream fixing |
| #19 | webedit is slow with many translations | Upstream fixing |
| #18 | Time zone conversion | Feature request |
| #12 | -json output for multi-lang generate | Works in our fork |

## Known Issues (Our Fork)

*All previously known issues have been resolved!*

## Date/Time Handling (Issue #18)

**Hugo example added:** `examples/hugo/content/*/blog/company-update.md`

### Date Locations in Hugo Markdown

| Location | Format | Toki Behavior |
|----------|--------|---------------|
| Front matter `date` | RFC3339: `2024-12-15T09:00:00+00:00` | **Preserved** (not extracted) |
| Front matter `lastmod` | RFC3339: `2024-12-15T09:00:00+00:00` | **Preserved** (not extracted) |
| Inline dates | Human: "December 1, 2024" | **Extracted** (translated by service) |
| Times with zones | "3:00 PM UTC" | **Extracted** (needs conversion) |

### What Works Now

1. **Front matter dates preserved** - toki doesn't extract `date` or `lastmod` fields
2. **Inline dates translated** - DeepL/Claude handle locale formatting:
   - EN: "December 1, 2024" → DE: "1. Dezember 2024" → ZH: "2024年12月1日"

### What Needs Work (Issue #18)

**Time zone conversion** - When a post says "3:00 PM UTC", translations should convert:
- EN: "3:00 PM UTC"
- DE: "16:00 Uhr MEZ" (UTC+1)
- ZH: "北京时间23:00" (UTC+8)

**Options to explore:**
1. **Shortcode approach**: `{{< time "2025-01-20T15:00:00Z" >}}` - Hugo handles conversion
2. **Toki marker**: `[time:2025-01-20T15:00:00Z]` - toki detects and converts
3. **Translation hint**: Add metadata to ARB for translator context
4. **Leave to translator**: Current behavior - human/AI handles conversion

## Future Ideas

- Use Claude for AI-assisted translations with ARB format
- DataStar webedit (upstream prefers HTMX)
- Extract nested YAML front matter for Hugo Plate themes 



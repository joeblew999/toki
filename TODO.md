# Toki Fork Status

## Completed

- [x] **Markdown support** (issue #13) - `feat/markdown-support` branch
  - Added `-md` flag for markdown content directory
  - Added `-md-only` flag for non-Go projects (Hugo)
  - Parses frontmatter (title, description)
  - Extracts headings, paragraphs, lists, blockquotes
  - Skips code blocks and URLs
  - ICU message escaping for apostrophes
  - Hugo example with 3 languages (en, de, zh)

- [x] **Taskfile improvements**
  - All examples work via `task verify`
  - Easy run tasks: `task simple:run`, `task complex:run`, `task simpleweb:run`

## Upstream Issues

| Issue | Title | Status |
|-------|-------|--------|
| #13 | Translate markdown | **Done** - our PR |
| #16 | webedit doesn't exit on port conflict | Upstream fixing |
| #17 | webedit ports can get stuck | Upstream fixing |
| #19 | webedit is slow with many translations | Upstream fixing |
| #18 | Time zone conversion | Feature request |
| #12 | -json output for multi-lang generate | Works in our fork |

## Future Ideas

- Date/time locale awareness (issue #18)
- Use Claude for AI-assisted translations with ARB format
- DataStar webedit (upstream prefers HTMX) 



---
title: Markdown Kitchen Sink
author: Benjamin Connelly
maintained_by: human
tags: [test, markdown, fur]
date: 2026-06-11
---

# H1 — Markdown Kitchen Sink

This file exercises every markdown construct `fur` renders, so you can flip
themes with `ctrl+t` and check each element. This opening paragraph is
deliberately written across several editor-wrapped source lines so you can
confirm soft-wrapped prose reflows to the pane width instead of stranding
short fragments on their own lines.

## H2 — Inline Text Styles

Plain text, then **bold**, then *italic*, then ***bold italic***, then
~~strikethrough~~, then `inline code` (should be a distinct color with no
background block), then a mix: a **bold `code` span** and an *italic `code`
span* to verify they stay visually separate.

Escapes: \*not italic\*, \`not code\`, \# not a heading, and a literal
backslash \\ here.

Hard line break below (two trailing spaces):
First line of the break.  
Second line of the break.

### H3 — Links and Autolinks

- Inline link: [fur on GitHub](https://github.com/Benjamin-Connelly/fur)
- Link with title: [hover me](https://example.com "Tooltip text")
- Reference link: [reference-style link][ref]
- Bare autolink: <https://example.com/autolink>
- Wikilink: [[some-note]] and aliased [[some-note|Display Name]]
- Relative doc link: [see the readme](./README.md#installation)

[ref]: https://example.com/reference "Reference destination"

#### H4 — Images

![Alt text for an image](https://example.com/image.png "Image title")

##### H5 — Heading Five

###### H6 — Heading Six

## H2 — Lists

### Unordered (with wrapping + nesting)

- A long first item that runs well past the available width so you can verify
  that continuation lines reflow correctly and that there is a blank line of
  breathing room before the next item rather than everything being squashed
  together into one dense block.
- Second item with inline `code`, **bold**, and a [link](https://example.com).
- Nested list:
  - Child item one
  - Child item two, also long enough to wrap so nested continuation behaviour
    is visible at narrow widths
    - Grandchild item
- Back to the top level.

### Ordered

1. First ordered item.
2. Second ordered item, long enough to wrap across the pane so ordered-list
   reflow and spacing can be checked too.
3. Third ordered item.
   1. Nested ordered child.
   2. Another nested child.

### Task list (GFM)

- [x] Completed task
- [ ] Incomplete task
- [ ] Task with `code` and **bold** and a long description that wraps to make
  sure task items reflow like the others

## H2 — Blockquotes

> A single-line blockquote.

> A multi-line blockquote that is soft-wrapped across several source lines to
> confirm blockquote text renders correctly and is not mangled by the
> soft-break unwrapper.
>
> > A nested blockquote inside the first.
>
> Back to the outer quote.

## H2 — Code

Inline: run `go build -o fur ./cmd/fur` to build.

Fenced Go:

```go
package main

import "fmt"

func main() {
	// indentation and blank lines inside fences must be preserved
	for i := 0; i < 3; i++ {
		fmt.Printf("line %d\n", i)
	}
}
```

Fenced Python:

```python
def greet(name: str) -> str:
    return f"hello, {name}"


print(greet("world"))
```

Fenced Bash:

```bash
#!/usr/bin/env bash
set -euo pipefail
for f in *.md; do
  echo "rendering $f"
done
```

Fenced JSON:

```json
{
  "name": "fur",
  "themes": ["catppuccin-mocha", "gruvbox", "nord"],
  "nested": { "a": 1, "b": [true, false, null] }
}
```

Fence with no language tag:

```
plain preformatted text
    with leading spaces preserved
```

Indented code block (four spaces):

    indented code line one
    indented code line two

## H2 — Tables

| Feature        | Status | Notes                          |
|----------------|:------:|--------------------------------|
| Headings       |   ✓    | H1–H6                          |
| Inline code    |   ✓    | distinct color, no background  |
| Tables         |   ✓    | alignment: left, center, right |
| Long cell text |   ✓    | wraps if the table is wide     |

Alignment check:

| Left | Center | Right |
| :--- | :----: | ----: |
| a    |   b    |     c |
| aaa  |  bbb   |   ccc |

## H2 — Definition List

Term one
: Definition of the first term, written long enough to wrap.

Term two
: Definition of the second term.
: A second definition for the same term.

## H2 — Horizontal Rules

Above the rule.

---

Between rules.

***

Below the rules.

## H2 — Footnotes (GFM)

Here is a statement with a footnote.[^1] And another one.[^note]

[^1]: The first footnote definition.
[^note]: A named footnote with `code` and **bold** inside it.

## H2 — Inline HTML

This paragraph contains <strong>inline HTML strong</strong> and
<em>inline HTML emphasis</em>.

<div>
  A raw HTML block.
</div>

## H2 — Edge Cases

A paragraph immediately followed by a list with no blank line between them:
- item right after a paragraph
- second item

Consecutive emphasis: **a***b* and `code1` `code2` `code3`.

Unicode and emoji: café, naïve, → ← ↑ ↓, ✓ ✗, 🚀 📝 🔥.

A very-long-unbreakable-token-to-test-overflow:
supercalifragilisticexpialidocioussupercalifragilisticexpialidocioussupercalifragilistic

The end.

# Design Review Prompt — FUR Brand Identity

> **Instructions**: Paste everything below into a capable LLM (Claude, GPT-4/5,
> Gemini) for a second opinion on the FUR brand work. The prompt is
> self-contained — it embeds the critical SVG/ASCII inline so the reviewer
> can work from text alone. If the reviewer has filesystem or GitHub access,
> paths and URLs are provided under **Asset Inventory** at the end.

---

## Your Role

You are a **senior visual design reviewer** with deep experience in three
overlapping domains:

1. **CLI / TUI tool branding** — neofetch / fastfetch distro logos, bat, exa,
   ripgrep, Charmbracelet's gum / vhs / glow, lazygit, gh. You know what
   reads as "developer tool" vs "consumer app."
2. **ASCII art craft** — the 2ch / Shift-JIS tradition (Mona, Samamine),
   Joan Stark's work, BBS ANSI scene archives (16colo.rs), modern Unicode
   block rendering (fastfetch half-block trick). You can tell amateur ASCII
   from crafted ASCII at a glance.
3. **Print / editorial branding** — you know Penguin Classics, Charley
   Harper's mid-century realism, Saul Bass silhouettes, Steinlen's *Le Chat
   Noir* 1896 poster, Art Nouveau simplification, vintage Halloween
   die-cuts, Memphis Group post-modernism.

Your job is to give a **critical, prioritized, specific** second opinion on
the brand identity work described below. Be honest about weaknesses. Do not
hedge. Use concrete language ("the bookmark ribbon at x=74 is floating in
dead space") rather than vague praise ("nice composition"). Cite specific
pieces by number.

## What Project You're Reviewing

**FUR** is an open-source terminal-native tool:

- **Tagline**: *Further Reading*
- **Function**: A modern replacement for the Unix `cat` command — a
  markdown navigator with TUI (Bubble Tea) and web (net/http) modes,
  inter-document link navigation, backlinks, broken-link detection,
  syntax highlighting, git awareness.
- **Language**: Pure Go, cross-compiles to linux/darwin × amd64/arm64.
- **Audience**: Developers, sysadmins, tech writers who live in the
  terminal and read a lot of markdown.
- **Version**: v1.0.1 (shipped). This brand work is for post-1.0
  identity / README / future README hero / favicon / CLI startup banner.
- **Repo**: https://github.com/Benjamin-Connelly/fur
- **Status**: Production-grade Go codebase, 478 tests across 14 packages.
  The code is done; the brand is the gap.

The mascot concept — established before this round — is a **scholarly cat**
(the Unix `cat` pun, elevated). The tagline anchors the mark in
*reading*, *scholarship*, *linked documents*.

## Design Journey So Far

1. **Round 1** — "Bracket Specs" direction: feline head with chunky
   reading glasses shaped like terminal brackets `[ ]`, `>_` prompt mouth,
   text-cursor `|` pupils. Palette: neon pink `#FF5F87`, deep purple
   `#B267E6`, dark slate `#1E1E1E`. Vibe: Dark Igloo subversive + VHS
   nostalgia + Charmbracelet polish.
2. **Round 2** — User asked to explore broader directions. Generated 16
   mockups across concept groups: Art History Homages (Steinlen, Harper,
   Wain), Terminal Ecosystem (NES pixel, cat-in-terminal, tail-as-cursor),
   Character Play (monocle, calico), Spooky/Retro (Halloween, tattoo
   flash, scaredy cat), Geometric (low-poly, Kuroneko badge), and one
   Scholar/Reading (books stack).
3. **Round 3** — User shortlisted: **loves Scholar's Stack**, also likes
   Harper Modular + Chat Noir. Runners-up: Vintage Halloween, Scaredy Cat,
   Ink Noir. Asked for a new palette that better suits "a bookish terminal
   tool" than the neon original.
4. **Round 4** — Introduced the "Bindery" palette (below), promoted
   Scholar's Stack to hero, retouched top 6 with new colors.
5. **Round 5** — User called out weak ASCII work. Did proper research on
   2ch / Joan Stark / Samamine traditions and rewrote all 16 ASCII
   companions using `∧` ears, kaomoji faces, minimal block usage.

## Current State

### The "Bindery" Palette

| Token | Hex | Role |
|---|---|---|
| Ink | `#1E1E1E` | Outlines, deepest dark |
| Oxblood | `#8B2635` | Warm primary — leather book spines, noses, drips |
| Forest | `#2D5A3D` | Cool secondary — hunter-green bookcloth, eyes |
| Brass | `#C9A961` | Metallic — gilt edges, glasses frames, bookmarks |
| Parchment | `#F5ECD9` | Page cream, highlights |
| Ginger | `#D97543` | Tabby cat fur |
| Sienna | `#A84A1F` | Tabby stripes, warm accent |

**Rationale**: evoke a well-worn reading room — Penguin Classics leather,
brass reading lamp, aged paper, ginger cat napping on a book stack.
Timeless, scholarly, warm. Replaces neon VHS palette.

### Hero Logo — `logo.svg` (Scholar's Stack direction)

Ginger tabby with brass round reading glasses perched atop a 3-book
stack (oxblood / forest-green / tan spines, oxblood bookmark ribbon).
Forest-green eyes. 100×100 viewBox.

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <!-- Book 1 (bottom, oxblood) -->
  <rect x="8"  y="74" width="84" height="16" fill="#8B2635"/>
  <rect x="8"  y="87" width="84" height="1.5" fill="#E8CF8D"/>
  <rect x="18" y="78" width="38" height="1"   fill="#C9A961"/>
  <rect x="18" y="82" width="28" height="1"   fill="#C9A961"/>
  <!-- Book 2 (middle, forest green) -->
  <rect x="12" y="58" width="76" height="16" fill="#2D5A3D"/>
  <rect x="12" y="71" width="76" height="1.5" fill="#E8CF8D"/>
  <rect x="22" y="62" width="32" height="1"   fill="#C9A961"/>
  <rect x="22" y="66" width="26" height="1"   fill="#C9A961"/>
  <!-- Book 3 (top, warm tan) -->
  <rect x="18" y="42" width="64" height="16" fill="#D4A855"/>
  <rect x="18" y="55" width="64" height="1.5" fill="#6B4F1D"/>
  <rect x="26" y="46" width="28" height="1"   fill="#2D5A3D"/>
  <rect x="26" y="50" width="22" height="1"   fill="#2D5A3D"/>
  <!-- Oxblood bookmark ribbon -->
  <rect x="74" y="42" width="2.5" height="22" fill="#8B2635"/>
  <polygon points="72,64 77,64 74.5,68" fill="#8B2635"/>
  <!-- Cat head (ginger tabby) -->
  <path d="M 30 42 L 30 34 Q 30 30 34 30 L 38 18 L 44 32 L 56 32
           L 62 18 L 66 30 Q 70 30 70 34 L 70 42 Z" fill="#D97543"/>
  <path d="M 38 38 L 62 38 L 60 42 L 40 42 Z" fill="#F0D4A8"/>
  <g stroke="#A84A1F" stroke-width="1.2" stroke-linecap="round" fill="none">
    <path d="M 37 22 L 39 30"/><path d="M 41 26 L 42 32"/>
    <path d="M 63 22 L 61 30"/><path d="M 59 26 L 58 32"/>
  </g>
  <polygon points="40,22 42,30 44,30" fill="#E8A8B8"/>
  <polygon points="60,22 58,30 56,30" fill="#E8A8B8"/>
  <!-- Brass reading glasses -->
  <circle cx="43" cy="38" r="4" fill="#F5ECD9"/>
  <circle cx="57" cy="38" r="4" fill="#F5ECD9"/>
  <circle cx="43" cy="38" r="4" fill="none" stroke="#C9A961" stroke-width="1.5"/>
  <circle cx="57" cy="38" r="4" fill="none" stroke="#C9A961" stroke-width="1.5"/>
  <line x1="47" y1="38" x2="53" y2="38" stroke="#C9A961" stroke-width="1.5"/>
  <line x1="39.2" y1="38.3" x2="33" y2="36" stroke="#C9A961" stroke-width="1.2"/>
  <line x1="60.8" y1="38.3" x2="67" y2="36" stroke="#C9A961" stroke-width="1.2"/>
  <!-- Forest eyes -->
  <circle cx="43" cy="38" r="1.5" fill="#2D5A3D"/>
  <circle cx="57" cy="38" r="1.5" fill="#2D5A3D"/>
  <circle cx="43.7" cy="37.3" r="0.5" fill="#F5ECD9"/>
  <circle cx="57.7" cy="37.3" r="0.5" fill="#F5ECD9"/>
  <!-- Oxblood nose -->
  <polygon points="48,42 52,42 50,45" fill="#8B2635"/>
</svg>
```

### Runner-up #1 — Chat Noir

Steinlen 1896 homage. Pure ink silhouette, brass glowing almond eyes
(gaslight reference), oxblood ribbon at the tail tip.

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <path d="M 32 10 L 38 34 L 46 30 L 54 30 L 62 34 L 68 10
    L 74 38 Q 80 44 78 52 L 76 78 Q 80 88 70 90 L 64 90
    L 62 84 L 56 84 L 54 90 L 46 90 L 44 84 L 38 84 L 36 90
    L 30 90 Q 20 88 24 78 L 22 52 Q 20 44 26 38 Z" fill="#1E1E1E"/>
  <ellipse cx="40" cy="48" rx="3" ry="5.5" fill="#C9A961" transform="rotate(-12 40 48)"/>
  <ellipse cx="60" cy="48" rx="3" ry="5.5" fill="#C9A961" transform="rotate(12 60 48)"/>
  <ellipse cx="40" cy="48" rx="0.9" ry="3.8" fill="#1E1E1E" transform="rotate(-12 40 48)"/>
  <ellipse cx="60" cy="48" rx="0.9" ry="3.8" fill="#1E1E1E" transform="rotate(12 60 48)"/>
  <path d="M 78 70 Q 96 60 92 42 Q 90 30 82 34"
        stroke="#1E1E1E" stroke-width="5.5" stroke-linecap="round" fill="none"/>
  <circle cx="82" cy="34" r="2" fill="#8B2635"/>
</svg>
```

### Runner-up #2 — Harper Modular

Charley Harper mid-century overlapping flat shapes. Oxblood triangular
ears, forest-green inverted-triangle head, parchment slit-pupil almonds,
brass tabby stripes on a parchment page.

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <rect width="100" height="100" fill="#F5ECD9"/>
  <polygon points="10,32 50,94 90,32" fill="#2D5A3D"/>
  <polygon points="10,32 22,4 38,32" fill="#8B2635"/>
  <polygon points="62,32 78,4 90,32" fill="#8B2635"/>
  <polygon points="18,22 25,10 30,26" fill="#F0D4A8"/>
  <polygon points="70,22 75,10 82,26" fill="#F0D4A8"/>
  <rect x="20" y="38" width="60" height="2.5" fill="#C9A961"/>
  <rect x="26" y="44" width="48" height="2"   fill="#C9A961" opacity="0.8"/>
  <ellipse cx="36" cy="56" rx="6.5" ry="8" fill="#F5ECD9"/>
  <ellipse cx="64" cy="56" rx="6.5" ry="8" fill="#F5ECD9"/>
  <rect x="35" y="50" width="2" height="12" fill="#1E1E1E"/>
  <rect x="63" y="50" width="2" height="12" fill="#1E1E1E"/>
  <polygon points="45,72 55,72 50,79" fill="#8B2635"/>
  <rect x="44" y="83" width="12" height="2" fill="#C9A961"/>
</svg>
```

### ASCII Companions (sample, hero pieces)

The ASCII is written in the **2ch / Joan Stark tradition**: `∧` (U+2227
logical-AND) for ears — the canonical Japanese ASCII cat ear glyph,
pointier than `/\`. Kaomoji faces like `(=^·.·^=)` for the standard face
grammar. Blocks used only for deliberate silhouettes (#02, #04, #11).
Everything else is regular typable ASCII.

**#14 Scholar's Stack** (lead, matches hero logo):

```
         ∧_____∧
        ( ◉ _ ◉ )
         \_____/
           U U
     ╔═══════════════╗
     ║  K&R · C      ║
     ╠═══════════════╣
     ║  SICP         ║
     ╠═══════════════╣
     ║  cat(1)       ║
     ╠═══════════════╣
     ║  fur(1) ← new ║
     ╚═══════════════╝
```

**#06 Terminal Tenant** (classic kaomoji inside terminal chrome):

```
  ┌ ● ● ○ ─── fur ──┐
  │ ~/docs/          │
  │                  │
  │      ∧___∧       │
  │     (=^·.·^=)    │
  │     (")_(")      │
  │                  │
  │ $ fur README.md ▮│
  └──────────────────┘
```

**#01 Bracket Specs** (the original direction, still a fallback):

```
      ∧_____∧
     (         )
     | [_]-[_] |
     |    ·    |
     |   >_    |
      \_______/
        U U
```

## Full Mockup Inventory

16 SVGs at `.github/design/assets/mockups/NN-<name>.svg` — all 100×100 viewBox.
Paired ASCII in `ascii.js` (all 16, keyed by id).

| # | Name | Vibe | Status |
|---|---|---|---|
| 01 | Bracket Specs v2 | Scholarly cat, terminal-bracket glasses, `>_` mouth | Legacy neon palette |
| 02 | **Chat Noir** | Steinlen silhouette, brass eyes | Bindery palette ✅ |
| 03 | **Harper Modular** | Charley Harper flat geometric | Bindery palette ✅ |
| 04 | NES Pixel | 8-bit sprite | Legacy neon palette |
| 05 | Monocle Scholar | Asymmetric, one monocle + chain | Legacy neon palette |
| 06 | Terminal Tenant | Cat inside terminal window + prompt | Legacy neon palette |
| 07 | Tail Cursor | Minimal cat, tail becomes cursor block | Legacy neon palette |
| 08 | Wain Stare | Louis Wain hypnotic concentric eyes | Legacy neon palette |
| 09 | Calico Patchwork | Tricolor asymmetric patches | Legacy neon palette |
| 10 | Scaredy Cat | Pilo-erect spiky arched fur | Bindery palette ✅ |
| 11 | Vintage Halloween | 1930s die-cut fanged face | Bindery palette ✅ |
| 12 | Tattoo Flash | Sailor Jerry bold outline + banner | Legacy neon palette |
| 13 | Low-Poly Prism | Triangulated faceted | Legacy neon palette |
| **14** | **Scholar's Stack** | **Cat on labeled book stack** | **Bindery ★ HERO** |
| 15 | Kuroneko Badge | Walking silhouette in oval | Legacy neon palette |
| 16 | Ink Noir | Sumi-e brush + ink splatter + drips | Bindery palette ✅ |

## What I Want From You

Please deliver in this order:

### 1. **Critique of the hero** (Scholar's Stack v2)

- Does it read at 16px favicon size? (Test mentally by squinting.)
- Does the cat + book stack composition balance, or is it top-heavy / bottom-heavy?
- Are the three book spines distinguishable? Is the bookmark ribbon earning its space?
- Is the ginger tabby the right fur color, or should it be different (gray tabby, pure black, cream)?
- Do the brass round glasses land, or do they dissolve at small sizes?
- **Verdict**: ship as-is, refine specific elements, or reject the direction entirely?

### 2. **Critique of the Bindery palette**

- Is this palette coherent? Too warm? Too folky?
- Does it fit a **terminal CLI tool** specifically? (Terminals often show
  white-on-black or white-on-dark. How does the palette survive?)
- Compare to: Charmbracelet's pink/purple, exa's green, bat's neutral,
  NixOS's blue diamond. Where does Bindery sit on that map?
- Would you kill any color? Add any?

### 3. **Critique of the runners-up**

- Chat Noir: too generic "black cat" or genuinely distinctive?
- Harper Modular: does the forest-face + oxblood-ears combo read Harper,
  or does it read generic geometric cat?
- Between these two, which is the stronger fallback direction?

### 4. **Critique of the ASCII work**

- Given the 2ch / Joan Stark canon you know, does this ASCII land?
- Which of the 3 embedded samples (#14, #06, #01) is strongest / weakest?
- Are there cheap improvements I'm missing (specific character swaps,
  tighter alignment, different kaomoji)?

### 5. **Anything I'm not seeing**

What would a seasoned CLI-tool designer tell me to reconsider? Is the
entire cat-on-books metaphor the right call for a `cat`-command
replacement? Is there a wordmark direction (logotype instead of mascot)
I should have explored? Is the mascot genre hurting the brand's
"serious tool" credibility?

### 6. **Prioritized action list**

Rank these 5 choices in order of what will most improve the identity:
- (a) Refine Scholar's Stack further, ship as-is
- (b) Pivot to Chat Noir or Harper Modular
- (c) Drop the mascot, build a wordmark / monogram
- (d) Expand the palette (add a third primary)
- (e) Something else (specify)

## Format Your Response

- **Use markdown headers** to match my numbered requests above
- **Lead with the verdict**, then explain
- **Be specific** — cite piece numbers, hex codes, coordinates when
  relevant
- **Be critical** — default to "this is weak because…" rather than
  softening with "it's good but…". I have seen too much sycophantic
  review feedback; I want disagreement and pushback.
- **Total length**: 800–1400 words. Tight is better than comprehensive.

## Anti-Patterns — Please Do Not

- Do not praise the breadth (16 mockups). Breadth ≠ quality.
- Do not suggest generic CLI clichés: hexagons, circuit boards, gradient
  terminal prompts, wireframe globes.
- Do not recommend AI-assistant-style warmth (no smiley cats, no waving
  paws).
- Do not propose a new mascot species (no owls, foxes, bears). The cat
  is anchored by the `cat` pun and is not negotiable.

## Asset Inventory (for reviewers with tool access)

All assets in the repo at:
**https://github.com/Benjamin-Connelly/fur/tree/master/.github/design/assets**

Local paths (if this prompt is fed to a tool-enabled agent on the repo):

```
.github/design/assets/
  logo.svg                          # hero (Scholar's Stack v2)
  variants.js                       # palette doc + mockup descriptions
  ascii.js                          # all 16 ASCII companions
  gallery.html                      # custom TUI-themed SVG × ASCII viewer
  preview.html                      # stock comparison preview
  mockups/
    01-bracket-specs.svg
    02-chat-noir.svg
    03-harper-modular.svg
    04-nes-pixel.svg
    05-monocle-scholar.svg
    06-terminal-tenant.svg
    07-tail-cursor.svg
    08-wain-stare.svg
    09-calico-patchwork.svg
    10-scaredy-cat.svg
    11-vintage-halloween.svg
    12-tattoo-flash.svg
    13-low-poly.svg
    14-scholars-stack.svg
    15-kuroneko-badge.svg
    16-ink-noir.svg
```

Additional repo context (if helpful):

- **Project README & architecture**: `CLAUDE.md` at repo root describes
  the Go codebase in detail.
- **CLI entry point**: `cmd/fur/main.go` (cobra commands: root, serve, cat,
  export, graph, tasks, doctor, mcp, version, completion, gen-man).
- **CLI banner file**: `internal/ui/banner.txt` — meant for `fur --help`
  or `fur version` output.
- **Follow-up issues filed**: lookit-t8a (pick finalist), lookit-kwf
  (README + asset pipeline integration), lookit-2mw (wire banner into
  CLI).

Thank you. Go hard on the critique.

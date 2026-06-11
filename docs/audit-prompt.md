# Comprehensive Test Suite & Security Audit for `fur`

> **Source of truth for the audit.** Bead `lookit-9py` and its children reference this
> file. Keep it in sync with the audit plan; if scope changes, the changes land
> here first, then propagate to bead descriptions.

## Repository

`https://github.com/Benjamin-Connelly/fur` — module `github.com/Benjamin-Connelly/fur`. Work on `audit/test-and-security`. Pure Go, Go 1.25+, no CGO. Stack: Cobra/Viper CLI, Bubble Tea TUI, Goldmark (web) / Glamour (TUI) markdown, Chroma syntax highlighting, Bleve full-text search at `~/.cache/fur/index.bleve`, go-git, `x/crypto/ssh` + `pkg/sftp`, fsnotify (100ms debounce), D3 + Mermaid client-side in web mode. Env var prefix `FUR_*`. Config at `~/.config/fur/config.yaml` (auto-migrates from legacy `~/.config/lookit/`). Per-project config `.fur.toml`/`.fur.yaml` walks up from CWD. Plugin hooks load from `~/.config/fur/plugins/*.yaml`.

### Read these before Phase 0

- `CLAUDE.md` — project conventions, including: "No external web frameworks. stdlib `net/http` only", "Pure Go, no CGO", "All errors handled explicitly. No panics.", bd workflow rules, and a mandatory session-completion protocol (`git pull --rebase` → `bd dolt push` → `git push`).
- `AGENTS.md`, `SECURITY.md` (currently absent — Phase 0 must flag this), `Makefile`, `.golangci.yml`.

### Central security chokepoints called out in `CLAUDE.md`

These get dedicated audit beads, not buried in the per-surface review:

- **`Index.ValidatePath()`** (`internal/index/index.go`) — the shared path security check that web delegates to. If this is wrong, Chains B, F, K all become trivial. Fuzz it directly with adversarial path inputs and assert containment under a synthetic root.
- **`render.Slugify()`** — single source of truth for anchor slugs across TUI and web. **Treat this as a high-value red-team target, not a glue function.** Anywhere a human-readable identifier is coerced into a machine identifier, collisions are wins for an attacker. Fuzz with: Unicode NFC vs NFD pairs (`café` two ways); homograph pairs (Cyrillic `а` U+0430 vs Latin `a` U+0061); case + punctuation + whitespace collapse (`# Quick Start` vs `# Quick-Start` vs `# quick.start!`); empty-slug-producing headings (`# 🚀`, `# ---`, `# !!!`); length-truncation collisions; and headings containing slug-significant characters in non-ASCII forms that may survive a permissive sanitizer. Assert: collision dedupe is deterministic and **not** influenced by heading order in the source document; slugs are NFKC-normalized before comparison; mixed-script headings are flagged or normalized rather than silently producing twin entries.

### Existing baseline

`go test ./...` reports 455 tests across 14 testable packages (down slightly from the 478 noted in CLAUDE.md due to the recent MCP removal). The audit's job in 1.1 is to **fill gaps and add adversarial cases**, not to rewrite from scratch. Phase 0 inventory produces a coverage diff (existing vs. proposed) so we know what's actually new.

## Threat Model

`fur` runs in **multi-user, shared-tenancy environments** — hardened bastions, shared dev boxes, MSP-managed servers, CI runners. Adversary classes:

- **Co-located user** with shell access on the same box. Can write to world-writable dirs (`/tmp`, `/var/tmp`, group-writable project dirs), plant config files (`.fur.toml`/`.fur.yaml`), control symlink targets, stage markdown, set env vars on a victim's shell (sourced rc files, container exec).
- **Directory adversary.** Victim runs `fur` inside a tree the adversary controls (cloned repo, downloaded tarball, NFS mount, shared docs dir).
- **Remote adversary** when SSH browsing is in use — compromised remote host, MITM during first connection, attacker-controlled `~/.ssh/config` entries.
- **Browser-side adversary** when web mode is up — another tab on the same machine, a co-located user hitting loopback, an MITM if the user ever binds non-loopback.
- **Content adversary** — crafted markdown, CSV/JSON/YAML, images, file and directory names containing terminal control sequences or shell metacharacters.

Goal: find every way `fur` can be coerced into **escalating privileges, exfiltrating data, executing code, escaping its intended root, or being weaponized against the user or other tenants on the box.**

## Task tracking with `bd` (beads)

The audit work graph is rooted at `lookit-9py` (top-level epic). Phase structure:

- `lookit-9py.1` — Phase 0 SECURITY-INVENTORY.md (this turn). Blocks every other audit bead.
- `lookit-9py.2.*` — Phase 1 test suite (P1_UNIT/FUZZ/GOLDEN/TUI/WEB/SSH/RACE/COV/CI).
- `lookit-9py.3.*` — Phase 2 audit (per-surface review, tooling pass, chokepoints, chains A–M each with PoC + Fix children).
- `lookit-9py.4.*` — Phase 2 hardening (plugin-trust, argv-exec, filename sanitizer, CSP, bind, cache, symlink, ssh strictness). Each depends on the chain PoCs that justify it.

### Bead conventions for this audit

- **One PR per leaf bead.** Failing-test PoC and fix are separate PRs, separate beads, linked via `bd dep add <fix-bead> <poc-bead>`. The team is fine with red tests on `master` between PoC merge and fix merge — **do not gate PoC tests behind build tags**, land them as real failing tests so they're visible in CI and can't be quietly forgotten.
- **Claim atomically.** `bd update <id> --claim` before starting work, never mid-edit.
- **Hardening beads depend on their chain beads.** Don't land the "plugin-hook trust model" hardening before Chain A has a failing test demonstrating why it's needed — the test is the regression guard for the hardening.
- **Close with a one-line summary that includes the PR or commit.** `bd close lookit-9py.3.5.1 "PR #42: Chain A PoC test"`.
- **Honor the session-completion protocol from `CLAUDE.md`.** At the end of every session: file follow-up issues, run quality gates, update statuses, then `git pull --rebase` → `bd dolt push` → `git push`. Work is not complete until `git status` shows "up to date with origin". Do not leave the audit branch with unpushed commits between sessions — context will drift.

### Persistent memory via `bd remember`

Stamped on 2026-05-25 (see `bd memories` for the full list). Highlights:

- Per-project `.fur.toml/.fur.yaml` MUST NOT reach the plugin loader (verified in Phase 0).
- `Index.ValidatePath()` is the single shared path-security chokepoint.
- `render.Slugify()` is the single source of truth for anchor slugs.
- No network egress in tests.
- Every security finding lands as a failing regression test before any fix.
- Default web bind 127.0.0.1; argv-safe exec; cache dirs 0700 / files 0600; refuse SSH ProxyCommand/Match exec.

### Working rhythm

1. `bd ready --json` — list unblocked tasks.
2. Pick highest-priority ready bead, `bd update <id> --claim`.
3. Do the work. Each bead's exit criterion is in its description — usually "PR open with failing test" or "PR merged with fix + test green".
4. `bd close <id> "<summary>"`.
5. Re-run `bd ready`.

Don't batch many beads into one PR. The graph is the audit log; one bead per reviewable unit keeps it readable.

## Phase 0 — Reconnaissance (stop here for review before writing anything)

1. Read `cmd/fur/`, `internal/`, `Makefile`, `.golangci.yml`, `CLAUDE.md`, `AGENTS.md`, `flake.nix`, `go.mod`, `install.sh`.
2. Inventory the existing test footprint: `go test ./... -list '.*'`, coverage baseline with `go test ./... -coverprofile=cover.out`, identify untested packages and obvious dead branches.
3. Dependency posture: `go mod graph`, latest CI `govulncheck` outcome, osv-scanner (CI or local) for known CVEs in the supply chain.
4. Enumerate every external interface: every flag, subcommand, env var (`FUR_*`), config key, plugin hook type, file format the renderer touches, HTTP route, every `os/exec` call site, every filesystem syscall, every `template.HTML` or raw-HTML pass-through in Goldmark, every place `lipgloss` styles user-controlled bytes (filenames, link text, status messages).
5. Produce **`SECURITY-INVENTORY.md`** at the repo root — the threat-surface ledger that everything else hangs off. **Stop and request review before proceeding.**

## Phase 1 — Testing Suite

Goal: high-confidence coverage on logic packages, behavioral coverage on TUI and web, fuzz coverage on every parser and untrusted-input transformation, race coverage on every concurrent subsystem, golden-file regression for the full rendering pipeline.

### 1.1 Unit & table-driven tests
Cover every `internal/` package. Priority targets: link graph, wikilink resolver, anchor resolver, broken-link detection, fragment validation, fuzzy matcher, BM25/Bleve wrapper, config loader and per-project `.fur` discovery, SSH config parser shim, `host:/path` argument parser, plugin-hook engine, task extractor (`!high`, `#tag`, `@due(...)`), permalink builders (GitHub/GitLab/Bitbucket/Gitea/Codeberg).

`testify/require` for assertions. `t.TempDir()` everywhere — never global `/tmp`. No test depends on the real `$HOME`/`$XDG_CONFIG_HOME` — inject via env and assert isolation.

### 1.2 Fuzz tests (Go native fuzzing)
A `FuzzXxx` for every parser and untrusted-input transformation:

- `FuzzMarkdownRender`, `FuzzWikilinkResolve`, `FuzzAnchorResolve`, `FuzzConfigLoad`, `FuzzHostPathParse`, `FuzzCSVTable`, `FuzzJSONPretty`, `FuzzYAMLFrontmatter`, `FuzzImageInfo`, `FuzzSSHKnownHosts`, `FuzzPermalinkBuild`, **plus** `FuzzValidatePath` and `FuzzSlugify` (chokepoints from CLAUDE.md).

Run fuzzing in CI ≥60s per target on PRs, ≥15m per target nightly.

### 1.3 Golden / snapshot tests
`testdata/golden/` with input fixtures and rendered outputs for: TUI render (drive Bubble Tea with `teatest`), web HTML output, markdown→HTML export, markdown→PDF export (build-tag gated). Include **adversarial fixtures**: raw HTML in markdown, `<script>`, `javascript:` and `data:` URL hrefs, `<iframe>`, SVG with embedded JS, Mermaid blocks attempting `<script>` smuggling, emoji shortcodes that masquerade as HTML, pathologically long lines, deeply nested lists, malformed front-matter, files whose names contain ANSI sequences.

### 1.4 TUI behavioral tests (teatest)
Every keybinding in every preset (default/vim/emacs). Every state transition. Assert clean shutdown via `uber-go/goleak`.

### 1.5 Web mode integration tests
`httptest.Server` for handler unit tests. Spin the real `fur serve` on `:0` for end-to-end and drive with `chromedp` for the D3 graph and Ctrl+K overlay. Assert strict CSP, security headers, ETag stability, SSE backpressure on slow clients, default 127.0.0.1 bind.

### 1.6 SSH/SFTP tests
In-process SSH server using `x/crypto/ssh` server side (already a project dep) + `pkg/sftp` server in a goroutine. Never hit a real host. Cover auth chain order, TOFU, known_hosts pinning, host-key change rejection, agent-forwarding boundaries, cache permissions + atomic writes.

### 1.7 Race & leak detection
`go test -race ./...` clean. `goleak` in `TestMain` for every package with goroutines.

### 1.8 Coverage gates
≥85% on `internal/` excluding pure rendering glue, ≥60% on `cmd/fur/`. Enforced in CI.

### 1.9 CI
GitHub Actions matrix: linux/amd64, linux/arm64, darwin/arm64; Go 1.25 and Go tip. Jobs: `lint` (golangci-lint with existing config plus `gosec`, `errcheck`, `bodyclose`, `noctx`, `gocritic`), `test`, `race`, `fuzz-short`, `vuln` (`govulncheck`), `coverage`. Nightly: `fuzz-long`, `osv-scanner`, dependency review.

## Phase 2 — Security Audit

Produce **`SECURITY-AUDIT.md`** at the repo root. Structure: one section per attack surface, then multi-chain scenarios, then a remediation backlog ranked Critical/High/Medium/Low with a fix recommendation and a regression test for each finding.

### Attack surfaces (per-surface review topics)

1. **Filesystem traversal** — every `os.Open`/`os.Stat`/`filepath.Walk`/`os.Readlink` taking a user-influenced path. Symlink follow behavior. TOCTOU between `Stat` and `Open` for fsnotify.
2. **Wikilinks & anchor resolution** — `[[../../etc/passwd]]`, URL-encoded variants, `[link](file:///etc/passwd)`.
3. **Config discovery walk-up** — `.fur.toml`/`.fur.yaml` walked up from CWD is the **git-config-style attack class**. Plugin hooks are the prize.
4. **Plugin hooks** — exact mechanism, sandbox boundaries, ANSI/HTML smuggling.
5. **Markdown rendering pipeline** — Goldmark extensions, raw HTML, GFM autolink schemes, Glamour ANSI handling, Chroma CVEs.
6. **Web mode** — bind, CSP, SSE auth, directory listing scope, path traversal per route, search overlay XSS, D3 graph injection, Mermaid `<script>` smuggling, custom CSS `@import` exfil, print stylesheet leaks.
7. **SSH/SFTP** — `known_hosts` policy, ProxyCommand/Match exec behavior, agent forwarding, IdentityFile expansion.
8. **Cache & state directories** — permissions, predictability, symlink attacks, cache poisoning persistence.
9. **Shell-out paths** — `$EDITOR`, `xdg-open`/`open`, chromium/wkhtmltopdf, clipboard helpers.
10. **Permalink builder** — `ssh://-oProxyCommand=evil` or `https://example.com/$(...)`.
11. **Stdin pipe** (`cat file | fur`) — escape injection from upstream tooling.
12. **`fur doctor`** — secret leakage, network egress.
13. **Shell completions `--install`** — rc-file mutation, symlink-following.
14. **`fur gen-man`** — write path, mode bits, symlink-following.
15. **`fur export`** — embedded CSS path traversal, output dir validation.

### Multi-chain scenarios (A–M)

For each chain: write a **failing integration test** that demonstrates the chain end-to-end against current `master`, then propose the fix.

- **Chain A** — Hostile-repo terminal hijack via per-project config or plugin hook.
- **Chain B** — Symlink escape via recent files (TOCTOU between Stat and Open).
- **Chain C** — Web mode same-origin exfil when bound non-loopback.
- **Chain D** — Mermaid → JS → fetch-local script smuggling.
- **Chain E** — SSH `ProxyCommand` RCE via planted `~/.ssh/config`. (Phase 0 verifies whether fur honors ProxyCommand.)
- **Chain F** — Cache poisoning across sessions.
- **Chain G** — Filename → exec injection via `$EDITOR`/`xdg-open`.
- **Chain H** — Bleve index disclosure on multi-user box.
- **Chain I** — Permalink builder → git remote URL → exec injection. (Phase 0 verifies go-git argv safety.)
- **Chain J** — ANSI escape via filename in TUI file tree.
- **Chain K** — Web mode path traversal across every route.
- **Chain L** — Env-var config pivot (`FUR_*` overrides in shared envs).
- **Chain M** — Slug collision → anchor hijack → content swap (homograph, cross-platform desync, empty-slug, broken-link "fix" trap).

### 2.3 Tooling pass

Run and triage findings from: `govulncheck`, `staticcheck`, `gosec`, `golangci-lint` with security linters enabled, `semgrep --config p/golang --config p/security-audit`, `osv-scanner -r .`, `syft . -o cyclonedx-json | grype`. For web mode: `nuclei` against `http://127.0.0.1:7777`, ZAP baseline scan.

### 2.4 Hardening recommendations

In `SECURITY-AUDIT.md`:

- **Per-project config trust model** — first-run prompt, `.fur.trusted` allowlist, or global setting.
- **Symlink containment** — `--follow-symlinks` opt-in, refuse to cross filesystem boundaries unless explicit.
- **Filename sanitizer** — single chokepoint that strips terminal control sequences before any user-controlled string hits stdout or the HTML pipeline.
- **Strict CSP** — nonce-based, no `unsafe-inline`, no `unsafe-eval`. Mermaid via pinned SRI hash or vendored locally.
- **Bind hardening** — default `127.0.0.1`, refuse `0.0.0.0` without `--listen-public` and a startup warning.
- **Cache hardening** — `0700` dirs, `0600` files, atomic writes via `os.CreateTemp` + `os.Rename` in the same dir.
- **SSH strictness** — refuse `ProxyCommand` and `Match exec` unless `--allow-ssh-exec` is passed.
- **Argv-safe exec everywhere** — `exec.Command(name, args...)` with separate args, never `sh -c`.

## Phase 3 — Deliverables on `audit/test-and-security`

1. `SECURITY-INVENTORY.md` — surface ledger from Phase 0.
2. `SECURITY-AUDIT.md` — full findings, ranked, with PoC tests and fix proposals.
3. `docs/threat-model.md` — adversary model, trust boundaries, data flows.
4. `internal/**/*_test.go` — the test suite from Phase 1.
5. `testdata/` — golden files and adversarial fixtures.
6. `.github/workflows/audit.yml` — CI gates (lint, race, fuzz-short, govulncheck, coverage).
7. One small, reviewable draft PR per Chain A–M: each either (a) demonstrates the chain with a failing test or (b) implements the fix and turns the test green.

## Workflow expectations

- **First commands in any session resuming the audit:** `bd prime`, then `bd list --status open` to find the current state of the graph.
- **Phase 0 first.** Already in progress on this branch (`lookit-9py.1`); blocks every Phase 1/2/Hardening leaf.
- **Work `bd ready` style.** Pick highest-priority unblocked bead, claim atomically with `bd update --claim`, do the work, open the PR, close the bead with the PR reference in the summary.
- **One reviewable unit per bead.** Don't batch chains into one PR.
- **Failing test before fix, always.** PoC PR lands a failing test on `master`. Fix PR turns it green. No build-tag gating.
- **No network egress in tests.** Everything in-process or `t.TempDir()`. `httptest.Server` is fine.
- **Surface breaking changes as separate beads** with a `breaking-change` label and a migration note in the description.
- **End every session per `CLAUDE.md`'s mandatory protocol.** File follow-up issues → run quality gates → update bead statuses → `git pull --rebase` → `bd dolt push` → `git push` → verify `git status` shows "up to date with origin".

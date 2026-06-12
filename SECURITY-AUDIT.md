# SECURITY-AUDIT.md

> **Phase 2 deliverable for the `fur-audit` bead graph (root: `lookit-9py`).**
> Findings, fixes, and regression tests across every external surface. Builds
> on the Phase 0 ledger in [`SECURITY-INVENTORY.md`](SECURITY-INVENTORY.md);
> see [`docs/audit-prompt.md`](docs/audit-prompt.md) for the charter and threat
> model.

| Field | Value |
|---|---|
| Module | `github.com/Benjamin-Connelly/fur` |
| Go version | 1.25 (`go.mod`), CI matrix `1.25`/`1.26` |
| Audit author | bconnelly |
| Method | One reviewable PR per chain; every finding lands a deterministic regression test, and the fix lands back-to-back so `master` stays green and deployable (the fleet dogfoods `origin/master`). |

## Threat model (summary)

`fur` runs in **multi-user, shared-tenancy environments**. Adversary classes:
a co-located shell user (writes world-writable dirs, plants config/symlinks,
sets env vars on a victim shell); a **directory adversary** (victim browses a
tree the attacker controls — cloned repo, tarball, NFS mount); a **remote
adversary** when SSH browsing is used; a **browser-side adversary** when web
mode is up; and a **content adversary** (crafted markdown, data files, file
and directory names carrying terminal control sequences or shell
metacharacters). Goal: find every way `fur` can be coerced into escalating
privileges, exfiltrating data, executing code, escaping its intended root, or
being weaponized against the user or other tenants. Full model in
`docs/audit-prompt.md`.

## Central chokepoints

| Chokepoint | Location | Guarantee | Guard |
|---|---|---|---|
| `Index.ValidatePath` | `internal/index/index.go` | Rejects `..` traversal; resolves symlinks; refuses targets outside root; **returns the resolved path** so callers open exactly what was validated (no TOCTOU re-follow). | `FuzzValidatePath`, `TestValidatePath_*` (lookit-9py.3.3) |
| `render.Slugify` / `render.AnchorSlugs` | `internal/render/markdown.go` | NFKC-normalizes before slugifying; single deterministic dedupe consumed by web TOC, `/__api/document`, and the TUI fragment scroller. | `TestSlugifyNFKCCollision`, `TestAnchorSlugs*` (lookit-9py.3.4) |
| `sanitize.Terminal` | `internal/sanitize/sanitize.go` | Strips ANSI/OSC/CSI and C0/C1 control bytes from attacker-controlled strings before any terminal write. | `FuzzTerminal`, `TestTerminal*` (lookit-9py.4.3) |
| argv-safe exec | repo-wide | No `sh -c`; every `exec.Command` uses separated args; filenames routed through `safeFilenameArg`; grep queries behind `--`. | `internal/audit` `TestNoShellExec`, `TestExecSitesAreKnown` (lookit-9py.4.2) |

## Findings by attack chain

Severity reflects exploitability **before** the fix, in the stated threat
model. All are **Fixed** on `master` with a regression test.

### Critical

| ID | Chain | Finding | Fix | Regression test | PR |
|---|---|---|---|---|---|
| A | Hostile-repo config pivot | `mergeProjectConfig` merged **every** key from a checked-out repo's `.fur.{toml,yaml,yml}`, letting a hostile repo set `server.custom_css` (attacker stylesheet), `server.host` (rebind), or `remotes.*` (attacker SSH target). | Per-project config restricted to a display/UX allowlist (`theme`, `keymap`, `show_hidden`, `ignore`, `scrolloff`, `reading_guide`, `mouse`). | `TestMergeProjectConfig_CustomCSSPivot`, `_RemotesPivot`, `_AllowlistedKeysStillApply` | #33 |
| G | Filename → exec injection | Editor / `xdg-open` received filenames as `argv[1]`; a planted name like `-c` was parsed as a flag by vim/ed/xdg-open. | `safeFilenameArg` prefixes `./` to leading-dash names; argv-form exec. | `TestEditorCmd_*`, `TestOpenSystemCmd_*` | #23 |
| K | Web path traversal | `handleAPIDocument` did a string `..` check + index lookup but never resolved symlinks, so an in-root symlink to an out-of-root file was served. | Delegate every path-accepting route to `Index.ValidatePath`. | `TestHandleAPIDocumentSymlinkEscape*`, `vpenforce_test.go` | #18, #21 |

### High

| ID | Chain | Finding | Fix | Regression test | PR |
|---|---|---|---|---|---|
| B | Symlink escape (directory adversary) | The indexer (`Lstat`-based walk) surfaced symlinks whose target escaped the browse root; the TUI/web then opened `entry.Path` and read out-of-root files (e.g. `notes.md -> ~/.ssh/id_rsa`). | Indexer drops escaping symlinks on OS-backed roots unless `--follow-symlinks`; `ValidatePath` returns the resolved path. | `TestBuild_SymlinkEscape*`, `TestValidatePath_*` | #34 |
| C | Web same-origin exfil | `Server.Start` bound `server.host:port` unconditionally; `0.0.0.0` exposed the file/search/document APIs (the whole tree) to the network. | `ValidateBind` refuses non-loopback unless `--listen-public` (with a warning). | `TestValidateBind`, `TestIsLoopbackHost` | #35 |
| D | Mermaid → JS → fetch-local | `script-src 'unsafe-inline'` + inline `<script>` bodies in templates let injected inline script execute. | Externalize all scripts to `/__static`; drop `'unsafe-inline'`; Mermaid `securityLevel:'strict'`. | `TestCSPScriptSrcNoUnsafeInline`, `TestNoInlineScriptsInTemplates`, `TestMermaidInitStrict` | #36 |
| J | ANSI escape via filename | File/dir names carrying OSC/CSI sequences were rendered verbatim into the TUI tree/status bar, reprogramming the terminal. | `sanitize.Terminal` chokepoint applied at all TUI display sites. | `TestFileListSanitizesAnsiFilename`, `internal/sanitize` suite | #40 |
| 4.1 | Plugin-hook trust | Plugin files auto-loaded from `~/.config/fur/plugins` with no ownership/permission check; an undocumented `command:` field was parsed (latent exec sink). | Owner-only trust gate (refuse group/other-writable, symlinks, foreign owner); `command:` field removed. | `TestTrustedPluginFile`, `TestLoadPlugins_SkipsUntrusted` | #45 |

### Medium

| ID | Chain | Finding | Fix | Regression test | PR |
|---|---|---|---|---|---|
| F / H | Cache poisoning / index disclosure | The Bleve cache (mirrors all browsed content) was `0755` dir / `0664` files; `recent.json` `0644` — readable by other tenants. | Cache clamped to `0700`/`0600` via root-scoped chmod (re-tightened on reopen); `recent.json` written `0600` atomically (temp + rename). | `TestFulltextCacheDirPerms`, `TestFulltextCacheReopenTightensPerms`, `TestRecentFiles_SavePerms` | #38 |
| M | Slug collision → anchor hijack | `Slugify` did no Unicode normalization (NFC vs NFD `café` → different slugs) and dedupe was reimplemented 3× and could diverge between TOC/API/TUI. | NFKC normalization in `Slugify`; single `render.AnchorSlugs` consumed everywhere. | `TestSlugifyNFKCCollision`, `TestAnchorSlugs*`, `TestHandleAPIDocumentSlugsCentralized` | #42 |

### Low / regression guards (verified safe; pinned against regression)

| ID | Chain | Verdict | Guard | PR |
|---|---|---|---|---|
| E | SSH `ProxyCommand` RCE | **Not exploitable.** fur reads only `User`/`Hostname`/`Port`/`IdentityFile` from `~/.ssh/config` and dials directly; `ProxyCommand`/`ProxyJump`/`LocalCommand`/`Match exec` are never read or executed. | `TestSSHConfigKeyAllowlist`, `TestNoExecOrProxyInRemote`, `TestProxyCommandParsedButIgnored` | #37 |
| I | Permalink → git remote → exec | **Not exploitable.** Permalinks are pure string manipulation over go-git (no `git` subprocess); the clipboard copy passes the final URL via stdin. | `TestGitPackageNoExec`, `TestNormalizeRemoteURLHostile` | #39 |
| L | Env-var config pivot | **Not exploitable.** viper's `AutomaticEnv` (no key replacer) does not map nested keys, so `FUR_SERVER_*` cannot pivot `server.*`; only top-level UX keys apply. Docs corrected. | `TestEnv_ServerKeysNotPivotable`, `TestEnv_TopLevelKeyApplies` | #41 |

## Hardening status (lookit-9py.4.*)

| Bead | Hardening | Status |
|---|---|---|
| 4.1 | Plugin-hook trust model | Done (#45) |
| 4.2 | Argv-safe exec audit (repo-wide guard) | Done (#44) |
| 4.3 | Filename sanitizer chokepoint | Done (#40) |
| 4.4 | Strict script CSP (no `unsafe-inline`) | Done (#36) |
| 4.5 | Default loopback bind; `--listen-public` for non-loopback | Done (#35) |
| 4.6 | Cache `0700`/`0600`, atomic writes | Done (#38) |
| 4.7 | Symlink containment | Done (#34) |
| 4.8 | Refuse SSH `ProxyCommand`/`Match exec` | Done — structural (no exec path) + guard (#37) |
| 4.9 | `ValidatePath` delegation in every web path handler | Done (#21) |

## Residual notes & follow-ups

- **`style-src 'unsafe-inline'`** is still permitted (inline `<style>` and
  `style=` attributes remain in templates). CSS injection is lower-severity
  than script execution; nonce-based style CSP and the custom-CSS `@import`
  exfil surface are a follow-up.
- **Mermaid/D3 load from CDNs** (`cdn.jsdelivr.net`, `d3js.org`) at floating
  major versions. SRI-pinning or vendoring is a follow-up (charter §2.4); the
  strict `script-src` already blocks inline injection.
- **`install.sh` is stale** (points at the pre-rename `lookit` repo) and
  **`SECURITY.md` is absent** — both flagged in Phase 0, tracked outside the
  chain graph.
- **Per-project config trust v2** (`.fur.trusted` allowlist / first-run
  prompt) is filed as `lookit-9jm` (P2) for a future opt-in beyond the current
  deny-by-default allowlist.

## Tooling pass (lookit-9py.3.2 / 2.10)

CI runs, on every PR, across the Go 1.25/1.26 matrix and four `GOOS/GOARCH`
targets:

| Tool | How | Result |
|---|---|---|
| `gofmt`, `go vet` | lint job | clean |
| `golangci-lint` v2.11.4 (`gosec`, `errcheck`, `bodyclose`, `staticcheck`, `gocritic`, `unparam`, …) | lint job, pinned action | 0 issues |
| `govulncheck` v1.2.0 | security job, **pinned** (was `@latest`) for reproducibility | see below |
| `go test -race` | test job | clean |

**govulncheck findings (2026-06-12):** two Go standard-library advisories —
`GO-2026-5037` (`crypto/x509`) and `GO-2026-5039` (`net/textproto`) — both
**fixed in go1.26.4**. They are remediated by the toolchain, not a code
change; CI's `setup-go: '1.26'` resolves to the patched 1.26.x on the runner.
No first-party or third-party-module call-path vulnerabilities.

**Pinning (2.10):** rather than a `tools/tools.go` (which would pull the full
transitive trees of gosec/staticcheck/govulncheck into `go.mod` and `go.sum`,
contrary to the repo's minimal-dependency stance), the audit tools are pinned
where they run: `golangci-lint` (bundling gosec + staticcheck) at `v2.11.4`
and `govulncheck` at `v1.2.0` in the workflow.

**Not run in this pass (require external installs — deferred under the
14-day supply-chain quarantine / human-gated install policy):** `semgrep`,
`osv-scanner`, `syft` + `grype`, ZAP/`nuclei` against the web server. These
are dynamic/SCA scanners whose installation needs explicit approval; tracked
as a follow-up rather than blocking the audit.

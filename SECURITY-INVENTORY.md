# SECURITY-INVENTORY.md

> **Phase 0 deliverable for the `fur-audit` bead graph (root: `lookit-9py`).**
> Threat-surface ledger that everything downstream hangs off. See `docs/audit-prompt.md` for the full plan and `bd show lookit-9py.1` for the bead.

| Field | Value |
|---|---|
| Branch | `audit/test-and-security` |
| Base commit | `56e8503` (master @ 2026-05-25T17:31Z, after the dependabot + MCP-removal sweep) |
| Module | `github.com/Benjamin-Connelly/fur` |
| Go version | 1.25 (declared in `go.mod`), CI matrix `1.25`/`1.26` |
| Build tags | none |
| Audit author | bconnelly (claimed `lookit-9py.1` 2026-05-25) |

---

## 1. Repository facts

- Pure Go, no CGO. Cross-compiles to `linux/darwin × amd64/arm64`.
- `SECURITY.md` **is absent** — Phase 0 flags this; recommend adding one in 2.1.
- `Makefile` is clean (post-rename, fur target, parameterized PREFIX). No symlink containment on the install step (`install -m 755` follows symlinks); flagged for hardening.
- `install.sh` is **stale** — points at the old `Benjamin-Connelly/lookit` repo and tries to download `lookit-${OS}-${ARCH}` binaries that were removed by `lookit-bw2`/`lookit-r1h`. `curl ... | sh` would 404. Either delete or rewrite to `go install`. Tracked as a follow-up note, not a chain.
- `flake.nix` declares license `MIT` (fur is private/personal; CLAUDE rule about WTFPL doesn't apply to forks; flagged because license declared once in nix but no `LICENSE` file at repo root — Phase 0 surfaces this as a docs gap, not a security issue).
- `.golangci.yml` enables `gosec` but **excludes** G104, G115, G203, G204, G301, G302, G304, G306, G401, G501, G702, G703, G704, G706. **Notable security-relevant exclusions:**
  - **G204** (subprocess audit) — directly relevant to Chains G + I (filename → exec, permalink → exec). Bead `P2_TOOLS` (lookit-9py.3.2) re-audits these explicitly.
  - **G304** (file path taint) — relevant to Chains B/F/K (path traversal). `P2_VALIDATEPATH` covers it.
  - **G401**/**G501** (insecure crypto). MD5 is used for ETag generation (handlers.go); flag as non-issue (ETags are not security boundaries) but document.

## 2. Test footprint baseline

`go test ./... -list '.*' | wc -l` = **455** tests (down from 478 in CLAUDE.md, due to removal of `internal/mcp/server_test.go` in PR #15 today).

### Per-package counts and coverage

| Package | Tests | Coverage |
|---|---:|---:|
| `cmd/fur` | 5 | **9.4%** |
| `internal/config` | 24 | 67.1% |
| `internal/doctor` | 26 | 84.5% |
| `internal/export` | 24 | 71.4% |
| `internal/git` | 20 | 62.3% |
| `internal/index` | 29 | 64.8% |
| `internal/manpages` | 5 | 75.0% |
| `internal/plugin` | 7 | 84.6% |
| `internal/remote` | 24 | **29.6%** |
| `internal/render` | 33 | 89.4% |
| `internal/tasks` | 7 | 83.0% |
| `internal/tui` | 175 | 48.2% |
| `internal/web` | 76 | 76.0% |
| `internal/web/templates` | 0 | 0.0% (embed dir) |
| `internal/web/static` | 0 | n/a (embed dir) |
| `demo` | 0 | n/a (asset dir) |

### Fuzz test inventory

`go test ./... -list '.*' | grep -c '^Fuzz'` = **0**. Zero native Go fuzz targets exist today. Bead `P1_FUZZ` (lookit-9py.2.2) adds the full slate.

### Lowest-coverage areas (high-priority for Phase 1.1)

1. **`cmd/fur` 9.4%** — almost no command-handler test coverage. Bead 1.1 should add table-driven tests for `resolveRoot`, `resolveRemoteTarget`, `loadConfig`, the stdin pipe path, the completion-install path.
2. **`internal/remote` 29.6%** — SSH/SFTP integration plumbing; bead 1.6 covers via in-process server.
3. **`internal/tui` 48.2%** — TUI behavioral coverage; bead 1.4 via teatest.
4. **`internal/index` 64.8%** — particularly `ValidatePath`, `Lookup`, fulltext index build/close paths. Bead 1.1 + 1.2 (fuzz).
5. **`internal/config` 67.1%** — env-var binding edge cases, per-project walk-up, theme/keymap validation, plugin discovery.

## 3. Dependency posture

Latest CI run on master is **green for `govulncheck`** (run id 26412452560, 2026-05-25). Previously failing on `x/crypto@v0.49.0` (GO-2026-5013..5021), patched today via PR #14 to v0.52.0.

Locally, `govulncheck`/`osv-scanner`/`staticcheck`/`gosec`/`semgrep`/`syft`/`grype` are **not installed**; Phase 2.3 (`P2_TOOLS`) drives that work and tooling install (via the user-confirmation flow per Supply Chain Defense).

### Key pinned versions (high-attack-surface deps)

| Dep | Version | Notes |
|---|---|---|
| `github.com/yuin/goldmark` | v1.8.2 | Markdown parser (web mode HTML render path). Bumped today (PR #8). |
| `github.com/yuin/goldmark-emoji` | v1.0.6 | Emoji rendering (PR #10). |
| `github.com/alecthomas/chroma/v2` | v2.23.1 | Syntax highlighting (TUI + web). Tracked CVEs historically. |
| `github.com/charmbracelet/glamour` | v1.0.0 | Markdown→ANSI (TUI). Verified byte-identical to v0.10.0 in PR #12. |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI runtime. |
| `github.com/blevesearch/bleve/v2` | v2.5.7 | Full-text search index. Cache-disclosure surface (Chain H). |
| `github.com/go-git/go-git/v5` | v5.19.0 | Pure-Go git, no shell-out (PR #9). Relevant to Chain I priority verdict. |
| `golang.org/x/crypto` | v0.52.0 | SSH stack. Patched today (PR #14). |
| `github.com/kevinburke/ssh_config` | (indirect) | SSH config parser. Relevant to Chain E. |
| `github.com/skeema/knownhosts` | (indirect) | `known_hosts` policy. Relevant to Chain E. |
| `github.com/pkg/sftp` | (indirect) | SFTP client. Used by `internal/remote`. |
| `github.com/spf13/viper` | v1.21.0 | Env-var binding via `SetEnvPrefix("FUR")` + `AutomaticEnv()`. Relevant to Chain L. |

## 4. External interfaces

### 4.1 Cobra subcommands

`fur` (root), `serve`, `cat`, `export`, `graph`, `doctor`, `tasks`, `version`, `config` (+ `init`/`path`/`show` children), `gen-man` (hidden), `completion [bash|zsh|fish|powershell]`.

Removed today (PR #15): `mcp`.

### 4.2 Environment variables

**Viper auto-binds every config key via `FUR_*`** (`internal/config/config.go:146`: `v.SetEnvPrefix("FUR"); v.AutomaticEnv()`). So **every key in §4.3** is an attack surface for Chain L.

Explicit `os.Getenv` reads outside the FUR_* envelope:

| Var | Site | Use |
|---|---|---|
| `SSH_AUTH_SOCK` | `internal/remote/conn.go:387` | ssh-agent socket. |
| `SSH_CLIENT`, `SSH_CONNECTION` | `internal/web/server.go:169` | Detect SSH session → suppress `--open` browser. |
| `EDITOR` | `internal/tui/handle_util.go:63` | Open file in editor on `e` key. **Chain G primary exec attack surface.** |
| `TERM`, `TERM_PROGRAM`, `LC_TERMINAL` | `internal/render/image.go:34–43` | Image protocol detection. |
| `TERM`, `COLORTERM`, `COLUMNS`, `LINES` | `internal/doctor/doctor.go:174–195` | Doctor diagnostic output. |
| `SHELL`, `PSModulePath` | `cmd/fur/main.go:778–787` | Shell detection for completion install. |
| `HOME` (implicit via `os.UserHomeDir()`) | `config.ConfigDir()`, `conn.go` known_hosts path | Multiple sites. |
| `XDG_CACHE_HOME` (implicit via `os.UserCacheDir()`) | `~/.cache/fur` resolution | Bleve index path, etc. |

### 4.3 Config schema (every key)

Top-level (`Config` struct, `internal/config/config.go`):

| Key | Type | Default | mapstructure |
|---|---|---|---|
| `root` | string | "." (CWD) | `root` |
| `theme` | string | "auto" | `theme` |
| `keymap` | string | "default" | `keymap` |
| `server` | nested | see ServerConfig | `server` |
| `git` | nested | see GitConfig | `git` |
| `ignore` | []string | `[]` | `ignore` |
| `mouse` | bool | false | `mouse` |
| `reading_guide` | bool | false | `reading_guide` |
| `scrolloff` | int | 0 | `scrolloff` |
| `debug` | bool | false | `debug` |
| `show_hidden` | bool | false (`.git`/`.hg`/`.svn`/`.bzr` always hidden) | `show_hidden` |
| `remotes` | map[string]RemoteConfig | nil | `remotes` |

`ServerConfig`:

| Key | Type | Default | Risk |
|---|---|---|---|
| `port` | int | 7777 | low |
| `host` | string | (varies; check default — Chain C/L pivot if "0.0.0.0") | **high** |
| `no_https` | bool | false | medium |
| `open` | bool | false | low |
| `custom_css` | string | "" | **high** (Chain A→D bridge) |

`GitConfig`: `enabled` (bool), `show_status` (bool), `remote` (string). Low risk.

`RemoteConfig` (per named remote): `host`, `user`, `port`, `path`. Used by `@alias` SCP-style.

### 4.4 HTTP routes (web mode)

Registered in `internal/web/server.go:312–331`:

| Route | Handler | Notes |
|---|---|---|
| `/__static/*` | `http.FileServer(http.FS(staticFS))` | go:embed static; safe (no user input). |
| `/__custom.css` | `handleCustomCSS` (server.go:188) | **Inline validation** (HasPrefix on resolved path) **not** via `ValidatePath`. Diverges from chokepoint. |
| `/__api/files?q=` | `handleAPIFiles` | Fuzzy filename search via `Index.FuzzySearch`. |
| `/__api/search?q=` | `handleAPISearch` | Bleve fulltext; falls back to `git grep -F` / `grep -F` via `exec.CommandContext` (handlers.go:427/429). |
| `/__api/graph` | `handleAPIGraph` | Link graph JSON. |
| `/__api/document?file=...` | `handleAPIDocument` (handlers.go:597) | **Weak path validation:** only `strings.Contains(filePath, "..")` + `idx.Lookup`. Doesn't call `ValidatePath`. Then `filepath.Join(s.idx.Root(), filePath)` and `afero.ReadFile`. **Chain K candidate.** |
| `/__api/tasks` | `handleAPITasks` | Aggregated `tasks.Extract` over indexed markdown. |
| `/__events` | `handleSSE` | Server-sent events broker. |
| `/graph` | `handleGraph` | D3 visualization (HTML). |
| `/*` | `handleRoot` (handlers.go:82) | Dispatcher. **Does** call `ValidatePath` (handlers.go:96) before dispatching to `handleDirectory`/`handleMarkdown`/`handleFile`. |

**Path-validation gap summary** — the chokepoint is honored by the dispatcher but **bypassed by**: `handleCustomCSS` (inline check), `handleAPIDocument` (weaker `strings.Contains("..")`), and `handleAPIFiles`/`handleAPISearch`/`handleAPIGraph`/`handleAPITasks` (which mostly query the in-memory index without re-validating; need explicit per-handler audit in Chain K's PoC).

### 4.5 Plugin hook types

`internal/plugin/plugin.go:13–20`:

```
HookBeforeRender   // Content field may be modified by hooks (prepend/append/replace)
HookAfterRender    // post-render
HookBeforeIndex    // pre-index
HookAfterIndex     // post-index
HookOnNavigate     // navigation events
```

Loaded only from `~/.config/fur/plugins/*.yaml` via `LoadPlugins(configDir)` (`cmd/fur/main.go:976`, `internal/plugin/plugin.go:90`). Trust verdict in §6.

### 4.6 File formats the renderer touches

- Markdown: `.md`, `.markdown`, `.mdown`
- Code (Chroma highlighting in TUI + web): every language Chroma knows.
- Data: `.json` (pretty-print), `.csv`/`.tsv` (table render with delimiter inference)
- Images (info-card only in TUI; full inline in `fur cat`): `.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.webp`, `.svg`, `.ico`
- YAML frontmatter (markdown subset)
- Front-matter: YAML
- Config: YAML (config.yaml), TOML/YAML (.fur.{toml,yaml,yml})
- Plugins: YAML (~/.config/fur/plugins/*.yaml)

## 5. Security chokepoints

### 5.1 `Index.ValidatePath()` (`internal/index/index.go:298`)

```go
func (idx *Index) ValidatePath(relPath string) (string, error) {
    if strings.Contains(relPath, "..") {
        return "", fmt.Errorf("path traversal not allowed")
    }
    absPath := filepath.Join(idx.root, relPath)
    resolved, err := filepath.EvalSymlinks(absPath)
    if err != nil { return "", fmt.Errorf("file not found") }
    if !strings.HasPrefix(resolved, idx.root+string(filepath.Separator)) && resolved != idx.root {
        return "", fmt.Errorf("path escapes index root")
    }
    return absPath, nil
}
```

**Gaps:**

1. `strings.Contains(relPath, "..")` **over- and under-blocks**:
   - Over-blocks legitimate filenames containing `..` (e.g. `notes..draft.md`).
   - Under-blocks URL-encoded variants (`..%2F`, `%2e%2e/`) — input is not decoded before checking. Whether this matters depends on whether the caller already decoded; **needs per-handler audit**.
2. **No NFKC normalization** before the `..` check — Unicode normalization forms could carry path-traversal semantics depending on the OS.
3. **No NUL-byte filter** — `\0` can truncate paths under some libcs / Go runtime exceptions.
4. **TOCTOU** — `EvalSymlinks` resolves at validate-time. The caller then uses the returned `absPath` (which is **the unresolved join**, not the symlink-resolved path). If the symlink target swaps between validate and open, the open path follows the new target. **Chain B is exactly this.**
5. **Inconsistent return** — returns `absPath` (pre-resolve) while the check operates on `resolved` (post-resolve). Callers using the returned path re-traverse symlinks; a symlink-swap race wins.
6. **Error masking** — `fmt.Errorf("file not found")` for any `EvalSymlinks` failure (could be ENOENT, EACCES, broken symlink, etc.). Bad for debugging and conflates security and existence checks.
7. **Caller delegation is partial** — see §4.4: `handleAPIDocument` and `handleCustomCSS` reimplement traversal checks. Phase 0 finding: the "single chokepoint" claim in CLAUDE.md is **aspirational, not actual** today.

### 5.2 `render.Slugify()` (`internal/render/markdown.go:202`)

```go
func Slugify(s string) string {
    s = strings.ToLower(s)
    s = strings.ReplaceAll(s, " ", "-")
    var b strings.Builder
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
            b.WriteRune(r)
        }
    }
    return b.String()
}
```

`HeadingSlugs` dedupe (`markdown.go:216`):

```go
for _, h := range headings {
    base := Slugify(h.Text)
    n := counts[base]
    counts[base]++
    slug := base
    if n > 0 { slug = base + "-" + strconv.Itoa(n) }
    slugs[slug] = true
}
```

**Gaps (all confirm Chain M is exploitable):**

1. **No Unicode normalization** before slugification. NFC `café` (`c-a-f-é`) becomes `caf` (the `é` U+00E9 is stripped). NFD `café` (`c-a-f-e + combining`) becomes `cafe` (the combining mark is stripped; `e` survives). Different inputs produce different slugs, but neither is normalized; both visually identical to a reader.
2. **Latin/Cyrillic homographs slug differently** (Latin `a` → `a`, Cyrillic `а` U+0430 → stripped). Mixed-script headings produce silently asymmetric slugs.
3. **Empty-slug headings collide deterministically**: `# 🚀`, `# !!!`, `# ---` all slug to `-` or `""` after filtering. Multiple such headings within a file collide and resolve by source order.
4. **Dedupe is source-order-dependent.** `ExtractHeadings` returns headings in document order; `HeadingSlugs` loops in that order; `counts[base]` increments deterministically by appearance. Swapping `# Quick Start` and `# Quick-Start` in the source changes which gets `quick-start` and which gets `quick-start-1`. **This is the exact Chain M bug.**
5. **No mixed-script detection** — homograph collision is invisible to a human reviewer.
6. **No length limit** — pathological headings can produce arbitrarily long slugs; not a security issue per se but tracks as a robustness gap.
7. **Documented as "GitHub's heading anchor generation"** — but GitHub uses a meaningfully different algorithm (it preserves non-ASCII letters and casefolds them). The CLAUDE.md comment is a **promise we can't keep without changes**; relates to Chain M's cross-platform-desync variant.

## 6. Per-project config → plugin loader reachability (Chain A verdict)

**Verdict: Per-project `.fur.toml/.fur.yaml` CANNOT directly set the plugin search path.**

Trace:

1. `loadConfig` (`cmd/fur/main.go:926`) calls `config.Load(cfgFile)`.
2. `config.Load` reads `~/.config/fur/config.yaml`, env vars (`FUR_*`), and CLI flags. It calls `mergeProjectConfig` which walks up from CWD looking for `.fur.toml`, `.fur.yaml`, `.fur.yml` and merges those values.
3. `loadConfig` then calls `plugin.LoadPlugins(configDir)` (`main.go:976`) where `configDir = config.ConfigDir()` — and `ConfigDir()` (`config.go:332`) hardcodes `$HOME/.config/fur` (with one-shot migration from `~/.config/lookit`). It does **not** consult `cfg`, env, or per-project files.
4. There is no `plugins:` key in the `Config` struct, and no override mechanism for the plugin directory.

**So Chain A's strongest variant (per-project plugin hook drops an ANSI payload) does NOT apply as currently coded.**

**However, Chain A's mutated variants are live:**

- Per-project config **can override** `server.host`, `server.custom_css`, `server.open`, `server.no_https`, `theme`, `keymap`, `root`, `ignore`, `remotes`, `mouse`, `scrolloff`, `show_hidden`, `reading_guide`, `debug`. Several of these are pivots:
  - `server.custom_css` → arbitrary CSS pulled into the rendered page (Chain A→D bridge; `@import url(file:///...)` data exfil; covert font-loading; webfont-based attacks).
  - `server.host` → bind change; if attacker controls config in a shared workspace, they can pivot fur to listen non-loopback (Chain A→C bridge).
  - `remotes.<name>.host`/`user`/`port`/`path` → could redirect a victim's `fur @docs` to attacker infrastructure (Chain A→E bridge, but blunted because Chain E's RCE path doesn't apply — see §7.2).

**Phase 0 finding to document in the audit report:** Chain A PoC bead (lookit-9py.3.5.1) should target the highest-impact reachable setting — likely `server.custom_css` pointing at an attacker-controlled CSS file, since that gives content-level injection inside the victim's already-trusted web mode.

## 7. `os/exec` call sites

All seven invocations across `cmd/`+`internal/` excluding tests:

| Site | Call | argv shape | User-controlled? | Risk |
|---|---|---|---|---|
| `internal/tui/preview_load.go:151` | `exec.Command(opener, filePath)` | argv-safe (separate args) | `opener` = `"xdg-open"`/`"open"`; `filePath` = entry path from index | **Chain G surface.** Filename can be a path with leading `-` interpreted as a flag by xdg-open's downstream handler. Argv-safe but flag injection possible. |
| `internal/tui/handle_util.go:68` | `exec.Command(editor, filePath)` | argv-safe | `editor` = `$EDITOR` (Chain L pivot); `filePath` = entry path | **Chain G + L primary surface.** EDITOR is untrusted in shared envs; filePath flag injection. |
| `internal/export/export.go:199` | `exec.Command(tool, args...)` | argv-safe | `tool` = wkhtmltopdf or chromium (resolved via `exec.LookPath`); `args` = generated, not user-controlled | Lower risk; tool comes from a fixed allowlist (`export.go:150–155`). Confirm allowlist is closed. |
| `internal/web/server.go:176` | `exec.Command("xdg-open", startURL)` | argv-safe | `startURL` = server-generated `http://addr/{initialFile}` | Safe; URL fully fur-generated. |
| `internal/web/handlers.go:427` | `exec.CommandContext(ctx, "git", "grep", "-n", "--no-color", "-I", "-F", "--", query)` | argv-safe; `--` separator + `-F` (fixed-string) prevents query→flag/regex injection | `query` is user-controlled (search input) | Safe under current flags. `cmd.Dir = s.idx.Root()` (line 431) — scoped to serve root. **Chain K-adjacent**: if root validation slips, grep scope follows. |
| `internal/web/handlers.go:429` | `exec.CommandContext(ctx, "grep", "-rn", "--no-color", "-I", "-F", "--", query, ".")` | argv-safe; `--` + `-F` | `query` user-controlled | Same as above. |
| `internal/doctor/doctor.go:103` | `exec.Command("git", "--version")` | argv-safe | No user input | Safe. |

Plus `exec.LookPath` calls (5 sites) which only resolve binaries (don't execute). All take string literals (`"git"`, `"wkhtmltopdf"`, `"xdg-open"`, `"open"`, etc.).

**Verdict on Chain I priority:** CLAUDE.md's "go-git, no shelling out" claim is **upheld in this snapshot**. Permalink builder (`internal/git/permalink.go`) generates URL strings only and never reaches `exec.Command`. **Chain I remains P1 as a regression-guard PoC** (test that a malicious remote URL like `ssh://-oProxyCommand=evil` cannot reach an exec site via any code path). Do not promote to P0.

## 8. Filesystem touchpoints

Total: **56** invocations across `internal/` + `cmd/` (excluding tests) of `os.{Open,OpenFile,Create,CreateTemp,Stat,Lstat,Readlink,Symlink,MkdirAll,Mkdir,Chmod,Remove,RemoveAll,Rename}`, `filepath.{Walk,EvalSymlinks}`, `os.ReadDir`, and `afero.ReadFile`.

Top contributors:

| Package | Count | Symlink-follow default | Risk surface |
|---|---:|---|---|
| `internal/config/config.go` | 9 | follows | Config discovery walk-up + Viper file open. Chain A pivot. |
| `cmd/fur/main.go` | 8 | follows | Resolve path arg, init/path/show config writers, completion install dest, gen-man output. |
| `internal/export/export.go` | 6 | follows | Export reads + writes (file output, embed assets). |
| `internal/remote/conn.go` | 5 | follows | known_hosts + IdentityFile open. |
| `internal/doctor/doctor.go` | 4 | follows | Diagnostic file checks. |
| `internal/tui/images.go` | 2 | follows | Image protocol probes. |
| `internal/index/fulltext.go` | 2 | follows | Bleve index dir at `~/.cache/fur/index.bleve`. Chain F/H surface. |
| `internal/web/server.go` | 1 | follows | Custom CSS read. |
| `internal/plugin/plugin.go` | 1 | follows | Plugin YAML scan. |
| `internal/manpages/manpages.go` | 1 | follows | Man-page install (`internal/manpages.Install`). |
| `internal/index/index.go` | 1 | follows | The walker (and `EvalSymlinks` in `ValidatePath`). |
| `internal/config/recent.go` | 1 | follows | Recent files persistence. |

**Default symlink-follow behavior**: Go's `os.Open`/`afero.ReadFile` follow symlinks. No call site uses `os.Lstat` to avoid following. **Chain B is exploitable wherever a victim's recent-files list or cache holds a path that an attacker can flip mid-session.** See §11.

## 9. User-controlled bytes → display (lipgloss + HTML)

### 9.1 TUI (lipgloss styling)

Sites where a user-controlled string is concatenated into a lipgloss-styled line **with no ANSI sanitization**:

| Site | Source | Surface |
|---|---|---|
| `internal/tui/filelist.go:343` | `fmt.Sprintf("%s%s %s", indent, icon, node.name)` | **node.name = filename**; rendered with the file-list lipgloss style. Chain J exact vector. |
| `internal/tui/filelist.go:422` | `fmt.Sprintf(" %s %s", icon, entry.RelPath)` | Filtered-list view. Same surface. |
| `internal/tui/handle_normal.go:283` | `"Copied to clipboard: " + entry.RelPath` | Status-bar message after `y`. |
| `internal/tui/preview.go` (multiple) | Heading / link text inside rendered Glamour output | Glamour may sanitize; needs verification. Pure-content path; not filename. |
| `internal/tui/statusbar.go:102–122` | Status fields incl. `connStyle` for SSH state | Mostly literal; remote display string can carry user input via SCP-style host parsing. |

**Lipgloss does not strip ANSI from input** — it wraps content in additional ANSI escapes. So pre-existing ANSI in the input survives. This is the technical basis for Chain J.

### 9.2 Web (template.HTML pass-throughs)

| Site | Source |
|---|---|
| `internal/web/handlers.go:303` | `template.HTML(rendered)` — Goldmark-rendered markdown body |
| `internal/web/handlers.go:350` | `template.HTML(highlighted)` — Chroma syntax-highlighted code |

**Goldmark default escapes raw HTML** (`html.WithEscapeHTML(true)`; no `html.WithUnsafe()` is set in `server.go:123`). Verified safe at this layer. But:

- **Mermaid blocks** are rendered client-side via D3/JS overlay; the `<script>`-smuggling risk is in the client-side dispatcher, not Goldmark.
- **CSV/JSON data preview** paths feed into the HTML template; need to confirm escaping at every entry.
- **Code-highlight HTML** from Chroma is trusted (it produces structured HTML with class names, no inline attrs from user content) but worth a fuzz pass.

## 10. Network surfaces

### 10.1 Web mode

- Bind: `${cfg.Server.Host}:${cfg.Server.Port}` (`server.go:135`). The default value of `cfg.Server.Host` needs to be verified (`config.go DefaultConfig`); **if it is anything other than `"127.0.0.1"` or `"localhost"`, Chain C is wide open by default.**
- TLS: handled outside this audit; `no_https` flag exists but the cert source / cert validation isn't yet enumerated.
- SSE channel `/__events`: no auth, no origin check beyond Same-Origin Policy from the browser. Chain C surface.
- CSP / security headers: present (CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy) per `internal/web/server.go middleware`. **Phase 2 confirms strictness**, in particular whether `unsafe-inline`/`unsafe-eval` are present in the CSP.

### 10.2 SSH/SFTP

- Uses `x/crypto/ssh` + `pkg/sftp`; no shell-out. No ProxyCommand handling (see §7 — only `User`, `Hostname`, `Port`, `IdentityFile` are pulled from `~/.ssh/config`). Chain E PoC is now a **regression-guard test** ensuring this remains the case.
- `known_hosts` via `github.com/skeema/knownhosts`. **Host-key change → rejection** (`conn.go:476`: `if knownhosts.IsHostKeyChanged(err)`). TOFU on unknown host: auto-adds (`conn.go:479–488`). Phase 2 audits whether the TOFU add path is logged / surfaced to the user.
- Auth chain: ssh-agent (`SSH_AUTH_SOCK`) → IdentityFile → fallback. Specifically traced in `conn.go:355–420`.

### 10.3 Other

- `fur doctor` does not perform network egress (verified by grep).
- `manpages.Install` writes locally only.
- No analytics, no telemetry.

## 11. Cache & state directories

| Path | Owner | Mode (current) | Risk |
|---|---|---|---|
| `~/.config/fur/config.yaml` | user | inherited from umask (typically `0644`) | Sensitive if `custom_css` or `remotes` carry secrets; recommend `0600`. |
| `~/.config/fur/plugins/*.yaml` | user | inherited | Plugin trust file. Should be `0600`. |
| `~/.cache/fur/index.bleve/` | user | inherited (Bleve sets its own; need to verify) | **Chain H surface.** Recommend `0700` dir, `0600` files. |
| `~/.cache/fur/remote/` (if remote browsing in use) | user | inherited | **Chain F surface.** Recommend `0700` dir, `0600` files, atomic writes via `os.CreateTemp` + `os.Rename`. |
| Recent files | (verify path) | inherited | **Chain B surface.** |

Phase 2 hardening beads (`HARDEN_CACHE` = `lookit-9py.4.6`, `HARDEN_SYMLINK` = `lookit-9py.4.7`) drive the fixes.

## 12. Proposed test coverage diff (per package)

Numbers reflect **proposed new test counts per package** beyond today's 455.

| Package | Existing | Adds (P1 unit) | Adds (P1 fuzz) | Adds (golden) | Notes |
|---|---:|---:|---:|---:|---|
| `cmd/fur` | 5 | +20 | +1 (HostPathParse) | – | Drive resolveRoot, completion install, gen-man, config init/path/show, loadConfig precedence (CLI > env > project > global). |
| `internal/config` | 24 | +10 | +1 (ConfigLoad yaml-bomb) | – | per-project walk-up, env precedence, plugin discovery isolation regression. |
| `internal/doctor` | 26 | +3 | – | – | Doctor `--json` path, missing tool branches. |
| `internal/export` | 24 | +5 | – | +6 fixtures | Adversarial HTML in exported markdown. |
| `internal/git` | 20 | +6 | +1 (PermalinkBuild) | – | Remote URL parser hostility cases (`ssh://-oProxyCommand=...`). |
| `internal/index` | 29 | +8 | +2 (ValidatePath, fragment) | – | NUL bytes, NFKC, symlink TOCTOU. |
| `internal/manpages` | 5 | +2 | – | – | Install-path symlink safety. |
| `internal/plugin` | 7 | +4 | – | +2 | Plugin hook precedence + ANSI/HTML smuggling fixtures. |
| `internal/remote` | 24 | +12 | +1 (SSHKnownHosts) | – | TOFU, host-key change rejection, ProxyCommand regression guard (Chain E). |
| `internal/render` | 33 | +6 | +3 (Markdown, Anchor, Slugify) | +10 fixtures | NFKC, homograph, empty-slug, source-order-stability tests. |
| `internal/tasks` | 7 | +4 | – | – | Edge cases on `!high`, `#tag`, `@due()`. |
| `internal/tui` | 175 | +20 | – | – (teatest snapshots in 1.4) | Behavioral coverage + goleak. |
| `internal/web` | 76 | +20 | – | +8 fixtures | Per-route validation + CSP/security-header assertions + path-traversal cases. |

Aggregate: **~120 unit tests + 9 fuzz targets + ~26 golden fixtures + the teatest suite**, plus a Chain A–M PoC pair (26 tests).

## 13. Phase 0 findings worth promoting to chain priority

1. **`handleAPIDocument` uses a weaker traversal check than `ValidatePath`.** Phase 0 lifts this from a Chain K subscenario to a primary PoC target. Already implicit in Chain K's description but worth explicit callout in `P2_VALIDATEPATH` (`lookit-9py.3.3`) and `CHAIN_K` (`lookit-9py.3.15`).

2. **`handleCustomCSS` re-implements path containment inline.** Likewise — `P2_VALIDATEPATH` should mandate "every file-serving handler delegates to `ValidatePath`; no inline re-implementations" as a hardening invariant.

3. **`$EDITOR` flow combines Chain G + Chain L.** A co-located attacker who can set `EDITOR` for a victim's shell **and** plant a file with a `-` prefix gets a flag-injection vector even without breaking out of argv-safe exec. PoC for Chain G (`lookit-9py.3.11.1`) should chain through `EDITOR=tee` + filename `-foo` to demonstrate impact.

4. **CSP strictness is currently unknown** (didn't read the full header values yet); `CHAIN_D` and `CHAIN_K` PoCs need to start by capturing the current CSP and asserting against a strict reference.

5. **`SECURITY.md` does not exist.** The audit should add one (linking to the audit graph and threat model) as part of 2.1.

6. **`install.sh` is broken** — file a separate cleanup bead (not a security issue but visible to anyone reading the README; risk is reputational + UX).

## 14. Phase 0 review answers (resolved 2026-05-25)

1. **`SECURITY.md` — real, now.** Use GitHub Private Vulnerability Reporting (free toggle in repo Security settings) with email fallback, 90-day coordinated disclosure default, explicit scope (in: binary behavior; out: OS-level host misconfiguration), no bounty. Filed as **`lookit-bg0`** (non-audit, parallel work, P2, off master).
2. **`install.sh` — rewrite, don't delete.** README still discovers it; lookit references make it actively misleading (worse than absent). Bundled with §3 into **`lookit-61w`** — single "stale-lookit cleanup" PR off master. Not on audit branch.
3. **`AGENTS.md` rename — separate cleanup PR.** Same bead as `lookit-61w`. Keeps audit-branch diffs focused on security work; preserves "one reviewable unit per bead" discipline.
4. **`internal/web/templates` — keep as data; test via rendered output.** No `go:embed` refactor. New bead pattern in Phase 1.3: `internal/web/templates_test.go` renders each template with synthetic data, parses output with `golang.org/x/net/html`, asserts structural invariants (no `<script>` in unexpected places, every `href` HTML-escaped, every user-data attribute in a quoted context, CSP nonce propagation correct). Adversarial fixtures from Chain D + Chain K feed in. Folded into existing 1.3 bead (`P1_GOLDEN`).
5. **Chain E — demoted to P1, kept as regression guard.** Priority measures urgency, not severity-if-true. ProxyCommand isn't honored on master today; the bug doesn't exist. The PoC becomes a *passing* regression-guard test that asserts current behavior. The "fix" bead (`E_FIX`) is **superseded** by a `bd remember` invariant ("fur deliberately reads only User/Hostname/Port/IdentityFile from ~/.ssh/config — ProxyCommand and Match exec are excluded by design. Any PR expanding the SSH config allowlist requires explicit security review."). The P0 slot freed here is reassigned to Chain K (§15).
6. **Tooling — pre-confirm Go-native; defer the rest.** Pre-approved now: `govulncheck`, `staticcheck`, `gosec`. Pinned via **`lookit-9py.2.10`** (new bead) using `tools/tools.go` blank imports so `go.mod` carries the versions and CI installs are reproducible. Deferred to one-by-one confirmation when Phase 2.3 lands: `semgrep` (Python; use Docker image), `osv-scanner` (Go but newish; version-pin at use), `syft`+`grype` (Anchore Docker images, CI-only). `P2_TOOLS` now depends on `lookit-9py.2.10`.

## 15. Rebalancing applied to the bead graph

| Change | Beads affected | Reason |
|---|---|---|
| **Chain K → P0** | `CHAIN_K` (lookit-9py.3.15), `K_POC` (.3.15.1), `K_FIX` (.3.15.2) | Phase 0 found `handleAPIDocument` + `handleCustomCSS` both bypass the chokepoint. Not a "test for traversal" finding — a *systemic violation of a claimed chokepoint*. Active vulnerability on master today. |
| **Chain E → P1, regression-guard label** | `CHAIN_E` (lookit-9py.3.9), `E_POC` (.3.9.1) retitled "regression guard" | ProxyCommand not honored on master; PoC is a passing test that locks in current behavior rather than a red exploit demo. |
| **Chain E Fix closed (superseded)** | `E_FIX` (lookit-9py.3.9.2) closed with --force | Replaced by a `bd remember` invariant. The memory does more long-term work than a one-shot fix bead. |
| **Phase 1.1 (cmd/fur unit) → P0** | `P1_UNIT` (lookit-9py.2.1) | 9.4% coverage on the CLI entry point is unacceptable; promoted to match Phase 1.5/1.6 priority. |
| **Chain A reframe** | `CHAIN_A` (lookit-9py.3.5) retitled "hostile-repo per-project config pivot via custom_css"; `A_POC` (.3.5.1) notes updated | Phase 0 verified the plugin-hook variant is moot. The real attack surface is `server.custom_css` override via `.fur.yaml`. |
| **Chain L composition** | `L_POC` (lookit-9py.3.16.1) notes updated | PoC includes L+K combined chain: env-var pivot of `custom_css` flowing through the chokepoint-bypassing `handleCustomCSS`. |
| **New hardening bead** | `HARDEN_VPENFORCE` (lookit-9py.4.9), depends on `K_POC` | P0. Deliverable: runtime test enumerating every web handler accepting a path-shaped input, asserting each delegates to `Index.ValidatePath` BEFORE any filesystem access. Turns CLAUDE.md's aspirational claim into an enforced invariant. |
| **New Phase 1 bead** | `TOOLS_PIN` (lookit-9py.2.10), `P2_TOOLS` now depends on it | Pin `govulncheck`/`staticcheck`/`gosec` via `tools/tools.go`. Unblocks 80% of Phase 2.3 tooling immediately. |
| **Parallel cleanup beads (off audit graph)** | `lookit-bg0` (SECURITY.md), `lookit-61w` (stale-lookit) | Non-audit parallel work; lives off master to keep audit-branch diff focused. |

### New persistent memories stamped (Phase 0 review)

- *"Every web handler accepting a path-shaped input MUST delegate to `Index.ValidatePath()`. No inline containment, no `strings.Contains('..')` checks. Enforced by handler-enumeration test in `internal/web/` (see HARDEN_VPENFORCE bead)."*
- *"fur deliberately reads only User/Hostname/Port/IdentityFile from `~/.ssh/config` — ProxyCommand and Match exec are excluded by design. Any PR expanding the SSH config allowlist requires explicit security review. Regression guard: lookit-9py.3.9.1."*
- *"Chain A's plugin-hook variant is moot (configDir hardcoded). Chain A's real attack surface is per-project `.fur.toml/.yaml` overriding `server.custom_css`. Chain L composes via `FUR_SERVER_CUSTOM_CSS` env override of the same setting."*
- *"Audit Phase 1.1 (cmd/fur unit tests) is P0, not P1: current coverage is 9.4%."*
- *"Audit tools split: Go-native (govulncheck/staticcheck/gosec) pinned via tools/tools.go; semgrep via Docker; osv-scanner version-pinned at use; syft+grype CI-only Docker."*

## 16. Closure

This inventory is the deliverable for `lookit-9py.1` (Phase 0). The review gate (PR #17) is approved with the §14 decisions and §15 rebalancing applied. Next:

1. **Merge PR #17** → master + audit branch get the inventory + audit prompt + bead graph.
2. **Close `lookit-9py.1`** with the PR ref.
3. **Start Chain M PoC and Chain K PoC in parallel** (both cheap red tests; both demonstrate active bugs against master; landing them in the same session is the moment to confirm the team is genuinely fine with red `master`).
4. **File `lookit-bg0` (SECURITY.md) and `lookit-61w` (stale-lookit cleanup)** as separate small PRs off master, in parallel with the audit. These should land before any chain PoC merges so the visible red tests have a private-reporting path documented.

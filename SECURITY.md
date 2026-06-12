# Security Policy

## Reporting a vulnerability

Please report security issues privately rather than opening a public issue.
Use GitHub's **"Report a vulnerability"** (Security → Advisories) on
[github.com/Benjamin-Connelly/fur](https://github.com/Benjamin-Connelly/fur),
or email the maintainer. Include a description, affected version
(`fur version`), and a reproduction if possible. You'll get an acknowledgement
and a remediation timeline.

## Supported versions

Fixes land on the latest `master`. There is no separate LTS branch.

## Threat model

`fur` is designed to be safe to run in **multi-user, shared-tenancy
environments** (shared dev boxes, bastions, CI runners). It defends against a
co-located shell user, a directory adversary (browsing an attacker-controlled
tree), a remote adversary during SSH browsing, a browser-side adversary in web
mode, and crafted content (markdown, data files, hostile file/dir names).

Hardening already in place (see [`SECURITY-AUDIT.md`](SECURITY-AUDIT.md) for
the full ledger and regression tests):

- Per-project `.fur.{toml,yaml,yml}` is restricted to a display/UX allowlist —
  it cannot set `server.*`, `git.*`, or `remotes.*`.
- The indexer contains symlinks to within the browse root (`--follow-symlinks`
  to opt out); `Index.ValidatePath` is the shared path chokepoint.
- The web server binds loopback only unless `--listen-public` is passed; a
  strict `script-src` CSP (no `unsafe-inline`) is sent.
- File/dir names are stripped of terminal control sequences before display.
- Cache and state files are owner-only (`0700`/`0600`).
- SSH config `ProxyCommand`/`Match exec` are never honored; all subprocess
  exec is argv-safe; plugin hooks load only from owner-owned files and never
  execute commands.

## Scope notes

- Binding non-loopback (`--listen-public`) intentionally exposes the browsed
  tree to the network; do so only on a trusted network.
- Passing `--follow-symlinks` re-enables out-of-root symlink traversal by
  design.

// Mermaid bootstrap, externalized from base.html so the page CSP can drop
// 'unsafe-inline' from script-src (audit Chain D / hardening 4.4).
// securityLevel 'strict' is set explicitly: it sanitizes diagram HTML labels
// and disables click handlers / inline scripts, blocking the
// Mermaid -> JS -> fetch-local smuggling path through a crafted ```mermaid
// block in untrusted markdown.
//
// Mermaid is vendored locally (mermaid.min.js, the self-contained UMD bundle),
// loaded as a classic <script> just before this one, so it defines the global
// `mermaid`. No CDN import — the page CSP is script-src 'self'. Both scripts
// load only on markdown pages that actually contain a diagram (see the
// "scripts" block override in markdown.html), so the 2.6MB bundle never loads
// on the common no-diagram page.
mermaid.initialize({ startOnLoad: true, theme: 'default', securityLevel: 'strict' });

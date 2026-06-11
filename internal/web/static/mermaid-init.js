// Mermaid bootstrap, externalized from base.html so the page CSP can drop
// 'unsafe-inline' from script-src (audit Chain D / hardening 4.4).
// securityLevel 'strict' is set explicitly: it sanitizes diagram HTML labels
// and disables click handlers / inline scripts, blocking the
// Mermaid -> JS -> fetch-local smuggling path through a crafted ```mermaid
// block in untrusted markdown.
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
mermaid.initialize({ startOnLoad: true, theme: 'default', securityLevel: 'strict' });

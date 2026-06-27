package static

import "embed"

// Vendored third-party assets (kept local so the web UI needs no CDN at runtime):
//
//	d3.v7.min.js — D3 v7.9.0 (https://d3js.org, Copyright 2010-2023 Mike Bostock)
//	  source: https://cdn.jsdelivr.net/npm/d3@7.9.0/dist/d3.min.js
//	  sha256: f2094bbf6141b359722c4fe454eb6c4b0f0e42cc10cc7af921fc158fceb86539
//
//	mermaid.min.js — Mermaid v11.6.0 (https://mermaid.js.org, MIT), self-contained
//	  UMD bundle (no runtime chunk loading); loaded only on pages with a diagram.
//	  source: https://cdn.jsdelivr.net/npm/mermaid@11.6.0/dist/mermaid.min.js
//	  sha256: 3a93016a73dc82ba890d919f9bbb176f3da9d98341650c0b517f2595cc68fef8
//
//go:embed *.css *.js *.png *.ico
var Files embed.FS

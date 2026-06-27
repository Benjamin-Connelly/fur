package static

import "embed"

// Vendored third-party asset (kept local so the web UI needs no CDN at runtime):
//
//	d3.v7.min.js — D3 v7.9.0 (https://d3js.org, Copyright 2010-2023 Mike Bostock)
//	  source: https://cdn.jsdelivr.net/npm/d3@7.9.0/dist/d3.min.js
//	  sha256: f2094bbf6141b359722c4fe454eb6c4b0f0e42cc10cc7af921fc158fceb86539
//
//go:embed *.css *.js *.png *.ico
var Files embed.FS

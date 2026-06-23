package static

import "embed"

//go:embed *.css *.js *.png *.ico
var Files embed.FS

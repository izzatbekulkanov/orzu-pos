package web

import "embed"

//go:embed kassa/* admin/* installer/* icons/* manifest.json
var WebFS embed.FS

package web

import "embed"

//go:embed kassa/* admin/* installer/* icons/* shared/* db/* backup/* manifest.json
var WebFS embed.FS

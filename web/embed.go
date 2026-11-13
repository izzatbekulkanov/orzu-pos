package web

import "embed"

//go:embed kassa/* admin/* installer/* icons/*
var WebFS embed.FS

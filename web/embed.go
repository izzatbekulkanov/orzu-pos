package web

import "embed"

//go:embed kassa/* admin/*
var WebFS embed.FS

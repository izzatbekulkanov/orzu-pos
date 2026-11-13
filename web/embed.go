package web

import "embed"

//go:embed kassa/* admin/* installer/*
var WebFS embed.FS

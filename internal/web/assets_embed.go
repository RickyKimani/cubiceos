package web

import (
	"embed"
)

// embeddedAssets contains the compiled CSS/JS for the web UI.
//
//go:embed assets/* assets/css/* assets/js/* assets/icons/*
var embeddedAssets embed.FS

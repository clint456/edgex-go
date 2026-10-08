package embed

import "embed"

// SQLFiles contains the schema migrations for support-mappings.
//
//go:embed sql
var SQLFiles embed.FS

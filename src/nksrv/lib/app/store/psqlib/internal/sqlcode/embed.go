package sqlcode

import "embed"

//go:embed schema
var Schema embed.FS

//go:embed statements
var Statements embed.FS

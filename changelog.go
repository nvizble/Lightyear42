// Package lightyear holds files from the repository root that are embedded
// in the binary (go:embed only reaches a package's own directory).
package lightyear

import _ "embed" // for go:embed

// Changelog is CHANGELOG.md, shown by `lightyear --version`.
//
//go:embed CHANGELOG.md
var Changelog string

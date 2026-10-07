// Package changelog reads lightyear's CHANGELOG.md: one "## [version] — date"
// section per release, newest first.
package changelog

import (
	"regexp"
	"strings"

	lightyear "github.com/nvizble/Lightyear42"
)

// Release is one version's section of the changelog.
type Release struct {
	// Version without a leading "v" (e.g. "1.3.0-canary.10").
	Version string
	Date    string
	// Notes is the section body, in Markdown.
	Notes string
}

// header matches "## [1.2.1] — 2026-10-06" (brackets, "v" and date optional).
var header = regexp.MustCompile(`^##\s+\[?v?([^\]\s]+)\]?(?:\s+[—–-]\s+(\S+))?\s*$`)

// Parse reads the release sections of a changelog, in file order.
func Parse(md string) []Release {
	var releases []Release
	var notes []string
	flush := func() {
		if n := len(releases); n > 0 {
			releases[n-1].Notes = strings.TrimSpace(strings.Join(notes, "\n"))
		}
		notes = nil
	}
	for _, line := range strings.Split(md, "\n") {
		if m := header.FindStringSubmatch(line); m != nil {
			flush()
			releases = append(releases, Release{Version: m[1], Date: m[2]})
			continue
		}
		if len(releases) > 0 {
			notes = append(notes, line)
		}
	}
	flush()
	return releases
}

// Find returns the release of version ("v" prefix optional).
func Find(releases []Release, version string) (Release, bool) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	for _, r := range releases {
		if r.Version == version {
			return r, true
		}
	}
	return Release{}, false
}

// Embedded is lightyear's own changelog, built into the binary.
func Embedded() []Release {
	return Parse(lightyear.Changelog)
}

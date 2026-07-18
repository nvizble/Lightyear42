package update

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// NormalizeVersion ensures a leading "v" for semver helpers.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// IsReleaseVersion reports whether v is a valid semver tag (not "dev").
func IsReleaseVersion(v string) bool {
	return semver.IsValid(NormalizeVersion(v))
}

// CompareVersions returns -1, 0, 1 like semver.Compare.
func CompareVersions(a, b string) int {
	return semver.Compare(NormalizeVersion(a), NormalizeVersion(b))
}

// IsNewer reports whether latest is a greater semver than current.
func IsNewer(current, latest string) (bool, error) {
	cur := NormalizeVersion(current)
	lat := NormalizeVersion(latest)
	if !semver.IsValid(lat) {
		return false, fmt.Errorf("versão remota inválida: %q", latest)
	}
	if !semver.IsValid(cur) {
		return false, fmt.Errorf("versão local inválida: %q (use --force para atualizar builds de desenvolvimento)", current)
	}
	return semver.Compare(lat, cur) > 0, nil
}

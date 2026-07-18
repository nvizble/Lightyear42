// Package update installs a newer lightyear binary from a release archive.
package update

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/nvizble/Lightyear42/internal/models"
)

// PlatformSuffix returns the GoReleaser archive suffix for goos/goarch
// (e.g. "Linux_x86_64.tar.gz"). Empty when unsupported.
func PlatformSuffix(goos, goarch string) string {
	osPart := map[string]string{
		"linux":   "Linux",
		"darwin":  "Darwin",
		"windows": "Windows",
	}[goos]
	if osPart == "" {
		return ""
	}

	archPart := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}[goarch]
	if archPart == "" {
		return ""
	}

	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return osPart + "_" + archPart + ext
}

// CurrentPlatformSuffix is PlatformSuffix for the running binary.
func CurrentPlatformSuffix() string {
	return PlatformSuffix(runtime.GOOS, runtime.GOARCH)
}

// SelectAsset picks the release archive matching goos/goarch.
// Prefers tar.gz/zip archives; ignores .deb and checksums.
func SelectAsset(assets []models.ReleaseAsset, goos, goarch string) (*models.ReleaseAsset, error) {
	suffix := PlatformSuffix(goos, goarch)
	if suffix == "" {
		return nil, fmt.Errorf("plataforma não suportada para update: %s/%s", goos, goarch)
	}

	for i := range assets {
		name := assets[i].Name
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".deb") || strings.HasSuffix(lower, ".rpm") {
			continue
		}
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		a := assets[i]
		if a.BrowserDownloadURL == "" {
			return nil, fmt.Errorf("asset %s sem URL de download", a.Name)
		}
		return &a, nil
	}
	return nil, fmt.Errorf("nenhum asset para %s no release", suffix)
}

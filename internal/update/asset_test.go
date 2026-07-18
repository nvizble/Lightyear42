package update

import (
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

func TestPlatformSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "Linux_x86_64.tar.gz"},
		{"linux", "arm64", "Linux_arm64.tar.gz"},
		{"darwin", "amd64", "Darwin_x86_64.tar.gz"},
		{"darwin", "arm64", "Darwin_arm64.tar.gz"},
		{"windows", "amd64", "Windows_x86_64.zip"},
		{"windows", "arm64", "Windows_arm64.zip"},
		{"plan9", "amd64", ""},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			t.Parallel()
			if got := PlatformSuffix(tt.goos, tt.goarch); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectAsset(t *testing.T) {
	t.Parallel()

	assets := []models.ReleaseAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://ex/c"},
		{Name: "lightyear_1.0.2_linux_amd64.deb", BrowserDownloadURL: "https://ex/deb"},
		{Name: "lightyear_1.0.2_Linux_x86_64.tar.gz", BrowserDownloadURL: "https://ex/linux"},
		{Name: "lightyear_1.0.2_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://ex/mac"},
		{Name: "lightyear_1.0.2_Windows_x86_64.zip", BrowserDownloadURL: "https://ex/win"},
	}

	tests := []struct {
		goos, goarch, wantURL string
	}{
		{"linux", "amd64", "https://ex/linux"},
		{"darwin", "arm64", "https://ex/mac"},
		{"windows", "amd64", "https://ex/win"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			t.Parallel()
			a, err := SelectAsset(assets, tt.goos, tt.goarch)
			if err != nil {
				t.Fatal(err)
			}
			if a.BrowserDownloadURL != tt.wantURL {
				t.Fatalf("url = %q, want %q", a.BrowserDownloadURL, tt.wantURL)
			}
		})
	}
}

func TestSelectAsset_Missing(t *testing.T) {
	t.Parallel()

	_, err := SelectAsset([]models.ReleaseAsset{
		{Name: "lightyear_1.0.2_linux_amd64.deb", BrowserDownloadURL: "https://ex/deb"},
	}, "linux", "amd64")
	if err == nil {
		t.Fatal("esperava erro sem tar.gz")
	}
}

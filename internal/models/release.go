package models

// Release is a GitHub Release used for self-update.
type Release struct {
	TagName string `json:"tag_name"`
	// Prerelease marks canary builds (tags like v1.3.0-canary.1).
	Prerelease bool           `json:"prerelease"`
	Draft      bool           `json:"draft"`
	Assets     []ReleaseAsset `json:"assets"`
}

// ReleaseAsset is a downloadable file attached to a GitHub Release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

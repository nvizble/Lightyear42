package lsp

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Missing servers are downloaded when that needs no sudo: clangd for Linux
// x86_64 (fresh 42 campus machines don't have it), from the clangd
// project's releases. Only the binary and its builtin headers (stddef.h...)
// are kept, in the user's cache folder (~/.cache/42cli).

// Download is a server's release, fetched when the server is missing.
type Download struct {
	URL, SHA256 string
	// Keep lists the zip entries to extract, by prefix; Binary is the
	// executable among them.
	Keep   []string
	Binary string
	MB     int // the zip's size, to tell the user
}

var clangdLinux = &Download{
	URL:    "https://github.com/clangd/clangd/releases/download/23.1.0/clangd-linux-23.1.0.zip",
	SHA256: "e53b1a96196095faedb7642cf64964f7fb9ad4a0c1f00dd2c172a3d9dcbafdfd",
	Keep:   []string{"clangd_23.1.0/bin/clangd", "clangd_23.1.0/lib/clang/23/include/"},
	Binary: "clangd_23.1.0/bin/clangd",
	MB:     118,
}

// downloadTimeout bounds a download (the clangd zip has 118 MB).
const downloadTimeout = 10 * time.Minute

// fetching keeps two servers of one editor from downloading at once.
var fetching sync.Mutex

// dir is the release's folder in the cache.
func (d *Download) dir() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "42cli", strings.TrimSuffix(path.Base(d.URL), ".zip")), nil
}

// downloaded reports a binary already fetched.
func (d *Download) downloaded() bool {
	dir, err := d.dir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, filepath.FromSlash(d.Binary)))
	return err == nil
}

// fetch returns the binary, downloading and extracting the release first
// when needed (checked against its sha256).
func (d *Download) fetch(ctx context.Context) (string, error) {
	fetching.Lock()
	defer fetching.Unlock()
	dir, err := d.dir()
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, filepath.FromSlash(d.Binary))
	if d.downloaded() {
		return bin, nil
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.URL, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", d.URL, res.Status)
	}
	zipFile, err := os.CreateTemp(parent, "download-*.zip")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(zipFile.Name()) }()
	defer zipFile.Close()
	sum := sha256.New()
	size, err := io.Copy(io.MultiWriter(zipFile, sum), res.Body)
	if err != nil {
		return "", err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != d.SHA256 {
		return "", fmt.Errorf("o arquivo baixado não confere (sha256 %s)", got)
	}

	// Extract next to the final folder, then move it in place at once.
	tmp, err := os.MkdirTemp(parent, "extract-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	zr, err := zip.NewReader(zipFile, size)
	if err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !filepath.IsLocal(f.Name) || !d.keeps(f.Name) {
			continue
		}
		if err := extract(f, filepath.Join(tmp, filepath.FromSlash(f.Name))); err != nil {
			return "", err
		}
	}
	if err := os.Rename(tmp, dir); err != nil && !d.downloaded() {
		return "", err
	}
	return bin, nil
}

func (d *Download) keeps(name string) bool {
	for _, k := range d.Keep {
		if strings.HasPrefix(name, k) {
			return true
		}
	}
	return false
}

func extract(f *zip.File, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

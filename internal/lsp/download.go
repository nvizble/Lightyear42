package lsp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

// Missing servers are downloaded when that needs no sudo, from their
// projects' releases, into the user's cache folder (~/.cache/42cli):
// clangd for Linux x86_64 (fresh 42 campus machines don't have it; only the
// binary and its builtin headers, stddef.h..., are kept) and ty for Python
// (a single binary, no Node or pip needed).

// Download is a server's release, fetched when the server is missing.
type Download struct {
	URL, SHA256 string
	// Dir is its folder in the cache, by release.
	Dir string
	// Keep lists the archive's entries to extract (a zip or a .tar.gz), by
	// prefix; Binary is the executable among them, run with Args.
	Keep   []string
	Binary string
	Args   []string
	MB     int // the archive's size, to tell the user
}

var clangdLinux = &Download{
	URL:    "https://github.com/clangd/clangd/releases/download/23.1.0/clangd-linux-23.1.0.zip",
	SHA256: "e53b1a96196095faedb7642cf64964f7fb9ad4a0c1f00dd2c172a3d9dcbafdfd",
	Dir:    "clangd-linux-23.1.0",
	Keep:   []string{"clangd_23.1.0/bin/clangd", "clangd_23.1.0/lib/clang/23/include/"},
	Binary: "clangd_23.1.0/bin/clangd",
	MB:     118,
}

// tyDownloads are ty's builds (static on Linux), by GOOS/GOARCH.
var tyDownloads = map[string]*Download{
	"linux/amd64":  tyDownload("x86_64-unknown-linux-musl", "52da86894541610d757293c4e9f1c03ec4a1d888ce445e4e373510e4d96595cb", 14),
	"linux/arm64":  tyDownload("aarch64-unknown-linux-musl", "cd8a3f7875ab680e46621a8dc141bfb68823e32bbcee251f644c733211efbe61", 13),
	"darwin/amd64": tyDownload("x86_64-apple-darwin", "79239298f523148d9566ea8c03ade8c96c7f6df2303faf96260f19304425c67f", 13),
	"darwin/arm64": tyDownload("aarch64-apple-darwin", "81c1c7c450cf01ce5dff83f8e62f3f2d0fd1e67462dc77c2bed84fddb31b3c8c", 13),
}

func tyDownload(target, sha256 string, mb int) *Download {
	return &Download{
		URL:    "https://github.com/astral-sh/ty/releases/download/0.0.85/ty-" + target + ".tar.gz",
		SHA256: sha256, Dir: "ty-0.0.85", MB: mb,
		Keep: []string{"ty-" + target + "/ty"}, Binary: "ty-" + target + "/ty", Args: []string{"server"},
	}
}

// downloadTimeout bounds a download (clangd's zip has 118 MB).
const downloadTimeout = 10 * time.Minute

// fetching keeps two servers of one editor from downloading at once.
var fetching sync.Mutex

// dir is the release's folder in the cache.
func (d *Download) dir() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "42cli", d.Dir), nil
}

// name is the program's name ("clangd", "ty").
func (d *Download) name() string { return path.Base(d.Binary) }

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
	archive, err := os.CreateTemp(parent, "download-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(archive.Name()) }()
	defer archive.Close()
	sum := sha256.New()
	size, err := io.Copy(io.MultiWriter(archive, sum), res.Body)
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
	if strings.HasSuffix(d.URL, ".zip") {
		err = d.unzip(archive, size, tmp)
	} else {
		err = d.untar(archive, tmp)
	}
	if err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dir); err != nil && !d.downloaded() {
		return "", err
	}
	return bin, nil
}

// unzip extracts the kept entries of the zip into dir.
func (d *Download) unzip(f io.ReaderAt, size int64, dir string) error {
	zr, err := zip.NewReader(f, size)
	if err != nil {
		return err
	}
	for _, e := range zr.File {
		if e.FileInfo().IsDir() || !d.keeps(e.Name) {
			continue
		}
		r, err := e.Open()
		if err != nil {
			return err
		}
		err = extract(filepath.Join(dir, e.Name), e.Mode(), r)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// untar extracts the kept files of the .tar.gz into dir.
func (d *Download) untar(f io.ReadSeeker, dir string) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeReg && d.keeps(h.Name) {
			if err := extract(filepath.Join(dir, h.Name), h.FileInfo().Mode(), tr); err != nil {
				return err
			}
		}
	}
}

// keeps reports an entry to extract: kept, and inside the folder.
func (d *Download) keeps(name string) bool {
	if !filepath.IsLocal(name) {
		return false
	}
	for _, k := range d.Keep {
		if strings.HasPrefix(name, k) {
			return true
		}
	}
	return false
}

// extract writes one file, keeping its permissions (the binary's x bit).
func extract(dst string, mode os.FileMode, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

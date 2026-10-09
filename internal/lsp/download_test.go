package lsp

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDownload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")

	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, mode := range map[string]os.FileMode{
		"fake_1/bin/clangd":                      0o755,
		"fake_1/lib/clang/1/include/stddef.h":    0o644,
		"fake_1/lib/clang/1/lib/libclang_rt.a":   0o644, // not kept
		"fake_1/bin/../../../../outside/clangd":  0o755, // not local
		"fake_1/LICENSE.TXT":                     0o644, // not kept
		"fake_1/lib/clang/1/include/sys/types.h": 0o644,
	} {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("#!/bin/sh\n# " + name + "\n"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())

	var gets atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gets.Add(1)
		_, _ = w.Write(archive.Bytes())
	}))
	defer srv.Close()

	d := &Download{URL: srv.URL + "/fake-linux-1.zip", SHA256: hex.EncodeToString(sum[:]), Dir: "fake-linux-1",
		Keep: []string{"fake_1/bin/clangd", "fake_1/lib/clang/1/include/"}, Binary: "fake_1/bin/clangd"}
	s := Server{Name: "fake", Commands: [][]string{{"lightyear-no-such-server"}}, Hint: "instale o fake", Download: d}
	if !s.WillDownload() {
		t.Fatal("sem o servidor instalado, ele vai ser baixado")
	}
	argv, err := s.command(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cache, _ := os.UserCacheDir()
	dir := filepath.Join(cache, "42cli", "fake-linux-1")
	if argv[0] != filepath.Join(dir, "fake_1", "bin", "clangd") {
		t.Fatalf("binário: %v", argv)
	}
	if info, err := os.Stat(argv[0]); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("o binário fica executável: %v %v", info, err)
	}
	var kept []string
	_ = filepath.Walk(filepath.Dir(dir), func(p string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			rel, _ := filepath.Rel(filepath.Dir(dir), p)
			kept = append(kept, filepath.ToSlash(rel))
		}
		return nil
	})
	want := "fake-linux-1/fake_1/bin/clangd fake-linux-1/fake_1/lib/clang/1/include/stddef.h fake-linux-1/fake_1/lib/clang/1/include/sys/types.h"
	if strings.Join(kept, " ") != want {
		t.Fatalf("extraído: %v\nqueria só o binário e os headers (sem o zip)", kept)
	}
	if _, err := os.Stat(filepath.Join(home, "outside")); err == nil {
		t.Fatal("nome fora da pasta não pode ser extraído")
	}

	if s.WillDownload() {
		t.Fatal("já baixado, não baixa de novo")
	}
	if _, err := s.command(context.Background()); err != nil || gets.Load() != 1 {
		t.Fatalf("a segunda vez usa o que já baixou: %v, %d downloads", err, gets.Load())
	}

	// A file that doesn't match the sha256 is refused, and nothing stays.
	bad := *d
	bad.URL, bad.Dir = srv.URL+"/fake-linux-2.zip", "fake-linux-2"
	bad.SHA256 = strings.Repeat("0", 64)
	s.Download = &bad
	if _, err := s.command(context.Background()); err == nil || !strings.Contains(err.Error(), "sha256") || !strings.Contains(err.Error(), "instale o fake") {
		t.Fatalf("download adulterado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cache, "42cli", "fake-linux-2")); err == nil {
		t.Fatal("nada do download recusado fica no cache")
	}
	if entries, _ := os.ReadDir(filepath.Join(cache, "42cli")); len(entries) != 1 {
		t.Fatalf("sobrou lixo no cache: %v", entries)
	}
}

func TestDownloadTarGz(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", "")
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	for name, mode := range map[string]int64{"tool-x/tool": 0o755, "tool-x/README.md": 0o644, "../tool": 0o755} {
		body := "#!/bin/sh\n# " + name + "\n"
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	sum := sha256.Sum256(archive.Bytes())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive.Bytes()) }))
	defer srv.Close()

	d := &Download{URL: srv.URL + "/tool-x.tar.gz", SHA256: hex.EncodeToString(sum[:]), Dir: "tool-1",
		Keep: []string{"tool-x/tool"}, Binary: "tool-x/tool", Args: []string{"server"}}
	s := Server{Name: "pyright", Commands: [][]string{{"lightyear-no-such-server"}}, Download: d}
	if s.Program() != "tool" {
		t.Fatalf("o programa é o que vai ser baixado: %s", s.Program())
	}
	argv, err := s.command(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cache, _ := os.UserCacheDir()
	if want := filepath.Join(cache, "42cli", "tool-1", "tool-x", "tool"); len(argv) != 2 || argv[0] != want || argv[1] != "server" {
		t.Fatalf("comando: %v", argv)
	}
	if entries, _ := os.ReadDir(filepath.Join(cache, "42cli", "tool-1", "tool-x")); len(entries) != 1 {
		t.Fatalf("só o binário: %v", entries)
	}
	if info, err := os.Stat(argv[0]); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("o binário fica executável: %v %v", info, err)
	}
}

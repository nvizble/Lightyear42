package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	binaryName      = "lightyear"
	downloadTimeout = 5 * time.Minute
	maxArchiveBytes = 200 << 20 // 200 MiB safety cap
)

// Installer downloads a release archive and replaces the running binary.
type Installer struct {
	HTTPClient *http.Client
	// Executable resolves the path of the binary to replace.
	// Defaults to os.Executable + EvalSymlinks.
	Executable func() (string, error)
}

func (i *Installer) http() *http.Client {
	if i.HTTPClient != nil {
		return i.HTTPClient
	}
	return &http.Client{Timeout: downloadTimeout}
}

func (i *Installer) executable() (string, error) {
	if i.Executable != nil {
		return i.Executable()
	}
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizar binário atual: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil //nolint:nilerr // fall back to unresolved path
	}
	return resolved, nil
}

// TargetPath returns the absolute path of the binary that would be replaced.
func (i *Installer) TargetPath() (string, error) {
	return i.executable()
}

// Install downloads archiveURL, extracts the lightyear binary and replaces target.
func (i *Installer) Install(ctx context.Context, archiveURL, archiveName string) error {
	target, err := i.executable()
	if err != nil {
		return err
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat do binário atual: %w", err)
	}
	dir := filepath.Dir(target)

	tmpArchive, err := os.CreateTemp(dir, "lightyear-update-*.archive")
	if err != nil {
		// Directory not writable — try system temp, then copy into place later.
		tmpArchive, err = os.CreateTemp("", "lightyear-update-*.archive")
		if err != nil {
			return fmt.Errorf("criar arquivo temporário: %w", err)
		}
	}
	archivePath := tmpArchive.Name()
	defer os.Remove(archivePath)

	if err := i.download(ctx, archiveURL, tmpArchive); err != nil {
		tmpArchive.Close()
		return err
	}
	if err := tmpArchive.Close(); err != nil {
		return err
	}

	binPath, err := extractBinary(archivePath, archiveName, dir)
	if err != nil {
		return err
	}
	defer os.Remove(binPath)

	if err := os.Chmod(binPath, info.Mode().Perm()); err != nil {
		return fmt.Errorf("ajustar permissões: %w", err)
	}

	return replaceExecutable(binPath, target)
}

func (i *Installer) download(ctx context.Context, url string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "lightyear-cli")

	resp, err := i.http().Do(req)
	if err != nil {
		return fmt.Errorf("baixar release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("baixar release: HTTP %d", resp.StatusCode)
	}

	n, err := io.Copy(w, io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("gravar download: %w", err)
	}
	if n > maxArchiveBytes {
		return fmt.Errorf("arquivo de release maior que o limite (%d bytes)", maxArchiveBytes)
	}
	return nil
}

func extractBinary(archivePath, archiveName, destDir string) (string, error) {
	want := binaryName
	if runtime.GOOS == "windows" || strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		if runtime.GOOS == "windows" {
			want = binaryName + ".exe"
		}
	}

	out, err := os.CreateTemp(destDir, "lightyear-bin-*")
	if err != nil {
		out, err = os.CreateTemp("", "lightyear-bin-*")
		if err != nil {
			return "", fmt.Errorf("criar temp do binário: %w", err)
		}
	}
	outPath := out.Name()
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(outPath)
		}
	}()

	switch {
	case strings.HasSuffix(strings.ToLower(archiveName), ".zip"):
		if err := extractFromZip(archivePath, want, out); err != nil {
			return "", err
		}
	default:
		if err := extractFromTarGz(archivePath, want, out); err != nil {
			return "", err
		}
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	ok = true
	return outPath, nil
}

func extractFromTarGz(path, want string, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		name := filepath.Base(hdr.Name)
		if name != want && name != binaryName {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if _, err := io.Copy(w, io.LimitReader(tr, maxArchiveBytes)); err != nil {
			return fmt.Errorf("extrair %s: %w", name, err)
		}
		return nil
	}
	return fmt.Errorf("binário %q não encontrado no archive", want)
}

func extractFromZip(path, want string, w io.Writer) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("zip: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != want && name != binaryName && name != binaryName+".exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, io.LimitReader(rc, maxArchiveBytes))
		_ = rc.Close()
		if copyErr != nil {
			return fmt.Errorf("extrair %s: %w", name, copyErr)
		}
		return nil
	}
	return fmt.Errorf("binário %q não encontrado no zip", want)
}

// replaceExecutable moves newPath over targetPath.
// On Windows, renames the running binary aside first.
func replaceExecutable(newPath, targetPath string) error {
	if runtime.GOOS == "windows" {
		return replaceWindows(newPath, targetPath)
	}
	if err := os.Rename(newPath, targetPath); err == nil {
		return nil
	}
	// Cross-device rename fails — copy into place.
	return copyOver(newPath, targetPath)
}

func copyOver(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := os.Stat(dst)
	perm := os.FileMode(0o755)
	if err == nil {
		perm = info.Mode().Perm()
	}

	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("sem permissão para escrever em %s: %w\nInstale em ~/.local/bin ou rode com permissão adequada", filepath.Dir(dst), err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("substituir binário: %w", err)
	}
	_ = os.Remove(src)
	return nil
}

func replaceWindows(newPath, targetPath string) error {
	backup := targetPath + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(targetPath, backup); err != nil {
		return fmt.Errorf("renomear binário atual (Windows): %w — feche outras instâncias do lightyear e tente de novo", err)
	}
	if err := os.Rename(newPath, targetPath); err != nil {
		_ = os.Rename(backup, targetPath)
		return fmt.Errorf("instalar novo binário: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

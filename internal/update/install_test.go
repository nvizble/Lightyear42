package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstaller_Install_TarGz(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "lightyear")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	archive := buildTarGz(t, "lightyear", []byte("#!/new-binary\n"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archive)
	}))
	t.Cleanup(srv.Close)

	inst := &Installer{
		HTTPClient: srv.Client(),
		Executable: func() (string, error) { return target, nil },
	}
	if err := inst.Install(context.Background(), srv.URL+"/lightyear_1.0.3_Linux_x86_64.tar.gz", "lightyear_1.0.3_Linux_x86_64.tar.gz"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "#!/new-binary\n" {
		t.Fatalf("conteúdo = %q", got)
	}
}

func buildTarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	pr, pw := io.Pipe()
	go func() {
		gw := gzip.NewWriter(pw)
		tw := tar.NewWriter(gw)
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(body)
		_ = tw.Close()
		_ = gw.Close()
		_ = pw.Close()
	}()
	data, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

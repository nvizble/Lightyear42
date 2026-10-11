package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// fakeGitHub serves /releases/latest (a redirect to the stable tag) and
// /releases.atom (the feed), counting the requests.
func fakeGitHub(t *testing.T, stable string, feed []string) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/o/r/releases/latest":
			w.Header().Set("Location", "https://github.com/o/r/releases/tag/"+stable)
			w.WriteHeader(http.StatusFound)
		case "/o/r/releases.atom":
			body := "<feed>"
			for _, tag := range feed {
				body += `<entry><link href="https://github.com/o/r/releases/tag/` + tag + `"/></entry>`
			}
			_, _ = w.Write([]byte(body + "</feed>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func notifier(t *testing.T, srv *httptest.Server, now *time.Time) *Notifier {
	n := NewNotifier(filepath.Join(t.TempDir(), "update-check.json"), "o", "r")
	n.BaseURL = srv.URL + "/o/r"
	n.Now = func() time.Time { return *now }
	return n
}

func TestNotifierStable(t *testing.T) {
	srv, hits := fakeGitHub(t, "v1.3.0", nil)
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	n := notifier(t, srv, &now)

	if !n.Stale("1.2.1") || n.Newer("1.2.1") != "" {
		t.Fatal("sem estado, deveria estar velho e sem aviso")
	}
	if err := n.Refresh(context.Background(), "1.2.1"); err != nil {
		t.Fatal(err)
	}
	if got := n.Newer("1.2.1"); got != "1.3.0" {
		t.Fatalf("Newer = %q, esperado 1.3.0", got)
	}
	if n.Newer("1.3.0") != "" || n.Newer("1.4.0") != "" {
		t.Error("quem está na última (ou adiante) não deveria ver aviso")
	}
	if n.Newer("dev") != "" {
		t.Error("builds de desenvolvimento não têm aviso")
	}
	if n.Stale("1.2.1") {
		t.Error("acabou de checar: não deveria estar velho")
	}
	now = now.Add(NoticeInterval)
	if !n.Stale("1.2.1") {
		t.Error("depois de 24h deveria checar de novo")
	}
	if *hits != 1 {
		t.Errorf("%d pedidos, esperado 1", *hits)
	}
	// The stable answer doesn't count for a canary build.
	if !n.Stale("1.3.0-canary.5") || n.Newer("1.3.0-canary.5") != "" {
		t.Error("o estado do canal estável não vale para a canary")
	}
}

func TestNotifierCanary(t *testing.T) {
	srv, _ := fakeGitHub(t, "v1.2.1", []string{"v1.3.0-canary.9", "v1.3.0-canary.10", "v1.2.1", "v1.3.0-canary.2"})
	now := time.Now()
	n := notifier(t, srv, &now)
	if err := n.Refresh(context.Background(), "1.3.0-canary.9"); err != nil {
		t.Fatal(err)
	}
	if got := n.Newer("1.3.0-canary.9"); got != "1.3.0-canary.10" {
		t.Fatalf("Newer = %q, esperado 1.3.0-canary.10 (a maior, não a primeira do feed)", got)
	}
	if n.Newer("1.3.0-canary.10") != "" {
		t.Error("na última canary não há aviso")
	}
}

func TestNotifierOffline(t *testing.T) {
	srv, _ := fakeGitHub(t, "v1.3.0", nil)
	srv.Close() // nothing answers
	now := time.Now()
	n := notifier(t, srv, &now)
	if err := n.Refresh(context.Background(), "1.2.1"); err == nil {
		t.Fatal("sem rede, Refresh deveria falhar")
	}
	if n.Stale("1.2.1") {
		t.Error("a falha também conta como checagem: senão cada comando tentaria de novo")
	}
	if n.Newer("1.2.1") != "" {
		t.Error("sem versão conhecida, sem aviso")
	}
}

func TestNewestTag(t *testing.T) {
	if _, err := newestTag("<feed></feed>"); err == nil {
		t.Error("feed vazio deveria dar erro")
	}
	got, err := newestTag(`href="/o/r/releases/tag/v1.0.0" href="/o/r/releases/tag/v2.0.0-canary.1" href="/o/r/releases/tag/not-a-version"`)
	if err != nil || got != "v2.0.0-canary.1" {
		t.Errorf("newestTag = %q, %v", got, err)
	}
}

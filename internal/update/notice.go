package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The new-version notice: once a day at most, the newest release of the
// installed build's channel is looked up and kept in a small file; every
// command reads that file and, when the build is behind, says so.
//
// It doesn't use the GitHub API (60 requests an hour per IP, which a whole
// campus behind one address runs out of): the stable comes from the
// /releases/latest redirect and the canary from the releases feed, the same
// way install.sh finds them.

// NoticeInterval is how long a lookup is trusted.
const NoticeInterval = 24 * time.Hour

// NoticeState is what the notice file keeps.
type NoticeState struct {
	CheckedAt time.Time `json:"checked_at"`
	Channel   string    `json:"channel"`
	Latest    string    `json:"latest"`
}

// Notifier looks up the newest release and remembers it in Path.
type Notifier struct {
	// Path is the state file (in the cache folder).
	Path string
	// BaseURL is the repository's page ("https://github.com/owner/repo").
	BaseURL string
	// Client does the requests; it must not follow redirects (see NewNotifier).
	Client *http.Client
	// Now is the clock (tests).
	Now func() time.Time
}

// NewNotifier returns a notifier for github.com/owner/repo keeping its state in path.
func NewNotifier(path, owner, repo string) *Notifier {
	return &Notifier{
		Path:    path,
		BaseURL: "https://github.com/" + owner + "/" + repo,
		Client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse // the redirect itself carries the tag
			},
		},
		Now: time.Now,
	}
}

// Channel is the channel a build follows: canary for a pre-release.
func Channel(current string) string {
	if IsPrerelease(current) {
		return "canary"
	}
	return "stable"
}

// Newer returns the newer release the state knows of for current, or "".
// It only answers for release builds of the same channel.
func (n *Notifier) Newer(current string) string {
	if !IsReleaseVersion(current) {
		return ""
	}
	st, err := n.load()
	if err != nil || st.Channel != Channel(current) || !IsReleaseVersion(st.Latest) {
		return ""
	}
	if newer, err := IsNewer(current, st.Latest); err != nil || !newer {
		return ""
	}
	return strings.TrimPrefix(st.Latest, "v")
}

// Stale reports whether the state is older than NoticeInterval (or about
// another channel, or missing).
func (n *Notifier) Stale(current string) bool {
	st, err := n.load()
	return err != nil || st.Channel != Channel(current) || n.Now().Sub(st.CheckedAt) >= NoticeInterval
}

// Touch marks the state as just checked, keeping the version it knew, so
// commands run while a lookup is under way don't start another one.
func (n *Notifier) Touch(current string) error {
	st, err := n.load()
	if err != nil || st.Channel != Channel(current) {
		st = NoticeState{Channel: Channel(current)}
	}
	st.CheckedAt = n.Now()
	return n.save(st)
}

// Refresh looks the newest release up and saves it. A failed lookup is
// remembered too (with no version), so an offline machine doesn't retry on
// every command.
func (n *Notifier) Refresh(ctx context.Context, current string) error {
	channel := Channel(current)
	latest, err := n.latest(ctx, channel)
	st := NoticeState{CheckedAt: n.Now(), Channel: channel, Latest: latest}
	if saveErr := n.save(st); saveErr != nil && err == nil {
		err = saveErr
	}
	return err
}

func (n *Notifier) latest(ctx context.Context, channel string) (string, error) {
	if channel == "canary" {
		body, err := n.get(ctx, n.BaseURL+"/releases.atom")
		if err != nil {
			return "", err
		}
		return newestTag(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.BaseURL+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	resp, err := n.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if m := tagInURL.FindStringSubmatch(resp.Header.Get("Location")); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("a última release não veio no redirect (status %d)", resp.StatusCode)
}

func (n *Notifier) get(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := n.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(body), err
}

var (
	tagInURL  = regexp.MustCompile(`/releases/tag/(v[^/?#\s]+)$`)
	tagInFeed = regexp.MustCompile(`/releases/tag/(v[^"<\s]+)`)
)

// newestTag is the highest release version named in the feed.
func newestTag(feed string) (string, error) {
	newest := ""
	for _, m := range tagInFeed.FindAllStringSubmatch(feed, -1) {
		if IsReleaseVersion(m[1]) && (newest == "" || CompareVersions(m[1], newest) > 0) {
			newest = m[1]
		}
	}
	if newest == "" {
		return "", errors.New("nenhuma release no feed")
	}
	return newest, nil
}

func (n *Notifier) load() (NoticeState, error) {
	var st NoticeState
	data, err := os.ReadFile(n.Path)
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(data, &st)
	return st, err
}

func (n *Notifier) save(st NoticeState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(n.Path), 0o755); err != nil {
		return err
	}
	tmp := n.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, n.Path)
}

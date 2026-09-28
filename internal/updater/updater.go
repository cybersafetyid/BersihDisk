// Package updater checks GitHub Releases, downloads the newest build with
// progress events, and hands the finished file to the OS to install.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// apiBase is overridable in tests; GitHub's API needs no authentication to read
// the latest release of a public repository.
var apiBase = "https://api.github.com"

// downloadHosts limits where an update may be fetched from. The asset URL comes
// from a remote API response, so it is validated instead of being followed
// blindly — this keeps a tampered or redirected feed from making the app fetch
// and run arbitrary URLs.
var downloadHosts = []string{"github.com", "githubusercontent.com"}

// maxDownload caps an update download so a hostile feed cannot fill the disk.
const maxDownload = 1 << 30

// Info tells the UI whether a newer release exists and what to download.
type Info struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`
	Asset     string `json:"asset"`
	URL       string `json:"url"`
	Bytes     int64  `json:"bytes"`
	Digest    string `json:"digest"` // "sha256:<hex>" when the release publishes one
	Page      string `json:"page"`   // release page in the browser
}

// Progress is one download update.
type Progress struct {
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	Total   int64  `json:"total"`
	Speed   int64  `json:"speed"` // bytes per second, averaged over the last interval
	Percent int    `json:"percent"`
	Phase   string `json:"phase"` // "download" | "verify" | "ready"
}

type release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	HTMLURL string  `json:"html_url"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// Check fetches the latest release of repo (owner/name) and compares it with the
// running version. A version that cannot be parsed, such as the "dev" default of
// a build without linker flags, is reported as an error rather than guessed.
func Check(ctx context.Context, repo, current string) (Info, error) {
	if repo == "" {
		return Info{}, fmt.Errorf("no release repository configured")
	}
	info := Info{Current: current}
	cur, err := parseVersion(current)
	if err != nil {
		return Info{}, fmt.Errorf("running version %q is not a release version", current)
	}

	httpCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, strings.Trim(repo, "/")), nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "bersihdisk-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Info{}, fmt.Errorf("check for updates failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("release feed returned HTTP %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return Info{}, fmt.Errorf("release feed could not be read: %w", err)
	}

	latest, err := parseVersion(rel.TagName)
	if err != nil {
		return Info{}, fmt.Errorf("latest release %q is not versioned", rel.TagName)
	}
	info.Latest = rel.TagName
	info.Notes = strings.TrimSpace(rel.Body)
	info.Page = rel.HTMLURL

	asset := pickAsset(runtime.GOOS, runtime.GOARCH, rel.Assets)
	if asset == nil {
		return info, nil // a newer release exists but has no build for this OS
	}
	info.Asset = asset.Name
	info.URL = asset.BrowserDownloadURL
	info.Bytes = asset.Size
	info.Digest = asset.Digest
	info.Available = latest.compare(cur) > 0
	return info, nil
}

// pickAsset chooses the build that matches goos/goarch, preferring the most
// installable format per platform. The release artifact names are produced by
// scripts/package.sh; TestPickAssetMatchesPackagedNames keeps the two in step.
func pickAsset(goos, goarch string, assets []asset) *asset {
	var wantOS, wantExt []string
	switch goos {
	case "darwin":
		wantOS = []string{"macos", "darwin", "apple"}
		wantExt = []string{".dmg", ".pkg", ".zip"}
	case "windows":
		wantOS = []string{"windows", "win64", "win"}
		wantExt = []string{".exe", ".msi", ".zip"}
	default:
		wantOS = []string{"linux"}
		wantExt = []string{".AppImage", ".deb", ".tar.gz", ".tgz", ".zip"}
	}
	arch := goarch
	if arch == "amd64" {
		arch = "x86_64"
	}

	for _, ext := range wantExt {
		for i := range assets {
			a := strings.ToLower(assets[i].Name)
			if !hasAny(a, wantOS) || !strings.HasSuffix(a, strings.ToLower(ext)) {
				continue
			}
			// When the release ships per-architecture builds, do not hand over a
			// build for another one.
			if arch != "" && (strings.Contains(a, "arm64") || strings.Contains(a, "x86_64")) &&
				!strings.Contains(a, strings.ToLower(arch)) {
				continue
			}
			return &assets[i]
		}
	}
	return nil
}

func hasAny(s string, options []string) bool {
	for _, o := range options {
		if strings.Contains(s, o) {
			return true
		}
	}
	return false
}

// Download saves url into a fresh temporary directory and reports progress. The
// file is verified against digest ("sha256:<hex>") when one is known, and a
// partial or mismatched download is deleted rather than offered for install.
func Download(ctx context.Context, url, digest string, onProgress func(Progress)) (string, error) {
	name, err := checkDownloadURL(url)
	if err != nil {
		return "", err
	}
	// An installer is run afterwards, so an unverifiable one is never fetched.
	want, err := hexDigest(digest)
	if err != nil {
		return "", fmt.Errorf("release has no sha256 digest to verify the download: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "bersihdisk-updater")

	// Every redirect hop must stay on a release host, not only the first URL.
	client := *http.DefaultClient
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		_, err := checkDownloadURL(r.URL.String())
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxDownload {
		return "", fmt.Errorf("download is larger than %d bytes", maxDownload)
	}

	dir, err := os.MkdirTemp("", "bersihdisk-update-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	if total < 0 {
		total = 0
	}
	hash := sha256.New()
	var written int64
	var lastEmit time.Time
	var lastBytes int64
	var lastAt = time.Now()

	buf := make([]byte, 256*1024)
	emit := func(force bool) {
		if onProgress == nil {
			return
		}
		now := time.Now()
		if !force && now.Sub(lastEmit) < 200*time.Millisecond {
			return
		}
		interval := now.Sub(lastAt).Seconds()
		var speed int64
		if interval > 0 {
			speed = int64(float64(written-lastBytes) / interval)
		}
		lastEmit, lastBytes, lastAt = now, written, now
		pct := 0
		if total > 0 {
			pct = int(written * 100 / total)
		}
		onProgress(Progress{Path: path, Bytes: written, Total: total, Speed: speed, Percent: pct, Phase: "download"})
	}

	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.RemoveAll(dir)
				return "", fmt.Errorf("could not write the download: %w", werr)
			}
			_, _ = hash.Write(buf[:n])
			written += int64(n)
			if written > maxDownload {
				f.Close()
				os.RemoveAll(dir)
				return "", fmt.Errorf("download is larger than %d bytes", maxDownload)
			}
			emit(false)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.RemoveAll(dir)
			if ctx.Err() != nil {
				return "", fmt.Errorf("download cancelled")
			}
			return "", fmt.Errorf("download stopped: %w", rerr)
		}
	}

	if err := f.Close(); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	emit(true)

	onProgressMaybe(onProgress, Progress{Path: path, Bytes: written, Total: written, Percent: 100, Phase: "verify"})
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		os.RemoveAll(dir)
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", want, got)
	}

	onProgressMaybe(onProgress, Progress{Path: path, Bytes: written, Total: written, Percent: 100, Phase: "ready"})
	return path, nil
}

func onProgressMaybe(onProgress func(Progress), p Progress) {
	if onProgress != nil {
		onProgress(p)
	}
}

// checkDownloadURL accepts only an https asset from one of the release hosts, and
// returns the file name to store it under.
func checkDownloadURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("download URL is not valid")
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("download URL must be https")
	}
	host := strings.ToLower(u.Hostname())
	allowed := false
	for _, suffix := range downloadHosts {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("download host %q is not a release host", host)
	}
	name := filepath.Base(u.Path)
	if name == "." || name == "/" || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("download URL has no file name")
	}
	return name, nil
}

// hexDigest accepts "sha256:<hex>" or a bare hex digest.
func hexDigest(digest string) (string, error) {
	hexPart := strings.ToLower(strings.TrimSpace(digest))
	if i := strings.Index(hexPart, ":"); i >= 0 {
		hexPart = hexPart[i+1:]
	}
	if len(hexPart) != 64 {
		return "", fmt.Errorf("release digest %q is not sha256", digest)
	}
	return hexPart, nil
}

var versionRE = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.]+)?`)

// version is major.minor.patch; pre marks a "-beta"-style suffix, which sorts
// before the release with the same numbers.
type version struct {
	n   [3]int
	pre bool
}

func parseVersion(s string) (version, error) {
	m := versionRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return version{}, fmt.Errorf("no version in %q", s)
	}
	var v version
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return version{}, err
		}
		v.n[i] = n
	}
	v.pre = m[4] != ""
	return v, nil
}

// compare returns 1 when v is newer than other, -1 when older, 0 when equal.
func (v version) compare(other version) int {
	for i := 0; i < 3; i++ {
		if v.n[i] != other.n[i] {
			if v.n[i] > other.n[i] {
				return 1
			}
			return -1
		}
	}
	switch {
	case v.pre && !other.pre:
		return -1
	case !v.pre && other.pre:
		return 1
	}
	return 0
}

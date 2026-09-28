package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
)

// useTestServer points the package at an httptest server: the same https-only
// rules apply, so the checks under test are the production ones.
func useTestServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prevAPI, prevClient, prevHosts := apiBase, http.DefaultClient, downloadHosts
	t.Cleanup(func() {
		apiBase, http.DefaultClient, downloadHosts = prevAPI, prevClient, prevHosts
	})
	apiBase = srv.URL
	http.DefaultClient = srv.Client()
	downloadHosts = append(downloadHosts, "127.0.0.1")
}

// assetName builds a release asset name that matches the running OS and arch.
func assetName(version string) string {
	osName := map[string]string{"darwin": "macos", "windows": "windows"}[runtime.GOOS]
	if osName == "" {
		osName = "linux"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	return fmt.Sprintf("bersihdisk-%s-%s-%s.zip", version, osName, arch)
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "1.2.3", 0},
		{"1.10.0", "1.9.9", 1},
		{"v0.9.0", "1.0.0", -1},
		{"1.2.3-beta1", "1.2.2", 1},
		{"1.2.3-beta1", "1.2.3", -1},
		{"1.2.3", "1.2.3-rc1", 1},
	}
	for _, c := range cases {
		a, err := parseVersion(c.a)
		if err != nil {
			t.Fatalf("parse %q: %v", c.a, err)
		}
		b, err := parseVersion(c.b)
		if err != nil {
			t.Fatalf("parse %q: %v", c.b, err)
		}
		if got := a.compare(b); got != c.want {
			t.Errorf("compare(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
	if _, err := parseVersion("dev"); err == nil {
		t.Error("parseVersion(\"dev\") must fail instead of pretending to be 0.0.0")
	}
}

// The release artifacts are named by scripts/package.sh. Every platform must get
// its installer, never another OS's file, the portable zip, or the checksum list.
func TestPickAssetMatchesPackagedNames(t *testing.T) {
	names := []string{
		"bersihdisk-v1.0.0-macos-universal.dmg",
		"bersihdisk-v1.0.0-windows-x86_64-setup.exe",
		"bersihdisk-v1.0.0-windows-x86_64-portable.zip",
		"bersihdisk-v1.0.0-linux-x86_64.deb",
		"bersihdisk-v1.0.0-linux-x86_64.tar.gz",
		"bersihdisk-v1.0.0-SHA256SUMS.txt",
	}
	var assets []asset
	for _, n := range names {
		assets = append(assets, asset{Name: n})
	}
	cases := []struct{ goos, goarch, want string }{
		{"darwin", "arm64", "bersihdisk-v1.0.0-macos-universal.dmg"},
		{"darwin", "amd64", "bersihdisk-v1.0.0-macos-universal.dmg"},
		{"windows", "amd64", "bersihdisk-v1.0.0-windows-x86_64-setup.exe"},
		{"linux", "amd64", "bersihdisk-v1.0.0-linux-x86_64.deb"},
		{"windows", "arm64", ""}, // no arm64 build is published
		{"linux", "arm64", ""},
	}
	for _, c := range cases {
		got := ""
		if a := pickAsset(c.goos, c.goarch, assets); a != nil {
			got = a.Name
		}
		if got != c.want {
			t.Errorf("%s/%s picked %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestCheckReportsNewerAsset(t *testing.T) {
	name := assetName("v2.0.0")
	body := fmt.Sprintf(`{"tag_name":"v2.0.0","name":"BersihDisk 2.0.0","body":"notes here","html_url":"https://github.com/o/r/releases/tag/v2.0.0",
		"assets":[{"name":%q,"size":4242,"browser_download_url":"https://127.0.0.1:1/download/v2.0.0/%s","digest":"sha256:%s"}]}`,
		name, name, strings.Repeat("ab", 32))

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, body)
	}))
	defer srv.Close()
	useTestServer(t, srv)

	info, err := Check(context.Background(), "owner/repo", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available {
		t.Errorf("Available = false, want true (info %+v)", info)
	}
	if info.Latest != "v2.0.0" || info.Asset != name || info.Bytes != 4242 {
		t.Errorf("unexpected release info: %+v", info)
	}
	if !strings.HasPrefix(info.Digest, "sha256:") {
		t.Errorf("digest = %q, want the sha256 prefix kept", info.Digest)
	}

	if info, err := Check(context.Background(), "owner/repo", "2.0.0"); err != nil || info.Available {
		t.Errorf("same version must not offer an update: %+v err=%v", info, err)
	}
	if _, err := Check(context.Background(), "owner/repo", "dev"); err == nil {
		t.Error("a dev build must report an error rather than guess its version")
	}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	payload := strings.Repeat("bersihdisk", 5000)
	sum := sha256.Sum256([]byte(payload))

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, payload)
	}))
	defer srv.Close()
	useTestServer(t, srv)

	url := srv.URL + "/download/v2.0.0/" + assetName("v2.0.0")

	var phases []string
	path, err := Download(context.Background(), url, "sha256:"+hex.EncodeToString(sum[:]),
		func(p Progress) { phases = append(phases, p.Phase) })
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != payload {
		t.Errorf("downloaded %d bytes, want %d", len(data), len(payload))
	}
	for _, want := range []string{"download", "verify", "ready"} {
		if !strings.Contains(strings.Join(phases, ","), want) {
			t.Errorf("phases %v are missing %q", phases, want)
		}
	}
}

func TestDownloadRejectsBadInput(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	}))
	defer srv.Close()
	useTestServer(t, srv)

	if _, err := Download(context.Background(), "http://127.0.0.1/x.zip", "", nil); err == nil {
		t.Error("plain http must be refused")
	}
	if _, err := Download(context.Background(), "https://evil.example/x.zip", "", nil); err == nil {
		t.Error("a host outside the release allowlist must be refused")
	}

	if _, err := Download(context.Background(), srv.URL+"/x.zip", "", nil); err == nil {
		t.Error("a release without a digest must be refused")
	}

	path, err := Download(context.Background(), srv.URL+"/x.zip", "sha256:"+strings.Repeat("00", 32), nil)
	if err == nil {
		t.Fatal("a mismatched checksum must be an error")
	}
	if path != "" {
		if _, serr := os.Stat(path); serr == nil {
			t.Errorf("the unverified file was left behind at %s", path)
		}
	}
}

func TestHexDigestFormats(t *testing.T) {
	good := strings.Repeat("cd", 32)
	for _, in := range []string{good, "sha256:" + good, "SHA256:" + strings.ToUpper(good)} {
		if got, err := hexDigest(in); err != nil || got != good {
			t.Errorf("hexDigest(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := hexDigest("md5:deadbeef"); err == nil {
		t.Error("a non-sha256 digest must be refused")
	}
}

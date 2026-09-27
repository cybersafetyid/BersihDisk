package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSizesDirsAndFlagsLinks(t *testing.T) {
	root := t.TempDir()
	big := filepath.Join(root, "big")
	if err := os.MkdirAll(filepath.Join(big, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path string, n int) {
		t.Helper()
		if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(big, "sub", "a.bin"), 4096)
	write(filepath.Join(root, "small.txt"), 2)
	if err := os.Symlink(filepath.Join(root, "small.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	entries, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d (%v), want 3", len(entries), entries)
	}

	if got := byName["big"]; !got.IsDir || got.Size != 4096 {
		t.Errorf("big = %+v, want a directory holding 4096 bytes", got)
	}
	if got := byName["small.txt"]; got.IsDir || got.Size != 2 {
		t.Errorf("small.txt = %+v, want a 2 byte file", got)
	}
	if got := byName["link.txt"]; !got.IsLink {
		t.Errorf("link.txt = %+v, want IsLink so the UI can warn it frees nothing", got)
	}
	if entries[0].Name != "big" {
		t.Errorf("largest entry first = %q, want \"big\"", entries[0].Name)
	}
}

func TestListRejectsMissingFolder(t *testing.T) {
	if _, err := List(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Error("listing a folder that does not exist must return an error")
	}
}

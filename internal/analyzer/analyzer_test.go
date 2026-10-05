// Tests for the drive-usage analyzer: sizes, ordering, skip rules and grading.
package analyzer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// mk creates a file of the given byte size inside dir.
func mk(t *testing.T, dir, name string, size int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Repeat("x", size)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListSizesAndOrder(t *testing.T) {
	root := t.TempDir()
	mk(t, filepath.Join(root, "big"), "data.bin", 500)
	mk(t, filepath.Join(root, "small"), "data.bin", 10)
	mk(t, root, "top.bin", 100)

	res, err := New(func(Progress) {}).List(root)
	if err != nil {
		t.Fatal(err)
	}

	if res.Files != 1 || res.Dirs != 2 {
		t.Errorf("files=%d dirs=%d, want 1/2", res.Files, res.Dirs)
	}
	if res.TotalBytes != 610 {
		t.Errorf("total=%d, want 610", res.TotalBytes)
	}
	want := []string{"big", "top.bin", "small"}
	for i, name := range want {
		if res.Entries[i].Name != name {
			t.Fatalf("entry %d = %q, want %q (all: %+v)", i, res.Entries[i].Name, name, res.Entries)
		}
	}
	if res.Entries[0].Size != 500 || res.Entries[2].Size != 10 {
		t.Errorf("directory sizes = %d/%d, want 500/10", res.Entries[0].Size, res.Entries[2].Size)
	}
	if res.Entries[0].Level != "safe" {
		t.Errorf("plain folder level = %q, want safe", res.Entries[0].Level)
	}
}

// A symlink must be listed but not followed: deleting it frees nothing, and
// walking through it could leave the drive or loop forever.
func TestListDoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}
	root := t.TempDir()
	mk(t, filepath.Join(root, "real"), "data.bin", 4000)
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	res, err := New(func(Progress) {}).List(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Entries {
		if e.Name != "link" {
			continue
		}
		if !e.IsLink || e.IsDir {
			t.Errorf("link = %+v, want isLink and not a directory", e)
		}
		if e.Size != 0 {
			t.Errorf("link size = %d, want 0 (not followed)", e.Size)
		}
		return
	}
	t.Fatal("link was not listed")
}

// Virtual/pseudo filesystems and the OS trash are skipped, so a whole-drive
// analysis stays stable; an ordinary folder with the same name is not.
func TestSkipDir(t *testing.T) {
	sep := string(filepath.Separator)
	root := sep
	if v := filepath.VolumeName(os.TempDir()); v != "" {
		root = v + sep
	}
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(root, "proc"), true},
		{filepath.Join(root, "home", "me", "proc"), false},
		{filepath.Join(root, "work", ".Trash"), true},
		{filepath.Join(root, "work", "System Volume Information"), true},
		{filepath.Join(root, "work", "src"), false},
	}
	for _, c := range cases {
		if got := skipDir(c.path); got != c.want {
			t.Errorf("skipDir(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestListMarksSkippedAndGradesBundles(t *testing.T) {
	root := t.TempDir()
	mk(t, filepath.Join(root, ".Trash", "old"), "junk.bin", 900)
	mk(t, filepath.Join(root, "Tool.app", "Contents", "Resources"), "asset.bin", 300)

	res, err := New(func(Progress) {}).List(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", res.Skipped)
	}
	byName := map[string]Entry{}
	for _, e := range res.Entries {
		byName[e.Name] = e
	}
	trash := byName[".Trash"]
	if !trash.Skipped || trash.Size != 0 {
		t.Errorf(".Trash = %+v, want skipped with size 0", trash)
	}
	bundle := byName["Tool.app"]
	if bundle.Level != "danger" || len(bundle.Reasons) == 0 || bundle.Reasons[0] != "appBundle" {
		t.Errorf("Tool.app = %+v, want danger/appBundle", bundle)
	}
	if res.TotalBytes != 300 {
		t.Errorf("total = %d, want 300 (the trash is not measured)", res.TotalBytes)
	}
}

func TestListCancelMarksPartial(t *testing.T) {
	root := t.TempDir()
	mk(t, filepath.Join(root, "d"), "x.bin", 10)

	a := New(func(Progress) {})
	a.Cancel()
	res, err := a.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Partial {
		t.Error("a cancelled listing must be marked partial")
	}
}

func TestListMissingDirectoryErrors(t *testing.T) {
	if _, err := New(func(Progress) {}).List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("listing a missing directory must return an error")
	}
}

// Deleter tests: permanent mode, dedup, cancel, and trash when available.
package deleter

import (
	"os"
	"path/filepath"
	"testing"
)

func mkDir(t *testing.T, parent, name string) string {
	t.Helper()
	p := filepath.Join(parent, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "data.bin"), make([]byte, 128), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDeletePermanent(t *testing.T) {
	tmp := t.TempDir()
	a := mkDir(t, tmp, "a")
	b := mkDir(t, tmp, "b")

	d := New(func(Progress) {})
	res := d.Delete([]Item{
		{Path: a, Size: 128},
		{Path: b, Size: 128},
	}, ModePermanent)

	if res.OK != 2 || res.Failed != 0 {
		t.Fatalf("want 2 OK 0 failed, got %+v", res)
	}
	if res.Bytes != 256 {
		t.Errorf("bytes = %d, want 256", res.Bytes)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Error("a should be deleted")
	}
}

func TestDeleteDedup(t *testing.T) {
	tmp := t.TempDir()
	a := mkDir(t, tmp, "a")

	d := New(func(Progress) {})
	res := d.Delete([]Item{
		{Path: a, Size: 10},
		{Path: a, Size: 10},
	}, ModePermanent)

	if res.OK != 1 {
		t.Errorf("dedup: want 1 OK, got %d", res.OK)
	}
}

// A path inside another selected path is removed by the parent, so it must not
// be deleted separately or have its bytes counted twice.
func TestDeleteSkipsNestedAndMissing(t *testing.T) {
	tmp := t.TempDir()
	parent := mkDir(t, tmp, "parent")
	child := mkDir(t, parent, "child")

	d := New(func(Progress) {})
	res := d.Delete([]Item{
		{Path: parent, Size: 300},
		{Path: child, Size: 128},
	}, ModePermanent)
	if res.OK != 1 || res.Failed != 0 || res.Bytes != 300 {
		t.Errorf("nested: want 1 OK, 300 bytes, got %+v", res)
	}

	res = d.Delete([]Item{{Path: filepath.Join(tmp, "gone"), Size: 50}}, ModePermanent)
	if res.OK != 0 || res.Failed != 1 || res.Bytes != 0 {
		t.Errorf("missing path must fail without freeing bytes, got %+v", res)
	}
}

func TestDeleteCancelled(t *testing.T) {
	tmp := t.TempDir()
	a := mkDir(t, tmp, "a")
	b := mkDir(t, tmp, "b")

	d := New(func(Progress) {})
	d.Cancel()
	res := d.Delete([]Item{{Path: a, Size: 1}, {Path: b, Size: 1}}, ModePermanent)

	if res.OK != 0 {
		t.Errorf("after cancel no item should succeed, got %d", res.OK)
	}
}

func TestDeleteTrash(t *testing.T) {
	if !TrashSupported() {
		t.Skip("trash not available in this environment")
	}
	tmp := t.TempDir()
	a := mkDir(t, tmp, "bersihdisk-test-trash")

	d := New(func(Progress) {})
	res := d.Delete([]Item{{Path: a, Size: 128}}, ModeTrash)

	if res.OK != 1 || res.Failed != 0 {
		t.Fatalf("trash: want 1 OK 0 failed, got %+v", res)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Error("a should have moved to the trash")
	}
}

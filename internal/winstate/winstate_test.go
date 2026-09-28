package winstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTripAndRejects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "window.json")
	want := State{Width: 1200, Height: 800, X: 40, Y: 60, HasPos: true, Maximised: true}
	if err := saveTo(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := loadFrom(path, 900, 620)
	if !ok || got != want {
		t.Fatalf("round trip = %+v, %v; want %+v", got, ok, want)
	}

	if _, ok := loadFrom(path, 2000, 620); ok {
		t.Error("bounds below the minimum must be rejected")
	}
	if _, ok := loadFrom(filepath.Join(t.TempDir(), "missing.json"), 900, 620); ok {
		t.Error("a missing file must fall back to the default")
	}
	os.WriteFile(path, []byte("{not json"), 0o644)
	if _, ok := loadFrom(path, 900, 620); ok {
		t.Error("a corrupt file must fall back to the default")
	}
}

func TestMergeKeepsNormalBoundsWhileMaximised(t *testing.T) {
	prev := State{Width: 1000, Height: 700, X: 10, Y: 20, HasPos: true}
	max := Merge(prev, 2560, 1440, 0, 0, true, false)
	if !max.Maximised || max.Width != 1000 || max.X != 10 {
		t.Errorf("maximised merge must keep the normal bounds: %+v", max)
	}
	normal := Merge(max, 1100, 750, 5, 6, false, false)
	if normal.Maximised || normal.Width != 1100 || normal.X != 5 || !normal.HasPos {
		t.Errorf("normal merge must record live bounds: %+v", normal)
	}
}

func TestPositionOnScreen(t *testing.T) {
	s := State{X: 50, Y: 50, HasPos: true}
	if !s.PositionOnScreen(1920, 1080) {
		t.Error("a position well inside the screen must be restorable")
	}
	s.X = 3000
	if s.PositionOnScreen(1920, 1080) {
		t.Error("a position beyond the screen must not be restored")
	}
	if (State{}).PositionOnScreen(1920, 1080) {
		t.Error("no saved position means nothing to restore")
	}
}

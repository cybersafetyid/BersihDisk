package main

import (
	"testing"

	"bersihdisk/internal/deleter"
	"bersihdisk/internal/safety"
	"bersihdisk/internal/scanner"
)

func appWith(items ...scanner.Item) *App {
	a := NewApp()
	a.scanItems = map[string]scanner.Item{}
	for _, it := range items {
		a.scanItems[it.Path] = it
	}
	return a
}

func TestVetDeleteEnforcesTheAssessment(t *testing.T) {
	a := appWith(
		scanner.Item{Path: "/p/safe", Level: safety.Safe},
		scanner.Item{Path: "/p/warn", Level: safety.Caution},
		scanner.Item{Path: "/p/danger", Level: safety.Danger},
		scanner.Item{Path: "/p/blocked", Level: safety.Blocked, Reasons: []string{"credentials"}},
		scanner.Item{Path: "/p/cache", Level: safety.Safe, KeepRoot: true},
	)

	// Safe items go through in the requested mode.
	items, denied, mode := a.vetDelete(DeleteRequest{Paths: []string{"/p/safe"}, Mode: deleter.ModePermanent})
	if len(items) != 1 || len(denied) != 0 || mode != deleter.ModePermanent {
		t.Fatalf("safe: %v %v %s", items, denied, mode)
	}

	// A caution needs the acknowledgement; without it nothing runs.
	items, denied, _ = a.vetDelete(DeleteRequest{Paths: []string{"/p/safe", "/p/warn"}})
	if len(items) != 0 || len(denied) != 2 {
		t.Fatalf("unacknowledged caution must stop the whole request: %v %v", items, denied)
	}
	items, _, _ = a.vetDelete(DeleteRequest{Paths: []string{"/p/warn"}, Acknowledged: true})
	if len(items) != 1 {
		t.Fatalf("acknowledged caution: %v", items)
	}

	// Danger is forced into the Trash even when permanent was asked for.
	_, _, mode = a.vetDelete(DeleteRequest{Paths: []string{"/p/danger"}, Mode: deleter.ModePermanent, Acknowledged: true})
	if mode != deleter.ModeTrash {
		t.Errorf("danger mode = %s, want trash", mode)
	}

	// Blocked and unknown paths are refused with a reason.
	items, denied, _ = a.vetDelete(DeleteRequest{Paths: []string{"/p/blocked", "/elsewhere"}, Acknowledged: true})
	if len(items) != 0 || len(denied) != 2 {
		t.Fatalf("blocked/unknown: %v %v", items, denied)
	}

	// KeepRoot survives only for the container itself, not for a child inside it.
	items, _, _ = a.vetDelete(DeleteRequest{Paths: []string{"/p/cache", "/p/cache/sub"}})
	if !items[0].KeepRoot || items[1].KeepRoot {
		t.Errorf("keepRoot flags = %v %v", items[0].KeepRoot, items[1].KeepRoot)
	}
}

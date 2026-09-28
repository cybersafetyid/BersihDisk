package uninstall

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// goBins lists the programs `go install` put in GOPATH/bin. Removing one is
// deleting its file, so the recipe has no command.
type goBins struct{}

func (goBins) ID() string { return "go" }

func (goBins) List(_ context.Context, h Host) ([]Installed, error) {
	dir := os.Getenv("GOBIN")
	if dir == "" {
		if h.Home == "" {
			return nil, ErrUnavailable
		}
		dir = filepath.Join(h.Home, "go", "bin")
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, ErrUnavailable
	}
	var pkgs []Installed
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		pkgs = append(pkgs, Installed{
			ID: "go:" + e.Name(), Provider: "go", Kind: KindPackage,
			Name: strings.TrimSuffix(e.Name(), ".exe"), Path: path, Icon: "go",
			recipe: Recipe{Paths: []Residue{{Spec: path, Exact: true}}},
		})
	}
	return pkgs, nil
}

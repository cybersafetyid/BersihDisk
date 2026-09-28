// Package browser lists the contents of a folder so a user can pick exactly what
// to delete, using the same size definition as the scanner.
package browser

import (
	"os"
	"path/filepath"
	"sort"

	"bersihdisk/internal/scanner"
)

// Entry is one child of a browsed directory.
type Entry struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	IsDir  bool   `json:"isDir"`
	IsLink bool   `json:"isLink"` // a symlink: deleting it frees nothing
	Size   int64  `json:"size"`   // bytes; for a directory, its whole subtree
}

// List returns the children of dir, largest first. Directory sizes are measured
// recursively through one shared pool of walkers, so opening a big folder costs a
// single pass over its subtree instead of one pass per child.
func List(dir string) ([]Entry, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(ents))
	var dirPaths []string
	var dirSlots []int
	for _, e := range ents {
		path := filepath.Join(dir, e.Name())
		en := Entry{Name: e.Name(), Path: path, IsDir: e.IsDir()}
		if info, ierr := e.Info(); ierr == nil {
			en.IsLink = info.Mode()&os.ModeSymlink != 0
			if !en.IsDir {
				en.Size = scanner.FileBytes(info)
			}
		}
		if en.IsDir {
			dirPaths = append(dirPaths, path)
			dirSlots = append(dirSlots, len(out))
		}
		out = append(out, en)
	}

	if len(dirPaths) > 0 {
		sizes := scanner.Sizes(dirPaths)
		for i, slot := range dirSlots {
			out[slot].Size = sizes[i]
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

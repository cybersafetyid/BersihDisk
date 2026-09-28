package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ProfileEdit describes the shell start-up lines a tool's installer added. Only
// lines that match exactly what the installer writes are removed.
type ProfileEdit struct {
	Lines  []*regexp.Regexp // any line matching one of these
	Blocks []Block          // from a start line to an end line, both included
}

// Block is a marked section of a profile ("# >>> conda initialize >>>").
type Block struct{ Start, End string }

// profileFiles are the start-up files installers are known to edit, relative to home.
var profileFiles = []string{
	".zshrc", ".zprofile", ".zshenv", ".bashrc", ".bash_profile", ".profile",
	".config/fish/config.fish",
}

// profileJob is one file plus the edit to apply to it.
type profileJob struct {
	file string
	edit ProfileEdit
}

// CleanProfile removes the lines and blocks e matches and returns the result with
// the removed lines. A block whose end marker is missing is left alone: better a
// leftover line than a half-deleted file.
func CleanProfile(content string, e ProfileEdit) (out string, removed []string) {
	lines := strings.SplitAfter(content, "\n")
	keep := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimRight(line, "\r\n")

		if end := blockEnd(lines, i, e.Blocks); end > i {
			for _, l := range lines[i : end+1] {
				removed = append(removed, strings.TrimRight(l, "\r\n"))
			}
			i = end
			continue
		}
		if matchesAny(trimmed, e.Lines) {
			removed = append(removed, trimmed)
			continue
		}
		keep = append(keep, line)
	}
	return strings.Join(keep, ""), removed
}

// blockEnd returns the index of the line closing the block that starts at i, or
// -1 when line i opens no block or the block never closes.
func blockEnd(lines []string, i int, blocks []Block) int {
	for _, b := range blocks {
		if !strings.Contains(lines[i], b.Start) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.Contains(lines[j], b.End) {
				return j
			}
		}
	}
	return -1
}

func matchesAny(line string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// profileJobs lists the existing start-up files that hold something e would
// remove, with the lines that would go.
func profileJobs(home string, e ProfileEdit) (jobs []profileJob, preview [][]string) {
	for _, rel := range profileFiles {
		file := filepath.Join(home, filepath.FromSlash(rel))
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if _, removed := CleanProfile(string(data), e); len(removed) > 0 {
			jobs = append(jobs, profileJob{file: file, edit: e})
			preview = append(preview, removed)
		}
	}
	return jobs, preview
}

// apply rewrites the file without the matched lines. The original is kept next to
// it as <file>.bersihdisk-<timestamp>.bak, and the write goes through a temporary
// file so an interruption never leaves a half-written profile.
func (j profileJob) apply() (removed int, err error) {
	data, err := os.ReadFile(j.file)
	if err != nil {
		return 0, err
	}
	st, err := os.Stat(j.file)
	if err != nil {
		return 0, err
	}
	out, gone := CleanProfile(string(data), j.edit)
	if len(gone) == 0 {
		return 0, nil // already clean
	}
	backup := fmt.Sprintf("%s.bersihdisk-%s.bak", j.file, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backup, data, st.Mode().Perm()); err != nil {
		return 0, fmt.Errorf("could not back up %s: %w", filepath.Base(j.file), err)
	}
	tmp := j.file + ".bersihdisk.tmp"
	if err := os.WriteFile(tmp, []byte(out), st.Mode().Perm()); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, j.file); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	return len(gone), nil
}

// ---- reusable patterns for the catalog ----

// exportsVar matches `export NAME=...` and `NAME=...`.
func exportsVar(name string) *regexp.Regexp {
	return regexp.MustCompile(`^\s*(export\s+)?` + regexp.QuoteMeta(name) + `=.*$`)
}

// pathMentions matches lines that put something containing sub on PATH, in
// POSIX and fish syntax.
func pathMentions(sub string) *regexp.Regexp {
	q := regexp.QuoteMeta(sub)
	return regexp.MustCompile(`^\s*((export\s+)?PATH=|set\s+(-\w+\s+)*PATH\b|fish_add_path\b).*` + q + `.*$`)
}

// sources matches `. file`, `source file`, `eval "$(...)"` and the guarded forms
// installers write — `[ -s file ] && . file`, `[[ -r file ]] && . file` (nvm) —
// for lines mentioning sub.
func sources(sub string) *regexp.Regexp {
	q := regexp.QuoteMeta(sub)
	return regexp.MustCompile(`^\s*(\.|source|eval|\[\[?\s+-\w)\s.*` + q + `.*$`)
}

package uninstall

import "strings"

// dropPathEntries removes the wanted entries from a PATH-style value and returns
// the new value with what was dropped. Entries compare case-insensitively without
// trailing separators, as Windows does.
func dropPathEntries(value string, drop []string, sep string) (string, []string) {
	want := map[string]bool{}
	for _, d := range drop {
		want[normEntry(d)] = true
	}
	var keep, gone []string
	for _, e := range strings.Split(value, sep) {
		if e != "" && want[normEntry(e)] {
			gone = append(gone, e)
			continue
		}
		keep = append(keep, e)
	}
	return strings.Join(keep, sep), gone
}

func normEntry(e string) string {
	e = strings.ReplaceAll(strings.TrimSpace(e), `\`, "/")
	return strings.ToLower(strings.TrimRight(e, "/"))
}

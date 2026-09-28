package reveal

import "testing"

func TestOpenURLRefusesNonWebSchemes(t *testing.T) {
	for _, raw := range []string{"", "file:///Applications/Calculator.app", "/etc/passwd", "javascript:alert(1)", "ms-msdt:/id"} {
		if err := OpenURL(raw); err == nil {
			t.Errorf("OpenURL(%q) must be refused", raw)
		}
	}
}

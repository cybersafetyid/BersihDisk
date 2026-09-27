//go:build !darwin

// detectDarwin stub for non-macOS so the package still compiles.
package drive

import "errors"

func detectDarwin() ([]Info, error) {
	return nil, errors.New("detectDarwin is only available on macOS")
}

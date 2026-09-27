//go:build !linux

// detectLinux stub for non-Linux so the package still compiles.
package drive

import "errors"

func detectLinux() ([]Info, error) {
	return nil, errors.New("detectLinux is only available on Linux")
}

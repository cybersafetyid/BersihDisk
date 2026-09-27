//go:build darwin

package appicon

// #cgo LDFLAGS: -framework AppKit -framework Foundation
// #include <stdlib.h>
// #include "appicon_darwin.h"
import "C"

import (
	"errors"
)

// Supported reports that macOS can swap the Dock icon while running.
func Supported() bool { return true }

// Set replaces the running application's icon with the given PNG.
func Set(png []byte) error {
	if len(png) == 0 {
		return errors.New("icon data is empty")
	}
	buf := C.CBytes(png)
	defer C.free(buf)
	if C.BDSetAppIcon((*C.char)(buf), C.int(len(png))) == 0 {
		return errors.New("the system refused the icon data")
	}
	return nil
}

// Package reveal opens a path in the operating system's file manager.
package reveal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Open shows path in Finder (macOS), Explorer (Windows) or the default file
// manager (Linux). For a file the parent folder is opened with the file selected,
// because that is where its space actually lives.
//
// The command is started directly rather than through a shell, so a path holding
// spaces or shell punctuation cannot be interpreted as a command.
func Open(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("path not accessible: %w", err)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		if st.IsDir() {
			cmd = exec.Command("open", path)
		} else {
			cmd = exec.Command("open", "-R", path)
		}
	case "windows":
		if st.IsDir() {
			cmd = exec.Command("explorer", path)
		} else {
			cmd = exec.Command("explorer", "/select,"+path)
		}
	default:
		target := path
		if !st.IsDir() {
			target = filepath.Dir(path)
		}
		cmd = exec.Command("xdg-open", target)
	}

	// Start only: Explorer reports a non-zero exit even when it succeeded, and a
	// file manager may stay open long after the request. Start still fails when
	// the helper binary is missing, which is the error worth surfacing.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start the file manager: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Launch hands a downloaded file to the operating system so it can be installed:
// macOS mounts a .dmg, offers a .pkg, or expands a .zip; Windows runs an .exe,
// installs an .msi, or falls back to the registered handler; Linux uses xdg-open.
func Launch(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}
	if st.IsDir() {
		return Open(path)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe":
			cmd = exec.Command(path)
		case ".msi":
			cmd = exec.Command("msiexec", "/i", path)
		default:
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
		}
	default:
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open the downloaded file: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// OpenURL hands a web or mail address to the user's default handler. The URL is
// passed as an argument, never evaluated by a shell.
func OpenURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("no url given")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open the link: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

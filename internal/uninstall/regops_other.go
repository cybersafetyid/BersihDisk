//go:build !windows

package uninstall

import "context"

// The Windows registry only exists on Windows; elsewhere these are inert.

type regApp struct {
	Key, Name, Version, Publisher, InstallLocation, Uninstall, QuietUninstall, DisplayIcon string
	SizeKB                                                                                 uint32
}

func listRegistryApps() ([]regApp, error)                      { return nil, ErrUnavailable }
func registryKeyExists(string) bool                            { return false }
func deleteRegistryKey(string, string) error                   { return ErrUnavailable }
func userPathHas([]string) []string                            { return nil }
func removeUserPathEntries([]string, string) ([]string, error) { return nil, ErrUnavailable }
func runRaw(context.Context, string) error                     { return ErrUnavailable }

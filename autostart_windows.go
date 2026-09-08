//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

const runValueName = "Ports"

// startOnBootEnabled reports whether the Run key entry exists.
func startOnBootEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	val, _, err := k.GetStringValue(runValueName)
	return err == nil && val != ""
}

// setStartOnBoot adds or removes the HKCU Run entry that launches Ports
// hidden at login.
func setStartOnBoot(enable bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve executable: %w", err)
	}
	exe = filepath.Clean(exe)

	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("could not open registry Run key: %w", err)
	}
	defer k.Close()

	if !enable {
		if err := k.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	cmd := `"` + exe + `" --hidden`
	return k.SetStringValue(runValueName, cmd)
}

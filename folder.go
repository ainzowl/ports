package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func openFolder(ctx context.Context, source, distro, path string) error {
	switch source {
	case "windows":
		return openContainingFolder(path)
	case "wsl":
		if path == "" {
			return fmt.Errorf("no path available for this process")
		}
		dir := path
		if i := strings.LastIndex(path, "/"); i > 0 {
			dir = path[:i]
		}
		unc := fmt.Sprintf(`\\wsl.localhost\%s%s`, distro, dir)
		if _, err := osStat(unc); err != nil {
			unc = fmt.Sprintf(`\\wsl$\%s%s`, distro, dir)
		}
		// No HideWindow here either: the flag would hide the Explorer window.
		cmd := exec.Command("explorer.exe", unc)
		return cmd.Start()
	default:
		return fmt.Errorf("unknown source %q", source)
	}
}

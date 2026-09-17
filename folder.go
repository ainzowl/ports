package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openFolder(ctx context.Context, source, distro, path string) error {
	switch source {
	case "windows":
		return openContainingFolder(path)
	case "docker":
		// The container's "location" is Docker Desktop itself.
		return openDockerDesktop(ctx)
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

// openDockerDesktop brings Docker Desktop to the foreground so the user can
// inspect the container (logs, env, volumes). Best-effort: if the desktop app
// isn't installed (engine-only installs), fall back to the docker:// scheme.
func openDockerDesktop(ctx context.Context) error {
	if p := dockerDesktopExe(); p != "" {
		return exec.Command("explorer.exe", "/select,"+p).Start()
	}
	return exec.CommandContext(ctx, "cmd", "/c", "start", "", "docker://").Start()
}

// dockerDesktopExe finds Docker Desktop.exe in the usual install locations.
func dockerDesktopExe() string {
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Docker", "Docker", "Docker Desktop.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Docker", "Docker Desktop.exe"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

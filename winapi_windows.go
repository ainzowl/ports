//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"
)

func windowsExePath(pid int) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf[:size]))
}

func openContainingFolder(path string) error {
	if path == "" {
		return fmt.Errorf("no path available for this process")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %s", path)
	}
	// explorer.exe is a GUI app and never flashes a console. It must NOT be
	// started with HideWindow: the SW_HIDE flag propagates to the new folder
	// window, which then opens invisibly.
	return exec.Command("explorer.exe", "/select,"+path).Start()
}

func wslExePath(ctx context.Context, distro string, pid int) string {
	out, err := wslRun(ctx, distro, fmt.Sprintf("readlink -f /proc/%d/exe 2>/dev/null || readlink /proc/%d/exe 2>/dev/null", pid, pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(decodeWslText(out))
}

func osStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

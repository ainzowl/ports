//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func windowsExePath(pid int) string { return "" }

func wslExePath(ctx context.Context, distro string, pid int) string {
	return ""
}

func osStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func explorerSysProcAttr() *syscall.SysProcAttr { return nil }
func openContainingFolder(path string) error {
	if path == "" {
		return fmt.Errorf("no path available for this process")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func explorerAvailable() bool { return true }

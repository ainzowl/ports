//go:build !windows

package main

import "fmt"

func startOnBootEnabled() bool { return false }

func setStartOnBoot(enable bool) error {
	return fmt.Errorf("start on boot is only supported on Windows")
}

//go:build !windows

package main

import (
	"os/exec"
	"runtime"
	"syscall"
)

// openBrowser opens the given URL in the default browser, detached from the
// TUI process so quitting the program does not kill the browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

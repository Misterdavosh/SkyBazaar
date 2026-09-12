//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

// openBrowser opens the given URL with the default handler via rundll32,
// detached from the TUI process.
func openBrowser(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rundll32: %w", err)
	}
	return nil
}

package webbrowser

import (
	"github.com/pkg/browser"
	"os/exec"
	"strings"
)

// NewNativeLauncher creates and returns a Launcher that will attempt to interact
// with the browser-launching mechanisms of the operating system where the
// program is currently running.
func NewNativeLauncher() Launcher {
	return nativeLauncher{}
}

type nativeLauncher struct{}

func hasProgram(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (l nativeLauncher) OpenURL(url string) error {
	// Windows Subsystem for Linux (bash for Windows) doesn't have xdg-open available
	// but you can execute cmd.exe from there; try to identify it
	if !hasProgram("xdg-open") && hasProgram("cmd.exe") {
		r := strings.NewReplacer("&", "^&")
		exec.Command("cmd.exe", "/c", "start", r.Replace(url)).Run()
	}

	return browser.OpenURL(url)
}

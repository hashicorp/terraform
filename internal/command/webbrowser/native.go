package webbrowser

import (
	"github.com/pkg/browser"
)

// NewNativeLauncher creates and returns a Launcher that will attempt to interact
// with the browser-launching mechanisms of the operating system where the
// program is currently running.
func NewNativeLauncher() Launcher {
	return nativeLauncher{}
}

type nativeLauncher struct{}

func (l nativeLauncher) OpenURL(url string) error {
	return browser.OpenURL(url)
}

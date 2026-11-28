// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package webbrowser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/browser"
)

// NewBrowserEnvLauncher creates and returns a Launcher that will attempt to use 
// BROWSER environment , otherwise full back to the browser-launching mechanisms of 
// the operating system where the program is currently running.
func NewBrowserEnvLauncher() Launcher {
	return browserEnvLauncher{}
}

type browserEnvLauncher struct{}

func (l browserEnvLauncher) OpenURL(url string) error {
	browserEnv := os.Getenv("BROWSER")
	if browserEnv != "" {
		browserSh := fmt.Sprintf("%s '%s'", browserEnv, url)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", browserSh)
		_, err := cmd.CombinedOutput()
		return err
	}

	return browser.OpenURL(url)
}

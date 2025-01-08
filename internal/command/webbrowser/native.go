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

// NewNativeLauncher creates and returns a Launcher that will attempt to interact
// with the browser-launching mechanisms of the operating system where the
// program is currently running.
func NewNativeLauncher() Launcher {
	return nativeLauncher{}
}

type nativeLauncher struct{}

func (l nativeLauncher) OpenURL(url string) error {
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

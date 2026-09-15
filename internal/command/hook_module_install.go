// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/initwd"
)

// uiModuleInstallHooks is used to log during module download and installation.
// Currently this occurs in both the init and get commands.
type uiModuleInstallHooks struct {
	initwd.ModuleInstallHookImpl
	ShowLocalPaths bool
	View           views.ModuleInstallationLogger
}

var _ initwd.ModuleInstallHook = uiModuleInstallHooks{}

func (h uiModuleInstallHooks) Download(modulePath, packageAddr string, v *version.Version) {
	var message string
	if v != nil {
		message = fmt.Sprintf("Downloading %s %s for %s...", packageAddr, v, modulePath)
	} else {
		message = fmt.Sprintf("Downloading %s for %s...", packageAddr, modulePath)
	}

	h.View.LogModuleDownload(message)
}

func (h uiModuleInstallHooks) Install(modulePath string, v *version.Version, localDir string) {
	var message string
	if h.ShowLocalPaths {
		message = fmt.Sprintf("- %s in %s", modulePath, localDir)
	} else {
		message = fmt.Sprintf("- %s", modulePath)
	}

	h.View.LogModuleInstallation(message)
}

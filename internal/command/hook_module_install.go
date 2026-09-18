// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
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
	h.View.LogModuleDownload(packageAddr, v, modulePath)
}

func (h uiModuleInstallHooks) Install(modulePath string, v *version.Version, localDir string) {
	if h.ShowLocalPaths {
		h.View.LogModuleInstallation(modulePath, localDir)
	} else {
		h.View.LogModuleInstallation(modulePath, "")
	}
}

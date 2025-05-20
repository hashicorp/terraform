// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"

	"github.com/hashicorp/cli"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/initwd"
)

type view interface {
	Log(message string, params ...any)
}
type uiModuleInstallHooks struct {
	initwd.ModuleInstallHooksImpl
	Ui             cli.Ui
	ShowLocalPaths bool
	View           view
}

var _ initwd.ModuleInstallHooks = uiModuleInstallHooks{}

func (h uiModuleInstallHooks) Download(modulePath, packageAddr string, v *version.Version) {
	if v != nil {
		h.log(fmt.Sprintf("Downloading %s %s for %s...", packageAddr, v, modulePath))
	} else {
		h.log(fmt.Sprintf("Downloading %s for %s...", packageAddr, modulePath))
	}
}

func (h uiModuleInstallHooks) Install(modulePath string, v *version.Version, localDir string) {
	if h.ShowLocalPaths {
		h.log(fmt.Sprintf("- %s in %s", modulePath, localDir))
	} else {
		h.log(fmt.Sprintf("- %s", modulePath))
	}
}

func (h uiModuleInstallHooks) log(message string) {
	switch h.View.(type) {
	case view:
		// there is no unformatted option for the View interface, so we need to
		// pass message as a parameter to avoid double escaping % characters
		h.View.Log("%s", message)
	default:
		h.Ui.Info(message)
	}
}

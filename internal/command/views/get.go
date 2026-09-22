// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Get is the view used for the `get` command.
type Get interface {
	ModuleInstallationLogger
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewGet(viewType arguments.ViewType, view *View) Get {
	switch viewType {
	case arguments.ViewHuman:
		return &GetHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// GetHuman renders module installation progress and diagnostics for direct consumption.
type GetHuman struct {
	view *View
}

var _ Get = (*GetHuman)(nil)

func (v *GetHuman) LogModuleDownload(packageAddr string, version *version.Version, modulePath string) {
	var message string
	if version == nil {
		message = fmt.Sprintf(moduleDownloadHuman, packageAddr, modulePath)
	} else {
		message = fmt.Sprintf(moduleDownloadWithVersionHuman, packageAddr, version, modulePath)
	}
	v.view.streams.Println(message)
}

func (v *GetHuman) LogModuleInstallation(modulePath string) {
	message := fmt.Sprintf(moduleInstallationHuman, modulePath)
	v.view.streams.Println(message)
}

func (v *GetHuman) LogModuleInstallationWithLocalPath(modulePath, localDir string) {
	message := fmt.Sprintf(moduleInstallationWithLocalPathHuman, modulePath, localDir)
	v.view.streams.Println(message)
}

func (v *GetHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

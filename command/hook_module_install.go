package command

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/mitchellh/cli"
)

type uiModuleInstallHooks struct {
	configload.InstallHooksImpl
	Ui             cli.Ui
	ShowLocalPaths bool
}

var _ configload.InstallHooks = uiModuleInstallHooks{}

func (h uiModuleInstallHooks) Download(modulePath, packageAddr string, v *version.Version) {
	if v != nil {
		h.Ui.Info(fmt.Sprintf("Downloading %s %s for %s...", packageAddr, v, modulePath))
	} else {
		h.Ui.Info(fmt.Sprintf("Downloading %s for %s...", packageAddr, modulePath))
	}
}

func (h uiModuleInstallHooks) Install(modulePath string, v *version.Version, localDir string) {
	if h.ShowLocalPaths {
		h.Ui.Info(fmt.Sprintf("- %s in %s", modulePath, localDir))
	} else {
		h.Ui.Info(fmt.Sprintf("- %s", modulePath))
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	version "github.com/hashicorp/go-version"
)

type ModuleInstallationLogger interface {
	// LogModuleDownload logs the start of a module download.
	// This may or may not include a version (nil versions impact the output format)
	LogModuleDownload(packageAddr string, version *version.Version, modulePath string)

	// LogModuleInstallation logs the completion of a module installation without showing details about where it's downloaded to.
	LogModuleInstallation(modulePath string)

	// LogModuleInstallationWithLocalPath logs the completion of a module installation including the local path.
	LogModuleInstallationWithLocalPath(modulePath, localDir string)
}

const (
	moduleDownloadHuman            = "Downloading %s for %s..."
	moduleDownloadWithVersionHuman = "Downloading %s %s for %s..."

	moduleInstallationHuman              = "- %s"
	moduleInstallationWithLocalPathHuman = "- %s in %s"
)

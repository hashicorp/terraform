// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

type ModuleInstallationLogger interface {
	// TODO: Refactor calling code so that these methods can format the messages internally
	LogModuleDownload(message string)
	LogModuleInstallation(message string)

	LogModuleUpgrade()
	LogModuleInitialization()
}

package cli

import (
	"github.com/posener/complete/cmd/install"
)

// autocompleteInstaller is an interface to be implemented to perform the
// autocomplete installation and uninstallation with a CLI.
//
// This interface is not exported because it only exists for unit tests
// to be able to test that the installation is called properly.
type autocompleteInstaller interface {
	Install(string) error
	Uninstall(string) error
}

// realAutocompleteInstaller uses the real install package to do the
// install/uninstall.
type realAutocompleteInstaller struct{}

func (i *realAutocompleteInstaller) Install(cmd string) error {
	return install.Install(cmd)
}

func (i *realAutocompleteInstaller) Uninstall(cmd string) error {
	return install.Uninstall(cmd)
}

// mockAutocompleteInstaller is used for tests to record the install/uninstall.
type mockAutocompleteInstaller struct {
	InstallCalled   bool
	UninstallCalled bool
}

func (i *mockAutocompleteInstaller) Install(cmd string) error {
	i.InstallCalled = true
	return nil
}

func (i *mockAutocompleteInstaller) Uninstall(cmd string) error {
	i.UninstallCalled = true
	return nil
}

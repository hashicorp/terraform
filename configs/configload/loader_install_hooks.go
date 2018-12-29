package configload

import version "github.com/hashicorp/go-version"

// InstallHooks is an interface used to provide notifications about the
// installation process being orchestrated by InstallModules.
//
// This interface may have new methods added in future, so implementers should
// embed InstallHooksImpl to get no-op implementations of any unimplemented
// methods.
type InstallHooks interface {
	// Download is called for modules that are retrieved from a remote source
	// before that download begins, to allow a caller to give feedback
	// on progress through a possibly-long sequence of downloads.
	Download(moduleAddr, packageAddr string, version *version.Version)

	// Install is called for each module that is installed, even if it did
	// not need to be downloaded from a remote source.
	Install(moduleAddr string, version *version.Version, localPath string)
}

// InstallHooksImpl is a do-nothing implementation of InstallHooks that
// can be embedded in another implementation struct to allow only partial
// implementation of the interface.
type InstallHooksImpl struct {
}

func (h InstallHooksImpl) Download(moduleAddr, packageAddr string, version *version.Version) {
}

func (h InstallHooksImpl) Install(moduleAddr string, version *version.Version, localPath string) {
}

var _ InstallHooks = InstallHooksImpl{}

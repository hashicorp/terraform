package configload

import (
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/spf13/afero"
)

type moduleMgr struct {
	FS afero.Afero

	// Dir is the path where descendent modules are (or will be) installed.
	Dir string

	// Services is a service discovery client that will be used to find
	// remote module registry endpoints. This object may be pre-loaded with
	// cached discovery information.
	Services *disco.Disco

	// Creds provides optional credentials for communicating with service hosts.
	Creds auth.CredentialsSource

	// Registry is a client for the module registry protocol, which is used
	// when a module is requested from a registry source.
	Registry *registry.Client

	// manifest tracks the currently-installed modules for this manager.
	//
	// The loader may read this. Only the installer may write to it, and
	// after a set of updates are completed the installer must call
	// writeModuleManifestSnapshot to persist a snapshot of the manifest
	// to disk for use on subsequent runs.
	manifest moduleManifest
}

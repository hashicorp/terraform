package configload

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/spf13/afero"
)

const manifestName = "modules.json"

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
}

func (m *moduleMgr) loadManifestRecords() (moduleRecords, error) {
	type moduleManifest struct {
		Modules []moduleRecord
	}

	manifest := moduleManifest{}

	manifestPath := filepath.Join(m.Dir, manifestName)
	data, err := m.FS.ReadFile(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	for _, r := range manifest.Modules {
		r.Version, err = version.NewVersion(r.VersionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid version string %q for %q: %s", r.VersionStr, r.SourceAddr, err)
		}
	}

	return manifest.Modules, nil
}

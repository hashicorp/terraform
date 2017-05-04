package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/plugin/discovery"
)

// mockGetProvider providers a GetProvider method for testing automatic
// provider downloads
type mockGetProvider struct {
	// A map of provider names to available versions.
	// The tests expect the versions to be in order from newest to oldest.
	Providers map[string][]string
}

func (m mockGetProvider) FileName(provider, version string) string {
	return fmt.Sprintf("terraform-provider-%s-V%s-X4", provider, version)
}

// GetProvider will check the Providers map to see if it can find a suitable
// version, and put an empty file in the dst directory.
func (m mockGetProvider) GetProvider(dst, provider string, req discovery.Constraints) error {
	versions := m.Providers[provider]
	if len(versions) == 0 {
		return fmt.Errorf("provider %q not found", provider)
	}

	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("error creating plugins directory: %s", err)
	}

	for _, v := range versions {
		version, err := discovery.VersionStr(v).Parse()
		if err != nil {
			panic(err)
		}

		if req.Has(version) {
			// provider filename
			name := m.FileName(provider, v)
			path := filepath.Join(dst, name)
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("error fetching provider: %s", err)
			}
			f.Close()
			return nil
		}
	}

	return fmt.Errorf("no suitable version for provider %q found with constraints %s", provider, req)
}

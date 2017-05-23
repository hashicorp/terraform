package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/plugin/discovery"
)

func (m *Meta) providerLockFile() string {
	return filepath.Join(m.pluginDir(), "providers.json")
}

// loadProvidersLock loads the current "locked" SHA256 hashes of provider
// plugins, persisted in a file in the data directory, so that it becomes
// impossible to load provider plugins other than those locked.
func (m *Meta) loadProvidersLock() error {
	var err error
	m.providersSHA256, err = m.lockedProvidersSHA256()
	return err
}

func (m *Meta) persistProvidersLock() error {
	return m.saveLockedProvidersSHA256(m.providersSHA256)
}

// setProvidersLock overrides the "locked" SHA256 hashes of provider plugins,
// for situations where the desired set of plugins comes from somewhere other
// than the data dir. This is used, for example, to require provider plugins
// used to apply a plan to exactly match those used to generate the plan.
func (m *Meta) setProvidersLock(digests map[string][]byte) {
	m.providersSHA256 = digests
}

func makeProvidersLock(metas map[string]discovery.PluginMeta) (map[string][]byte, error) {
	ret := make(map[string][]byte)
	for name, meta := range metas {
		var err error
		ret[name], err = meta.SHA256()
		if err != nil {
			return nil, fmt.Errorf("failed to calculate digest for provider plugin %q: %s", name, err)
		}
	}
	return ret, nil
}

// lockedProvidersSHA256 returns the "locked" SHA256 hashes of provider plugins
// that was created by "terraform init". The lock map should be used for any
// operation that interacts with providers in a way that may affect external
// resources, to ensure that versions don't accidentally drift during use.
//
// The one exception to this rule is when applying plans. In that case, the
// plan.ProvidersSHA256 map should be used instead, to ensure that providers
// are still the same as they were when the plan was created.
func (m *Meta) lockedProvidersSHA256() (map[string][]byte, error) {
	buf, err := ioutil.ReadFile(m.providerLockFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("provider plugins not yet initialised; use \"terraform init\" to configure them")
		}
		return nil, fmt.Errorf("failed to read provider lock file: %s", err)
	}

	var digestsStr map[string]string
	err = json.Unmarshal(buf, &digestsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provider lock file: %s", err)
	}

	digests := make(map[string][]byte)
	for k, digestStr := range digestsStr {
		var digest []byte
		_, err := fmt.Sscanf(digestStr, "%x", &digest)
		if err != nil {
			return nil, fmt.Errorf("provider %s digest is corrupted; run \"terraform init\" to re-initialize provider plugins", k)
		}
		digests[k] = digest
	}

	return digests, nil
}

// saveLockedProvidersSHA256 replaces the manifest of locked SHA256 hashes
// of provider plugins, as would be returned by lockedProvidersSHA256.
func (m *Meta) saveLockedProvidersSHA256(digests map[string][]byte) error {
	digestsStr := make(map[string]string)
	for k, digest := range digests {
		digestsStr[k] = fmt.Sprintf("%x", digest)
	}
	buf, err := json.MarshalIndent(digestsStr, "", "    ")
	if err != nil {
		// should never happen
		return fmt.Errorf("failed to serialize provider plugins as JSON: %s", err)
	}

	lockFile := m.providerLockFile()
	os.MkdirAll(filepath.Dir(lockFile), os.ModePerm) // ignore error since our WriteFile below will catch it

	err = ioutil.WriteFile(lockFile, buf, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to save provider plugin versions: %s", err)
	}

	return nil

}

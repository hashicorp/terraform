package cliconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/configs/hcl2shim"
	pluginDiscovery "github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/svchost"
	svcauth "github.com/hashicorp/terraform/svchost/auth"
)

// credentialsConfigFile returns the path for the special configuration file
// that the credentials source will use when asked to save or forget credentials
// and when a "credentials helper" program is not active.
func credentialsConfigFile() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "credentials.tfrc.json"), nil
}

// CredentialsSource creates and returns a service credentials source whose
// behavior depends on which "credentials" and "credentials_helper" blocks,
// if any, are present in the receiving config.
func (c *Config) CredentialsSource(helperPlugins pluginDiscovery.PluginMetaSet) (*CredentialsSource, error) {
	credentialsFilePath, err := credentialsConfigFile()
	if err != nil {
		// If we managed to load a Config object at all then we would already
		// have located this file, so this error is very unlikely.
		return nil, fmt.Errorf("can't locate credentials file: %s", err)
	}

	var helper svcauth.CredentialsSource
	var helperType string
	for givenType, givenConfig := range c.CredentialsHelpers {
		available := helperPlugins.WithName(givenType)
		if available.Count() == 0 {
			log.Printf("[ERROR] Unable to find credentials helper %q; ignoring", helperType)
			break
		}

		selected := available.Newest()

		helperSource := svcauth.HelperProgramCredentialsSource(selected.Path, givenConfig.Args...)
		helper = svcauth.CachingCredentialsSource(helperSource) // cached because external operation may be slow/expensive
		helperType = givenType

		// There should only be zero or one "credentials_helper" blocks. We
		// assume that the config was validated earlier and so we don't check
		// for extras here.
		break
	}

	return c.credentialsSource(helperType, helper, credentialsFilePath), nil
}

// EmptyCredentialsSourceForTests constructs a CredentialsSource with
// no credentials pre-loaded and which writes new credentials to a file
// at the given path.
//
// As the name suggests, this function is here only for testing and should not
// be used in normal application code.
func EmptyCredentialsSourceForTests(credentialsFilePath string) *CredentialsSource {
	cfg := &Config{}
	return cfg.credentialsSource("", nil, credentialsFilePath)
}

// credentialsSource is an internal factory for the credentials source which
// allows overriding the credentials file path, which allows setting it to
// a temporary file location when testing.
func (c *Config) credentialsSource(helperType string, helper svcauth.CredentialsSource, credentialsFilePath string) *CredentialsSource {
	configured := map[svchost.Hostname]cty.Value{}
	for userHost, creds := range c.Credentials {
		host, err := svchost.ForComparison(userHost)
		if err != nil {
			// We expect the config was already validated by the time we get
			// here, so we'll just ignore invalid hostnames.
			continue
		}

		// For now our CLI config continues to use HCL 1.0, so we'll shim it
		// over to HCL 2.0 types. In future we will hopefully migrate it to
		// HCL 2.0 instead, and so it'll be a cty.Value already.
		credsV := hcl2shim.HCL2ValueFromConfigValue(creds)
		configured[host] = credsV
	}

	writableLocal := readHostsInCredentialsFile(credentialsFilePath)
	unwritableLocal := map[svchost.Hostname]cty.Value{}
	for host, v := range configured {
		if _, exists := writableLocal[host]; !exists {
			unwritableLocal[host] = v
		}
	}

	return &CredentialsSource{
		configured:          configured,
		unwritable:          unwritableLocal,
		credentialsFilePath: credentialsFilePath,
		helper:              helper,
		helperType:          helperType,
	}
}

// CredentialsSource is an implementation of svcauth.CredentialsSource
// that can read and write the CLI configuration, and possibly also delegate
// to a credentials helper when configured.
type CredentialsSource struct {
	// configured describes the credentials explicitly configured in the CLI
	// config via "credentials" blocks. This map will also change to reflect
	// any writes to the special credentials.tfrc.json file.
	configured map[svchost.Hostname]cty.Value

	// unwritable describes any credentials explicitly configured in the
	// CLI config in any file other than credentials.tfrc.json. We cannot update
	// these automatically because only credentials.tfrc.json is subject to
	// editing by this credentials source.
	unwritable map[svchost.Hostname]cty.Value

	// credentialsFilePath is the full path to the credentials.tfrc.json file
	// that we'll update if any changes to credentials are requested and if
	// a credentials helper isn't available to use instead.
	//
	// (This is a field here rather than just calling credentialsConfigFile
	// directly just so that we can use temporary file location instead during
	// testing.)
	credentialsFilePath string

	// helper is the credentials source representing the configured credentials
	// helper, if any. When this is non-nil, it will be consulted for any
	// hostnames not explicitly represented in "configured". Any writes to
	// the credentials store will also be sent to a configured helper instead
	// of the credentials.tfrc.json file.
	helper svcauth.CredentialsSource

	// helperType is the name of the type of credentials helper that is
	// referenced in "helper", or the empty string if "helper" is nil.
	helperType string
}

// Assertion that credentialsSource implements CredentialsSource
var _ svcauth.CredentialsSource = (*CredentialsSource)(nil)

func (s *CredentialsSource) ForHost(host svchost.Hostname) (svcauth.HostCredentials, error) {
	v, ok := s.configured[host]
	if ok {
		return svcauth.HostCredentialsFromObject(v), nil
	}

	if s.helper != nil {
		return s.helper.ForHost(host)
	}

	return nil, nil
}

func (s *CredentialsSource) StoreForHost(host svchost.Hostname, credentials svcauth.HostCredentialsWritable) error {
	return s.updateHostCredentials(host, credentials)
}

func (s *CredentialsSource) ForgetForHost(host svchost.Hostname) error {
	return s.updateHostCredentials(host, nil)
}

// HostCredentialsLocation returns a value indicating what type of storage is
// currently used for the credentials for the given hostname.
//
// The current location of credentials determines whether updates are possible
// at all and, if they are, where any updates will be written.
func (s *CredentialsSource) HostCredentialsLocation(host svchost.Hostname) CredentialsLocation {
	if _, unwritable := s.unwritable[host]; unwritable {
		return CredentialsInOtherFile
	}
	if _, exists := s.configured[host]; exists {
		return CredentialsInPrimaryFile
	}
	if s.helper != nil {
		return CredentialsViaHelper
	}
	return CredentialsNotAvailable
}

// CredentialsFilePath returns the full path to the local credentials
// configuration file, so that a caller can mention this path in order to
// be transparent about where credentials will be stored.
//
// This file will be used for writes only if HostCredentialsLocation for the
// relevant host returns CredentialsInPrimaryFile or CredentialsNotAvailable.
//
// The credentials file path is found relative to the current user's home
// directory, so this function will return an error in the unlikely event that
// we cannot determine a suitable home directory to resolve relative to.
func (s *CredentialsSource) CredentialsFilePath() (string, error) {
	return s.credentialsFilePath, nil
}

// CredentialsHelperType returns the name of the configured credentials helper
// type, or an empty string if no credentials helper is configured.
func (s *CredentialsSource) CredentialsHelperType() string {
	return s.helperType
}

func (s *CredentialsSource) updateHostCredentials(host svchost.Hostname, new svcauth.HostCredentialsWritable) error {
	switch loc := s.HostCredentialsLocation(host); loc {
	case CredentialsInOtherFile:
		return ErrUnwritableHostCredentials(host)
	case CredentialsInPrimaryFile, CredentialsNotAvailable:
		// If the host already has credentials stored locally then we'll update
		// them locally too, even if there's a credentials helper configured,
		// because the user might be intentionally retaining this particular
		// host locally for some reason, e.g. if the credentials helper is
		// talking to some shared remote service like HashiCorp Vault.
		return s.updateLocalHostCredentials(host, new)
	case CredentialsViaHelper:
		// Delegate entirely to the helper, then.
		if new == nil {
			return s.helper.ForgetForHost(host)
		}
		return s.helper.StoreForHost(host, new)
	default:
		// Should never happen because the above cases are exhaustive
		return fmt.Errorf("invalid credentials location %#v", loc)
	}
}

func (s *CredentialsSource) updateLocalHostCredentials(host svchost.Hostname, new svcauth.HostCredentialsWritable) error {
	// This function updates the local credentials file in particular,
	// regardless of whether a credentials helper is active. It should be
	// called only indirectly via updateHostCredentials.

	filename, err := s.CredentialsFilePath()
	if err != nil {
		return fmt.Errorf("unable to determine credentials file path: %s", err)
	}

	oldSrc, err := ioutil.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %s", filename, err)
	}

	var raw map[string]interface{}

	if len(oldSrc) > 0 {
		// When decoding we use a custom decoder so we can decode any numbers as
		// json.Number and thus avoid losing any accuracy in our round-trip.
		dec := json.NewDecoder(bytes.NewReader(oldSrc))
		dec.UseNumber()
		err = dec.Decode(&raw)
		if err != nil {
			return fmt.Errorf("cannot read %s: %s", filename, err)
		}
	} else {
		raw = make(map[string]interface{})
	}

	rawCredsI, ok := raw["credentials"]
	if !ok {
		rawCredsI = make(map[string]interface{})
		raw["credentials"] = rawCredsI
	}
	rawCredsMap, ok := rawCredsI.(map[string]interface{})
	if !ok {
		return fmt.Errorf("credentials file %s has invalid value for \"credentials\" property: must be a JSON object", filename)
	}

	// We use display-oriented hostnames in our file to mimick how a human user
	// would write it, so we need to search for and remove any key that
	// normalizes to our target hostname so we won't generate something invalid
	// when the existing entry is slightly different.
	for givenHost := range rawCredsMap {
		canonHost, err := svchost.ForComparison(givenHost)
		if err == nil && canonHost == host {
			delete(rawCredsMap, givenHost)
		}
	}

	// If we have a new object to store we'll write it in now. If the previous
	// object had the hostname written in a different way then this will
	// appear to change it into our canonical display form, with all the
	// letters in lowercase and other transforms from the Internationalized
	// Domain Names specification.
	if new != nil {
		toStore := new.ToStore()
		rawCredsMap[host.ForDisplay()] = ctyjson.SimpleJSONValue{
			Value: toStore,
		}
	}

	newSrc, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot serialize updated credentials file: %s", err)
	}

	// Now we'll write our new content over the top of the existing file.
	// Because we updated the data structure surgically here we should not
	// have disturbed the meaning of any other content in the file, but it
	// might have a different JSON layout than before.
	// We'll create a new file with a different name first and then rename
	// it over the old file in order to make the change as atomically as
	// the underlying OS/filesystem will allow.
	{
		dir, file := filepath.Split(filename)
		f, err := ioutil.TempFile(dir, file)
		if err != nil {
			return fmt.Errorf("cannot create temporary file to update credentials: %s", err)
		}
		tmpName := f.Name()
		moved := false
		defer func(f *os.File, name string) {
			// Always close our file, and remove it if it's still at its
			// temporary name. We're ignoring errors here because there's
			// nothing we can do about them anyway.
			f.Close()
			if !moved {
				os.Remove(name)
			}
		}(f, tmpName)

		// Credentials file should be readable only by its owner. (This may
		// not be effective on all platforms, but should at least work on
		// Unix-like targets and should be harmless elsewhere.)
		if err := f.Chmod(0600); err != nil {
			return fmt.Errorf("cannot set mode for temporary file %s: %s", tmpName, err)
		}

		_, err = f.Write(newSrc)
		if err != nil {
			return fmt.Errorf("cannot write to temporary file %s: %s", tmpName, err)
		}

		// Temporary file now replaces the original file, as atomically as
		// possible. (At the very least, we should not end up with a file
		// containing only a partial JSON object.)
		err = replaceFileAtomic(tmpName, filename)
		if err != nil {
			return fmt.Errorf("failed to replace %s with temporary file %s: %s", filename, tmpName, err)
		}
		moved = true
	}

	if new != nil {
		s.configured[host] = new.ToStore()
	} else {
		delete(s.configured, host)
	}

	return nil
}

// readHostsInCredentialsFile discovers which hosts have credentials configured
// in the credentials file specifically, as opposed to in any other CLI
// config file.
//
// If the credentials file isn't present or is unreadable for any reason then
// this returns an empty set, reflecting that effectively no credentials are
// stored there.
func readHostsInCredentialsFile(filename string) map[svchost.Hostname]struct{} {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil
	}

	var raw map[string]interface{}
	err = json.Unmarshal(src, &raw)
	if err != nil {
		return nil
	}

	rawCredsI, ok := raw["credentials"]
	if !ok {
		return nil
	}
	rawCredsMap, ok := rawCredsI.(map[string]interface{})
	if !ok {
		return nil
	}

	ret := make(map[svchost.Hostname]struct{})
	for givenHost := range rawCredsMap {
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			// We expect the config was already validated by the time we get
			// here, so we'll just ignore invalid hostnames.
			continue
		}
		ret[host] = struct{}{}
	}
	return ret
}

// ErrUnwritableHostCredentials is an error type that is returned when a caller
// tries to write credentials for a host that has existing credentials configured
// in a file that we cannot automatically update.
type ErrUnwritableHostCredentials svchost.Hostname

func (err ErrUnwritableHostCredentials) Error() string {
	return fmt.Sprintf("cannot change credentials for %s: existing manually-configured credentials in a CLI config file", svchost.Hostname(err).ForDisplay())
}

// Hostname returns the host that could not be written.
func (err ErrUnwritableHostCredentials) Hostname() svchost.Hostname {
	return svchost.Hostname(err)
}

// CredentialsLocation describes a type of storage used for the credentials
// for a particular hostname.
type CredentialsLocation rune

const (
	// CredentialsNotAvailable means that we know that there are no credential
	// available for the host.
	//
	// Note that CredentialsViaHelper might also lead to no credentials being
	// available, depending on how the helper answers when we request credentials
	// from it.
	CredentialsNotAvailable CredentialsLocation = 0

	// CredentialsInPrimaryFile means that there is already a credentials object
	// for the host in the credentials.tfrc.json file.
	CredentialsInPrimaryFile CredentialsLocation = 'P'

	// CredentialsInOtherFile means that there is already a credentials object
	// for the host in a CLI config file other than credentials.tfrc.json.
	CredentialsInOtherFile CredentialsLocation = 'O'

	// CredentialsViaHelper indicates that no statically-configured credentials
	// are available for the host but a helper program is available that may
	// or may not have credentials for the host.
	CredentialsViaHelper CredentialsLocation = 'H'
)

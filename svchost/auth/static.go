package auth

import (
	"fmt"

	"github.com/hashicorp/terraform/svchost"
)

// StaticCredentialsSource is a credentials source that retrieves credentials
// from the provided map. It returns nil if a requested hostname is not
// present in the map.
//
// The caller should not modify the given map after passing it to this function.
func StaticCredentialsSource(creds map[svchost.Hostname]map[string]interface{}) CredentialsSource {
	return staticCredentialsSource(creds)
}

type staticCredentialsSource map[svchost.Hostname]map[string]interface{}

func (s staticCredentialsSource) ForHost(host svchost.Hostname) (HostCredentials, error) {
	if s == nil {
		return nil, nil
	}

	if m, exists := s[host]; exists {
		return HostCredentialsFromMap(m), nil
	}

	return nil, nil
}

func (s staticCredentialsSource) StoreForHost(host svchost.Hostname, credentials HostCredentialsWritable) error {
	return fmt.Errorf("can't store new credentials in a static credentials source")
}

func (s staticCredentialsSource) ForgetForHost(host svchost.Hostname) error {
	return fmt.Errorf("can't discard credentials from a static credentials source")
}

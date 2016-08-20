package config

import (
	"time"

	vault "github.com/hashicorp/vault/api"
)

const (
	// DefaultVaultConnectRetryIntv is the retry interval between trying to
	// connect to Vault
	DefaultVaultConnectRetryIntv = 30 * time.Second
)

// VaultConfig contains the configuration information necessary to
// communicate with Vault in order to:
//
// - Renew Vault tokens/leases.
//
// - Pass a token for the Nomad Server to derive sub-tokens.
//
// - Create child tokens with policy subsets of the Server's token.
type VaultConfig struct {

	// Enabled enables or disables Vault support.
	Enabled bool `mapstructure:"enabled"`

	// Token is the Vault token given to Nomad such that it can
	// derive child tokens. Nomad will renew this token at half its lease
	// lifetime.
	Token string `mapstructure:"token"`

	// AllowUnauthenticated allows users to submit jobs requiring Vault tokens
	// without providing a Vault token proving they have access to these
	// policies.
	AllowUnauthenticated bool `mapstructure:"allow_unauthenticated"`

	// TaskTokenTTL is the TTL of the tokens created by Nomad Servers and used
	// by the client.  There should be a minimum time value such that the client
	// does not have to renew with Vault at a very high frequency
	TaskTokenTTL string `mapstructure:"task_token_ttl"`

	// Addr is the address of the local Vault agent. This should be a complete
	// URL such as "http://vault.example.com"
	Addr string `mapstructure:"address"`

	// ConnectionRetryIntv is the interval to wait before re-attempting to
	// connect to Vault.
	ConnectionRetryIntv time.Duration

	// TLSCaFile is the path to a PEM-encoded CA cert file to use to verify the
	// Vault server SSL certificate.
	TLSCaFile string `mapstructure:"tls_ca_file"`

	// TLSCaFile is the path to a directory of PEM-encoded CA cert files to
	// verify the Vault server SSL certificate.
	TLSCaPath string `mapstructure:"tls_ca_path"`

	// TLSCertFile is the path to the certificate for Vault communication
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile is the path to the private key for Vault communication
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// TLSSkipVerify enables or disables SSL verification
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`

	// TLSServerName, if set, is used to set the SNI host when connecting via TLS.
	TLSServerName string `mapstructure:"tls_server_name"`
}

// DefaultVaultConfig() returns the canonical defaults for the Nomad
// `vault` configuration.
func DefaultVaultConfig() *VaultConfig {
	return &VaultConfig{
		AllowUnauthenticated: false,
		Addr:                 "https://vault.service.consul:8200",
		ConnectionRetryIntv:  DefaultVaultConnectRetryIntv,
	}
}

// Merge merges two Vault configurations together.
func (a *VaultConfig) Merge(b *VaultConfig) *VaultConfig {
	result := *a

	if b.Token != "" {
		result.Token = b.Token
	}
	if b.TaskTokenTTL != "" {
		result.TaskTokenTTL = b.TaskTokenTTL
	}
	if b.Addr != "" {
		result.Addr = b.Addr
	}
	if b.ConnectionRetryIntv.Nanoseconds() != 0 {
		result.ConnectionRetryIntv = b.ConnectionRetryIntv
	}
	if b.TLSCaFile != "" {
		result.TLSCaFile = b.TLSCaFile
	}
	if b.TLSCaPath != "" {
		result.TLSCaPath = b.TLSCaPath
	}
	if b.TLSCertFile != "" {
		result.TLSCertFile = b.TLSCertFile
	}
	if b.TLSKeyFile != "" {
		result.TLSKeyFile = b.TLSKeyFile
	}
	if b.TLSServerName != "" {
		result.TLSServerName = b.TLSServerName
	}

	result.AllowUnauthenticated = b.AllowUnauthenticated
	result.TLSSkipVerify = b.TLSSkipVerify
	result.Enabled = b.Enabled
	return &result
}

// ApiConfig() returns a usable Vault config that can be passed directly to
// hashicorp/vault/api.
func (c *VaultConfig) ApiConfig() (*vault.Config, error) {
	conf := vault.DefaultConfig()
	tlsConf := &vault.TLSConfig{
		CACert:        c.TLSCaFile,
		CAPath:        c.TLSCaPath,
		ClientCert:    c.TLSCertFile,
		ClientKey:     c.TLSKeyFile,
		TLSServerName: c.TLSServerName,
		Insecure:      c.TLSSkipVerify,
	}
	if err := conf.ConfigureTLS(tlsConf); err != nil {
		return nil, err
	}

	conf.Address = c.Addr
	return conf, nil
}

// Copy returns a copy of this Vault config.
func (c *VaultConfig) Copy() *VaultConfig {
	if c == nil {
		return nil
	}

	nc := new(VaultConfig)
	*nc = *c
	return nc
}

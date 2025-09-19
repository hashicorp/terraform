package reattach

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/addrs"
)

const TF_REATTACH_PROVIDERS = "TF_REATTACH_PROVIDERS"

// ParseReattachProviders parses information used for reattaching to unmanaged providers out of a
// JSON-encoded environment variable (TF_REATTACH_PROVIDERS).
func ParseReattachProviders(in string) (map[addrs.Provider]*plugin.ReattachConfig, error) {
	os.Getenv(TF_REATTACH_PROVIDERS)
	unmanagedProviders := map[addrs.Provider]*plugin.ReattachConfig{}
	if in != "" {
		type reattachConfig struct {
			Protocol        string
			ProtocolVersion int
			Addr            struct {
				Network string
				String  string
			}
			Pid  int
			Test bool
		}
		var m map[string]reattachConfig
		err := json.Unmarshal([]byte(in), &m)
		if err != nil {
			return unmanagedProviders, fmt.Errorf("Invalid format for %s: %w", TF_REATTACH_PROVIDERS, err)
		}
		for p, c := range m {
			a, diags := addrs.ParseProviderSourceString(p)
			if diags.HasErrors() {
				return unmanagedProviders, fmt.Errorf("Error parsing %q as a provider address: %w", a, diags.Err())
			}
			var addr net.Addr
			switch c.Addr.Network {
			case "unix":
				addr, err = net.ResolveUnixAddr("unix", c.Addr.String)
				if err != nil {
					return unmanagedProviders, fmt.Errorf("Invalid unix socket path %q for %q: %w", c.Addr.String, p, err)
				}
			case "tcp":
				addr, err = net.ResolveTCPAddr("tcp", c.Addr.String)
				if err != nil {
					return unmanagedProviders, fmt.Errorf("Invalid TCP address %q for %q: %w", c.Addr.String, p, err)
				}
			default:
				return unmanagedProviders, fmt.Errorf("Unknown address type %q for %q", c.Addr.Network, p)
			}
			unmanagedProviders[a] = &plugin.ReattachConfig{
				Protocol:        plugin.Protocol(c.Protocol),
				ProtocolVersion: c.ProtocolVersion,
				Pid:             c.Pid,
				Test:            c.Test,
				Addr:            addr,
			}
		}
	}
	return unmanagedProviders, nil
}

// IsProviderReattached determines if a given provider is being supplied to Terraform via the TF_REATTACH_PROVIDERS
// environment variable.
func IsProviderReattached(provider addrs.Provider) (bool, error) {
	in := os.Getenv(TF_REATTACH_PROVIDERS)
	if in != "" {
		var m map[string]any // We are only going to use the unmarshaled provider names
		err := json.Unmarshal([]byte(in), &m)
		if err != nil {
			return false, fmt.Errorf("Invalid format for %s: %w", TF_REATTACH_PROVIDERS, err)
		}
		for p := range m {
			a, diags := addrs.ParseProviderSourceString(p)
			if diags.HasErrors() {
				return false, fmt.Errorf("Error parsing %q as a provider address: %w", a, diags.Err())
			}
			if a.Equals(provider) {
				return true, nil
			}
		}
	}
	return false, nil
}

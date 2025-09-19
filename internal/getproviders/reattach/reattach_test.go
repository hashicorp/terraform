package reattach

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-plugin"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
)

func Test_parseReattachProviders(t *testing.T) {
	cases := map[string]struct {
		reattachProviders string
		expectedOutput    map[addrs.Provider]*plugin.ReattachConfig
		expectErr         bool
	}{
		"simple parse - 1 provider": {
			reattachProviders: `{
				"test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
			expectedOutput: map[addrs.Provider]*plugin.ReattachConfig{
				tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "test"): func() *plugin.ReattachConfig {
					addr, err := net.ResolveUnixAddr("unix", "/var/folders/xx/abcde12345/T/plugin12345")
					if err != nil {
						t.Fatal(err)
					}
					return &plugin.ReattachConfig{
						Protocol:        plugin.Protocol("grpc"),
						ProtocolVersion: 6,
						Pid:             12345,
						Test:            true,
						Addr:            addr,
					}
				}(),
			},
		},
		"complex parse - 2 providers via different protocols etc": {
			reattachProviders: `{
				"test-grpc": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String": "/var/folders/xx/abcde12345/T/plugin12345"
					}
				},
				"test-netrpc": {
					"Protocol": "netrpc",
					"ProtocolVersion": 5,
					"Pid": 6789,
					"Test": false,
					"Addr": {
						"Network": "tcp",
						"String":"127.0.0.1:1337"
					}
				}
			}`,
			expectedOutput: map[addrs.Provider]*plugin.ReattachConfig{
				//test-grpc
				tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "test-grpc"): func() *plugin.ReattachConfig {
					addr, err := net.ResolveUnixAddr("unix", "/var/folders/xx/abcde12345/T/plugin12345")
					if err != nil {
						t.Fatal(err)
					}
					return &plugin.ReattachConfig{
						Protocol:        plugin.Protocol("grpc"),
						ProtocolVersion: 6,
						Pid:             12345,
						Test:            true,
						Addr:            addr,
					}
				}(),
				//test-netrpc
				tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "test-netrpc"): func() *plugin.ReattachConfig {
					addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:1337")
					if err != nil {
						t.Fatal(err)
					}
					return &plugin.ReattachConfig{
						Protocol:        plugin.Protocol("netrpc"),
						ProtocolVersion: 5,
						Pid:             6789,
						Test:            false,
						Addr:            addr,
					}
				}(),
			},
		},
		"can specify the providers host and namespace": {
			// The key here has host and namespace data, vs. just "test"
			reattachProviders: `{
				"example.com/my-org/test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
			expectedOutput: map[addrs.Provider]*plugin.ReattachConfig{
				tfaddr.NewProvider("example.com", "my-org", "test"): func() *plugin.ReattachConfig {
					addr, err := net.ResolveUnixAddr("unix", "/var/folders/xx/abcde12345/T/plugin12345")
					if err != nil {
						t.Fatal(err)
					}
					return &plugin.ReattachConfig{
						Protocol:        plugin.Protocol("grpc"),
						ProtocolVersion: 6,
						Pid:             12345,
						Test:            true,
						Addr:            addr,
					}
				}(),
			},
		},
		"error - bad JSON": {
			// Missing closing brace
			reattachProviders: `{
				"test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			`,
			expectErr: true,
		},
		"error - bad provider address": {
			reattachProviders: `{
				"bad provider addr": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
			expectErr: true,
		},
		"error - unrecognized protocol": {
			reattachProviders: `{
				"test": {
					"Protocol": "carrier-pigeon",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "pigeon",
						"String":"fly home little pigeon"
					}
				}
			}`,
			expectErr: true,
		},
		"error - unrecognized network": {
			reattachProviders: `{
				"test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "linkedin",
						"String":"http://www.linkedin.com/"
					}
				}
			}`,
			expectErr: true,
		},
		"error - bad tcp address": {
			// Addr.String has no port at the end
			reattachProviders: `{
				"test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "tcp",
						"String":"127.0.0.1"
					}
				}
			}`,
			expectErr: true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			t.Setenv("TF_REATTACH_PROVIDERS", tc.reattachProviders)

			output, err := ParseReattachProviders()
			if err != nil {
				if !tc.expectErr {
					t.Fatal(err)
				}
				// an expected error occurred
				return
			}
			if err == nil && tc.expectErr {
				t.Fatal("expected error but there was none")
			}
			if diff := cmp.Diff(output, tc.expectedOutput); diff != "" {
				t.Fatalf("expected diff:\n%s", diff)
			}
		})
	}
}

func Test_isProviderReattached(t *testing.T) {
	cases := map[string]struct {
		provider          addrs.Provider
		reattachProviders string
		expectedOutput    bool
	}{
		"identifies when a matching provider is present in TF_REATTACH_PROVIDERS": {
			// Note that the source in the TF_REATTACH_PROVIDERS value is just the provider name.
			// It'll be assumed to be under the default registry host and in the 'hashicorp' namespace.
			reattachProviders: `{
				"test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
			provider:       tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "hashicorp", "test"),
			expectedOutput: true,
		},
		"identifies when a provider doesn't have a match in TF_REATTACH_PROVIDERS": {
			// Note the mismatch on namespace
			reattachProviders: `{
				"hashicorp/test": {
					"Protocol": "grpc",
					"ProtocolVersion": 6,
					"Pid": 12345,
					"Test": true,
					"Addr": {
						"Network": "unix",
						"String":"/var/folders/xx/abcde12345/T/plugin12345"
					}
				}
			}`,
			provider:       tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "dadgarcorp", "test"),
			expectedOutput: false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			t.Setenv("TF_REATTACH_PROVIDERS", tc.reattachProviders)

			output, err := IsProviderReattached(tc.provider)
			if err != nil {
				t.Fatal(err)
			}
			if output != tc.expectedOutput {
				t.Fatalf("expected returned value to be %v, got %v", tc.expectedOutput, output)
			}
		})
	}
}

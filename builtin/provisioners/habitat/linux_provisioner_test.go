package habitat

import (
	"io"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const linuxDefaultSystemdUnitFileContents = `[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run --peer host1 --peer 1.2.3.4 --auto-update
Restart=on-failure
[Install]
WantedBy=default.target`

const linuxCustomSystemdUnitFileContents = `[Unit]
Description=Habitat Supervisor

[Service]
ExecStart=/bin/hab sup run --listen-ctl 192.168.0.1:8443 --listen-gossip 192.168.10.1:9443 --listen-http 192.168.20.1:8080 --peer host1 --peer host2 --peer 1.2.3.4 --peer 5.6.7.8 --peer foo.example.com
Restart=on-failure
Environment="HAB_SUP_GATEWAY_AUTH_TOKEN=ea7-beef"
Environment="HAB_AUTH_TOKEN=dead-beef"
[Install]
WantedBy=default.target`

func TestLinuxProvisioner_linuxInstallHabitat(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
	}{
		"Installation with sudo": {
			Config: map[string]interface{}{
				"version":     "0.79.1",
				"auto_update": true,
				"use_sudo":    true,
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl --silent -L0 https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh > install.sh'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'bash ./install.sh -v 0.79.1'":                                                                                          true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab install core/busybox'":                                                                                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab pkg exec core/busybox adduser -D -g \"\" hab'":                                                                     true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'rm -f install.sh'":                                                                                                     true,
			},
		},
		"Installation without sudo": {
			Config: map[string]interface{}{
				"version":     "0.79.1",
				"auto_update": true,
				"use_sudo":    false,
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'curl --silent -L0 https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh > install.sh'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'bash ./install.sh -v 0.79.1'":                                                                                          true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'hab install core/busybox'":                                                                                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'hab pkg exec core/busybox adduser -D -g \"\" hab'":                                                                     true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'rm -f install.sh'":                                                                                                     true,
			},
		},
		"Installation with Habitat license acceptance": {
			Config: map[string]interface{}{
				"version":        "0.81.0",
				"accept_license": true,
				"auto_update":    true,
				"use_sudo":       true,
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl --silent -L0 https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh > install.sh'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'bash ./install.sh -v 0.81.0'":                                                                                          true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab install core/busybox'":                                                                                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab pkg exec core/busybox adduser -D -g \"\" hab'":                                                                     true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'HAB_LICENSE=accept hab -V'":                                                                                            true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'rm -f install.sh'":                                                                                                     true,
			},
		},
	}

	ui := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	for k, tc := range cases {
		comm.Commands = tc.Commands

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		p.ui = ui
		p.comm = comm

		err = p.linuxInstallHabitat()
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestLinuxProvisioner_linuxStartHabitat(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Start systemd Habitat with sudo": {
			Config: map[string]interface{}{
				"version":      "0.79.1",
				"auto_update":  true,
				"use_sudo":     true,
				"service_name": "hab-sup",
				"peer":         "--peer host1",
				"peers":        []interface{}{"1.2.3.4"},
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab install core/hab-sup/0.79.1'":                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'systemctl enable hab-sup && systemctl start hab-sup'":         true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mv /tmp/hab-sup.service /etc/systemd/system/hab-sup.service'": true,
			},

			Uploads: map[string]string{
				"/tmp/hab-sup.service": linuxDefaultSystemdUnitFileContents,
			},
		},
		"Start systemd Habitat without sudo": {
			Config: map[string]interface{}{
				"version":      "0.79.1",
				"auto_update":  true,
				"use_sudo":     false,
				"service_name": "hab-sup",
				"peer":         "--peer host1",
				"peers":        []interface{}{"1.2.3.4"},
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'hab install core/hab-sup/0.79.1'":                     true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true /bin/bash -c 'systemctl enable hab-sup && systemctl start hab-sup'": true,
			},

			Uploads: map[string]string{
				"/etc/systemd/system/hab-sup.service": linuxDefaultSystemdUnitFileContents,
			},
		},
		"Start unmanaged Habitat with sudo": {
			Config: map[string]interface{}{
				"version":      "0.81.0",
				"license":      "accept-no-persist",
				"auto_update":  true,
				"use_sudo":     true,
				"service_type": "unmanaged",
				"peer":         "--peer host1",
				"peers":        []interface{}{"1.2.3.4"},
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab install core/hab-sup/0.81.0'":                                                                                true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mkdir -p /hab/sup/default && chmod o+w /hab/sup/default'":                                                        true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c '(setsid hab sup run --peer host1 --peer 1.2.3.4 --auto-update > /hab/sup/default/sup.log 2>&1 <&1 &) ; sleep 1'": true,
			},

			Uploads: map[string]string{
				"/etc/systemd/system/hab-sup.service": linuxDefaultSystemdUnitFileContents,
			},
		},
		"Start Habitat with custom config": {
			Config: map[string]interface{}{
				"version":            "0.79.1",
				"auto_update":        false,
				"use_sudo":           true,
				"service_name":       "hab-sup",
				"peer":               "--peer host1 --peer host2",
				"peers":              []interface{}{"1.2.3.4", "5.6.7.8", "foo.example.com"},
				"listen_ctl":         "192.168.0.1:8443",
				"listen_gossip":      "192.168.10.1:9443",
				"listen_http":        "192.168.20.1:8080",
				"builder_auth_token": "dead-beef",
				"gateway_auth_token": "ea7-beef",
				"ctl_secret":         "bad-beef",
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true HAB_AUTH_TOKEN=dead-beef sudo -E /bin/bash -c 'hab install core/hab-sup/0.79.1'":                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true HAB_AUTH_TOKEN=dead-beef sudo -E /bin/bash -c 'systemctl enable hab-sup && systemctl start hab-sup'":         true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true HAB_AUTH_TOKEN=dead-beef sudo -E /bin/bash -c 'mv /tmp/hab-sup.service /etc/systemd/system/hab-sup.service'": true,
			},

			Uploads: map[string]string{
				"/tmp/hab-sup.service": linuxCustomSystemdUnitFileContents,
			},
		},
	}

	ui := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	for k, tc := range cases {
		comm.Commands = tc.Commands
		comm.Uploads = tc.Uploads

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		p.ui = ui
		p.comm = comm

		err = p.linuxStartHabitat()
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestLinuxProvisioner_linuxUploadRingKey(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Commands map[string]bool
	}{
		"Upload ring key": {
			Config: map[string]interface{}{
				"version":          "0.79.1",
				"auto_update":      true,
				"use_sudo":         true,
				"service_name":     "hab-sup",
				"peers":            []interface{}{"1.2.3.4"},
				"ring_key":         "test-ring",
				"ring_key_content": "dead-beef",
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'echo -e \"dead-beef\" | hab ring key import'": true,
			},
		},
	}

	ui := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	for k, tc := range cases {
		comm.Commands = tc.Commands

		p, err := decodeConfig(
			schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
		)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		p.ui = ui
		p.comm = comm

		err = p.linuxUploadRingKey()
		if err != nil {
			t.Fatalf("Test %q failed: %v", k, err)
		}
	}
}

func TestLinuxProvisioner_linuxStartOrReconfigureHabitatService(t *testing.T) {
	cases := map[string]struct {
		Config   map[string]interface{}
		Stdout   io.Reader
		Commands map[string]bool
		Uploads  map[string]string
	}{
		"Start Habitat service with sudo": {
			Config: map[string]interface{}{
				"version":          "0.79.1",
				"auto_update":      false,
				"use_sudo":         true,
				"service_name":     "hab-sup",
				"peers":            []interface{}{"1.2.3.4"},
				"ring_key":         "test-ring",
				"ring_key_content": "dead-beef",
				"service": []interface{}{
					map[string]interface{}{
						"name":      "core/foo",
						"topology":  "standalone",
						"strategy":  "none",
						"channel":   "stable",
						"user_toml": "[config]\nlisten = 0.0.0.0:8080",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "prod",
							},
						},
					},
					map[string]interface{}{
						"name":      "core/bar",
						"topology":  "standalone",
						"strategy":  "rolling",
						"channel":   "staging",
						"group":     "prod",
						"user_toml": "[config]\nlisten = 0.0.0.0:443",
					},
				},
			},

			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl -s http://127.0.0.1:9631/services/foo/prod'":                                                                                                                                                                true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab pkg install core/foo  --channel stable'":                                                                                                                                                                     true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mkdir -p /hab/user/foo/config'":                                                                                                                                                                                  true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'chmod o-r /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml && mv /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml /hab/user/foo/config/user.toml'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab svc load core/foo  --topology standalone --strategy none --channel stable --bind backend:bar.prod'":                                                                                                          true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl -s http://127.0.0.1:9631/services/bar/prod'":                                                                                                                                                                true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab pkg install core/bar  --channel staging'":                                                                                                                                                                    true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mkdir -p /hab/user/bar/config'":                                                                                                                                                                                  true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'chmod o-r /tmp/user-6466ae3283ae1bd4737b00367bc676c6465b25682169ea5f7da222f3f078a5bf.toml && mv /tmp/user-6466ae3283ae1bd4737b00367bc676c6465b25682169ea5f7da222f3f078a5bf.toml /hab/user/bar/config/user.toml'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab svc load core/bar  --topology standalone --strategy rolling --channel staging --group prod'":                                                                                                                 true,
			},
			Uploads: map[string]string{
				"/tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml": "[config]\nlisten = 0.0.0.0:8080",
				"/tmp/user-6466ae3283ae1bd4737b00367bc676c6465b25682169ea5f7da222f3f078a5bf.toml": "[config]\nlisten = 0.0.0.0:443",
			},
		},
		"Reconfigure Habitat service that has diverged": {
			Config: map[string]interface{}{
				"version":          "0.79.1",
				"auto_update":      false,
				"use_sudo":         true,
				"service_name":     "hab-sup",
				"peers":            []interface{}{"1.2.3.4"},
				"ring_key":         "test-ring",
				"ring_key_content": "dead-beef",
				"service": []interface{}{
					map[string]interface{}{
						"name":      "core/foo",
						"topology":  "standalone",
						"strategy":  "none",
						"channel":   "stable",
						"user_toml": "[config]\nlisten = 0.0.0.0:8080",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl -s http://127.0.0.1:9631/services/foo/default'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab svc unload core/foo'":                            true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab pkg install core/foo  --channel stable'":         true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mkdir -p /hab/user/foo/config'":                      true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'chmod o-r /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml && mv /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml /hab/user/foo/config/user.toml'": true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'hab svc load core/foo  --topology standalone --strategy none --channel stable --bind backend:bar.default'":                                                                                                       true,
			},
			// Return a truncated version of http://<hab-http-api>/services/<service-name>/<group-name>
			// NOTE: the channel and update stategy intentionally diverges from
			// the specification to trigger a reconfigure
			Stdout: strings.NewReader(`{
  "binds": [
    "backend:bar.default"
  ],
  "bldr_url": "https://bldr.habitat.sh",
  "channel": "current",
  "service_group": "backend.default",
  "topology": "standalone",
  "update_strategy": "at-once"
}`),
			Uploads: map[string]string{
				"/tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml": "[config]\nlisten = 0.0.0.0:8080",
				"/tmp/user-6466ae3283ae1bd4737b00367bc676c6465b25682169ea5f7da222f3f078a5bf.toml": "[config]\nlisten = 0.0.0.0:443",
			},
		},
		"Don't reload an up-to-date service": {
			Config: map[string]interface{}{
				"version":          "0.79.1",
				"auto_update":      false,
				"use_sudo":         true,
				"service_name":     "hab-sup",
				"peers":            []interface{}{"1.2.3.4"},
				"ring_key":         "test-ring",
				"ring_key_content": "dead-beef",
				"service": []interface{}{
					map[string]interface{}{
						"name":      "core/foo",
						"topology":  "standalone",
						"strategy":  "at-once",
						"channel":   "stable",
						"user_toml": "[config]\nlisten = 0.0.0.0:8080",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Commands: map[string]bool{
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'curl -s http://127.0.0.1:9631/services/foo/default'":                                                                                                                                                             true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'mkdir -p /hab/user/foo/config'":                                                                                                                                                                                  true,
				"env HAB_NONINTERACTIVE=true HAB_NOCOLORING=true sudo -E /bin/bash -c 'chmod o-r /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml && mv /tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml /hab/user/foo/config/user.toml'": true,
			},
			// Return a truncated version of http://<hab-http-api>/services/<service-name>/<group-name>
			Stdout: strings.NewReader(`{
  "binds": [
    "backend:bar.default"
  ],
  "bldr_url": "https://bldr.habitat.sh",
  "channel": "stable",
  "service_group": "backend.default",
  "topology": "standalone",
  "update_strategy": "at-once"
}`),
			Uploads: map[string]string{
				"/tmp/user-a5b83ec1b302d109f41852ae17379f75c36dff9bc598aae76b6f7c9cd425fd76.toml": "[config]\nlisten = 0.0.0.0:8080",
				"/tmp/user-6466ae3283ae1bd4737b00367bc676c6465b25682169ea5f7da222f3f078a5bf.toml": "[config]\nlisten = 0.0.0.0:443",
			},
		},
	}

	ui := new(terraform.MockUIOutput)
	comm := new(communicator.MockCommunicator)

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			comm.Commands = tc.Commands
			comm.Uploads = tc.Uploads
			comm.Stdout = tc.Stdout

			p, err := decodeConfig(
				schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
			)
			if err != nil {
				t.Fatalf("Error: %v", err)
			}
			p.ui = ui
			p.comm = comm

			var errs []error
			for _, s := range p.Services {
				err = p.linuxStartOrReconfigureHabitatService(s)
				if err != nil {
					errs = append(errs, err)
				}
			}

			if len(errs) > 0 {
				for _, e := range errs {
					t.Logf("Test %q failed: %v", k, e)
					t.Fail()
				}
			}
		})
	}
}

func TestLinuxProvisioner_ServiceInfo_Equal(t *testing.T) {
	cases := map[string]struct {
		Config map[string]interface{}
		Info   *ServiceInfo
		Equal  bool
	}{
		"equal": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"group":    "default",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: true,
		},
		"equal with default bldr url and service group": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: true,
		},
		"divergent strategy": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "at-once",
						"topology": "standalone",
						"channel":  "stable",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: false,
		},
		"divergent topology": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "leader",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: false,
		},
		"divergent channel": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "current",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: false,
		},
		"divergent bldr URL": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"url":      "https://mybldr.myorg.org",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: false,
		},
		"divergent service group": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"url":      defaultBldrURL,
						"group":    "bar.default",
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "backend",
								"service": "bar",
								"group":   "default",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"backend:bar.default"},
			},
			Equal: false,
		},
		"divergent many binds": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "db",
								"service": "postgres",
								"group":   "staging",
							},
							map[string]interface{}{
								"alias":   "db",
								"service": "elasticsearch",
								"group":   "prod",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"db:elasticsearch.prod", "db:postgres.prod"},
			},
			Equal: false,
		},
		"equal many unsorted binds": {
			Config: map[string]interface{}{
				"service": []interface{}{
					map[string]interface{}{
						"name":     "core/foo",
						"strategy": "none",
						"topology": "standalone",
						"channel":  "stable",
						"url":      defaultBldrURL,
						"bind": []interface{}{
							map[string]interface{}{
								"alias":   "db",
								"service": "postgres",
								"group":   "prod",
							},
							map[string]interface{}{
								"alias":   "db",
								"service": "elasticsearch",
								"group":   "prod",
							},
						},
					},
				},
			},
			Info: &ServiceInfo{
				UpdateStrategy: "none",
				Topology:       "standalone",
				Channel:        "stable",
				BldrURL:        defaultBldrURL,
				ServiceGroup:   "foo.default",
				Binds:          []string{"db:elasticsearch.prod", "db:postgres.prod"},
			},
			Equal: true,
		},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			p, err := decodeConfig(
				schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, tc.Config),
			)
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

			equal, err := tc.Info.Equal(p.Services[0])
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

			if equal != tc.Equal {
				t.Fatalf("Error: expected %t, got %t", tc.Equal, equal)
			}
		})
	}
}

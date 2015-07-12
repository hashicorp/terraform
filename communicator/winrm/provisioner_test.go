package winrm

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestProvisioner_connInfo(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "winrm",
				"user":     "Administrator",
				"password": "supersecret",
				"host":     "127.0.0.1",
				"port":     "5985",
				"https":    "true",
				"timeout":  "30s",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "Administrator" {
		t.Fatalf("expected: %v: got: %v", "Administrator", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("expected: %v: got: %v", "supersecret", conf)
	}
	if conf.Host != "127.0.0.1" {
		t.Fatalf("expected: %v: got: %v", "127.0.0.1", conf)
	}
	if conf.Port != 5985 {
		t.Fatalf("expected: %v: got: %v", 5985, conf)
	}
	if conf.HTTPS != true {
		t.Fatalf("expected: %v: got: %v", true, conf)
	}
	if conf.Timeout != "30s" {
		t.Fatalf("expected: %v: got: %v", "30s", conf)
	}
	if conf.ScriptPath != DefaultScriptPath {
		t.Fatalf("expected: %v: got: %v", DefaultScriptPath, conf)
	}
}

func TestProvisioner_formatDuration(t *testing.T) {
	cases := map[string]struct {
		InstanceState *terraform.InstanceState
		Result        string
	}{
		"testSeconds": {
			InstanceState: &terraform.InstanceState{
				Ephemeral: terraform.EphemeralState{
					ConnInfo: map[string]string{
						"timeout": "90s",
					},
				},
			},

			Result: "PT1M30S",
		},
		"testMinutes": {
			InstanceState: &terraform.InstanceState{
				Ephemeral: terraform.EphemeralState{
					ConnInfo: map[string]string{
						"timeout": "5m",
					},
				},
			},

			Result: "PT5M",
		},
		"testHours": {
			InstanceState: &terraform.InstanceState{
				Ephemeral: terraform.EphemeralState{
					ConnInfo: map[string]string{
						"timeout": "1h",
					},
				},
			},

			Result: "PT1H",
		},
	}

	for name, tc := range cases {
		conf, err := parseConnectionInfo(tc.InstanceState)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		result := formatDuration(conf.TimeoutVal)
		if result != tc.Result {
			t.Fatalf("%s: expected: %s got: %s", name, tc.Result, result)
		}
	}
}

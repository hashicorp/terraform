package ssh

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestSSHConfig_RemotePath(t *testing.T) {
	cases := []struct {
		Input   string
		Pattern string
	}{
		{
			"/tmp/script.sh",
			`^/tmp/script\.sh$`,
		},
		{
			"/tmp/script_%RAND%.sh",
			`^/tmp/script_(\d+)\.sh$`,
		},
	}

	for _, tc := range cases {
		config := &SSHConfig{ScriptPath: tc.Input}
		output := config.RemotePath()

		match, err := regexp.Match(tc.Pattern, []byte(output))
		if err != nil {
			t.Fatalf("bad: %s\n\nerr: %s", tc.Input, err)
		}
		if !match {
			t.Fatalf("bad: %s\n\n%s", tc.Input, output)
		}
	}
}

func TestResourceProvider_verifySSH(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "telnet",
			},
		},
	}
	if err := VerifySSH(r); err == nil {
		t.Fatalf("expected error with telnet")
	}
	r.Ephemeral.ConnInfo["type"] = "ssh"
	if err := VerifySSH(r); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestResourceProvider_sshConfig(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "ssh",
				"user":     "root",
				"password": "supersecret",
				"key_file": "/my/key/file.pem",
				"host":     "127.0.0.1",
				"port":     "22",
				"timeout":  "30s",
			},
		},
	}

	conf, err := ParseSSHConfig(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.KeyFile != "/my/key/file.pem" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Host != "127.0.0.1" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Port != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Timeout != "30s" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ScriptPath != DefaultScriptPath {
		t.Fatalf("bad: %v", conf)
	}
}

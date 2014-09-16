package ssh

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

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

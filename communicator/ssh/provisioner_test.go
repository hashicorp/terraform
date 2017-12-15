package ssh

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestProvisioner_connInfo(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "127.0.0.1",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "127.0.1.1",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.PrivateKey != "someprivatekeycontents" {
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
	if conf.BastionHost != "127.0.1.1" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPort != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionUser != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPassword != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPrivateKey != "someprivatekeycontents" {
		t.Fatalf("bad: %v", conf)
	}
}

func TestProvisioner_connInfo_bastionList(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "127.0.0.1",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "127.0.1.1,127.0.1.2",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.PrivateKey != "someprivatekeycontents" {
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
	if conf.BastionHost != "127.0.1.1,127.0.1.2" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPort != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPortList != "22,22" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionUser != "root,root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPassword != "supersecret,supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPrivateKey != "someprivatekeycontents,someprivatekeycontents" {
		t.Fatalf("bad: %v", conf)
	}
}

func TestProvisioner_connInfo_transparentBastion(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "127.0.0.1",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "127.0.1.1",
			},
		},
	}
	os.Setenv("TRANSPARENT_BASTIONHOST", "127.0.1.2")
	os.Setenv("TRANSPARENT_BASTIONUSER", "wheel")
	os.Setenv("TRANSPARENT_BASTIONPASSWORD", "extremesecret")
	os.Setenv("TRANSPARENT_BASTIONPRIVATEKEY", "anotherprivatekeycontents")

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.PrivateKey != "someprivatekeycontents" {
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
	if conf.BastionHost != "127.0.1.2,127.0.1.1" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPort != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPortList != "22,22" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionUser != "wheel,root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPassword != "extremesecret,supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPrivateKey != "anotherprivatekeycontents,someprivatekeycontents" {
		t.Fatalf("bad: %v", conf)
	}
	os.Unsetenv("TRANSPARENT_BASTIONHOST")
	os.Unsetenv("TRANSPARENT_BASTIONUSER")
	os.Unsetenv("TRANSPARENT_BASTIONPASSWORD")
	os.Unsetenv("TRANSPARENT_BASTIONPRIVATEKEY")
}

func TestProvisioner_connInfoIpv6(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "::1",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "::1",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.Host != "[::1]" {
		t.Fatalf("bad: %v", conf)
	}

	if conf.BastionHost != "[::1]" {
		t.Fatalf("bad %v", conf)
	}
}

func TestProvisioner_connInfoHostname(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "example.com",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "example.com",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.Host != "example.com" {
		t.Fatalf("bad: %v", conf)
	}

	if conf.BastionHost != "example.com" {
		t.Fatalf("bad %v", conf)
	}
}

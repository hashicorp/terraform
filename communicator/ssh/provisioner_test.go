package ssh

import (
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

func TestProvisioner_connInfoRemoteForward(t *testing.T) {
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

				"bastion_host":   "example.com",
				"remote_forward": "127.0.1.1:8080:127.0.0.1:80,8081:host:8081",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(conf.RemoteForwardVal) != 2 {
		t.Fatalf("bad: %v", conf)
	}

	if conf.RemoteForwardVal[0].BindPort != 8080 {
		t.Fatalf("bad: %v", conf)
	}

	if conf.RemoteForwardVal[0].BindHost != "127.0.1.1" {
		t.Fatalf("bad %v", conf)
	}

	if conf.RemoteForwardVal[0].Port != 80 {
		t.Fatalf("bad: %v", conf)
	}

	if conf.RemoteForwardVal[0].Host != "127.0.0.1" {
		t.Fatalf("bad %v", conf)
	}

	if conf.RemoteForwardVal[1].BindPort != 8081 {
		t.Fatalf("bad: %v", conf)
	}

	if conf.RemoteForwardVal[1].BindHost != "localhost" {
		t.Fatalf("bad %v", conf)
	}

	if conf.RemoteForwardVal[1].Port != 8081 {
		t.Fatalf("bad: %v", conf)
	}

	if conf.RemoteForwardVal[1].Host != "host" {
		t.Fatalf("bad %v", conf)
	}

}

func TestProvisioner_connInfoRemoteForwardNone(t *testing.T) {
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
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(conf.RemoteForwardVal) != 0 {
		t.Fatalf("bad: %v", conf)
	}
}

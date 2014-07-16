package remoteexec

import (
	"bytes"
	"io"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = new(ResourceProvisioner)
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"inline": "echo foo",
	})
	p := new(ResourceProvisioner)
	warn, errs := p.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"invalid": "nope",
	})
	p := new(ResourceProvisioner)
	warn, errs := p.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvider_verifySSH(t *testing.T) {
	p := new(ResourceProvisioner)
	r := &terraform.ResourceState{
		ConnInfo: map[string]string{
			"type": "telnet",
		},
	}
	if err := p.verifySSH(r); err == nil {
		t.Fatalf("expected error with telnet")
	}
	r.ConnInfo["type"] = "ssh"
	if err := p.verifySSH(r); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestResourceProvider_sshConfig(t *testing.T) {
	p := new(ResourceProvisioner)
	r := &terraform.ResourceState{
		ConnInfo: map[string]string{
			"type":     "ssh",
			"user":     "root",
			"password": "supersecret",
			"key_file": "/my/key/file.pem",
			"host":     "127.0.0.1",
			"port":     "22",
			"timeout":  "30s",
		},
	}

	conf, err := p.sshConfig(r)
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

func TestResourceProvider_generateScript(t *testing.T) {
	p := new(ResourceProvisioner)
	conf := testConfig(t, map[string]interface{}{
		"inline": []string{
			"cd /tmp",
			"wget http://foobar",
			"exit 0",
		},
	})
	out, err := p.generateScript(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != expectedScriptOut {
		t.Fatalf("bad: %v", out)
	}
}

var expectedScriptOut = `#!/bin/sh
cd /tmp
wget http://foobar
exit 0
`

func TestResourceProvider_CollectScripts_inline(t *testing.T) {
	p := new(ResourceProvisioner)
	conf := testConfig(t, map[string]interface{}{
		"inline": []string{
			"cd /tmp",
			"wget http://foobar",
			"exit 0",
		},
	})

	scripts, err := p.collectScripts(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("bad: %v", scripts)
	}

	var out bytes.Buffer
	_, err = io.Copy(&out, scripts[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(out.Bytes()) != expectedScriptOut {
		t.Fatalf("bad: %v", out.Bytes())
	}
}

func TestResourceProvider_CollectScripts_script(t *testing.T) {
	p := new(ResourceProvisioner)
	conf := testConfig(t, map[string]interface{}{
		"script": "test-fixtures/script1.sh",
	})

	scripts, err := p.collectScripts(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("bad: %v", scripts)
	}

	var out bytes.Buffer
	_, err = io.Copy(&out, scripts[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if string(out.Bytes()) != expectedScriptOut {
		t.Fatalf("bad: %v", out.Bytes())
	}
}

func TestResourceProvider_CollectScripts_scripts(t *testing.T) {
	p := new(ResourceProvisioner)
	conf := testConfig(t, map[string]interface{}{
		"scripts": []interface{}{
			"test-fixtures/script1.sh",
			"test-fixtures/script1.sh",
			"test-fixtures/script1.sh",
		},
	})

	scripts, err := p.collectScripts(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 3 {
		t.Fatalf("bad: %v", scripts)
	}

	for idx := range scripts {
		var out bytes.Buffer
		_, err = io.Copy(&out, scripts[idx])
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if string(out.Bytes()) != expectedScriptOut {
			t.Fatalf("bad: %v", out.Bytes())
		}
	}
}

func testConfig(
	t *testing.T,
	c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}

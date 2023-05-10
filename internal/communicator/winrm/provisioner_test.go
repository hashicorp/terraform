// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package winrm

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestProvisioner_defaultHTTPSPort(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"type":     cty.StringVal("winrm"),
		"user":     cty.StringVal("Administrator"),
		"password": cty.StringVal("supersecret"),
		"host":     cty.StringVal("127.0.0.1"),
		"https":    cty.True,
	})

	conf, err := parseConnectionInfo(v)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if conf.Port != 5986 {
		t.Fatalf("expected: %v: got: %v", 5986, conf)
	}
	if conf.HTTPS != true {
		t.Fatalf("expected: %v: got: %v", true, conf)
	}
}

func TestProvisioner_connInfo(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"type":     cty.StringVal("winrm"),
		"user":     cty.StringVal("Administrator"),
		"password": cty.StringVal("supersecret"),
		"host":     cty.StringVal("127.0.0.1"),
		"port":     cty.StringVal("5985"),
		"https":    cty.True,
		"use_ntlm": cty.True,
		"timeout":  cty.StringVal("30s"),
	})

	conf, err := parseConnectionInfo(v)
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
	if conf.NTLM != true {
		t.Fatalf("expected: %v: got: %v", true, conf)
	}
	if conf.Timeout != "30s" {
		t.Fatalf("expected: %v: got: %v", "30s", conf)
	}
	if conf.ScriptPath != DefaultScriptPath {
		t.Fatalf("expected: %v: got: %v", DefaultScriptPath, conf)
	}
}

func TestProvisioner_connInfoCACert(t *testing.T) {
	caCert := `
-----BEGIN CERTIFICATE-----
MIIDBjCCAe4CCQCGWwBmOiHQdTANBgkqhkiG9w0BAQUFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTE2MDYyMTE2MzM0MVoXDTE3MDYyMTE2MzM0MVowRTELMAkG
A1UEBhMCQVUxEzARBgNVBAgTClNvbWUtU3RhdGUxITAfBgNVBAoTGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AL+LFlsCJG5txZp4yuu+lQnuUrgBXRG+irQqcTXlV91Bp5hpmRIyhnGCtWxxDBUL
xrh4WN3VV/0jDzKT976oLgOy3hj56Cdqf+JlZ1qgMN5bHB3mm3aVWnrnsLbBsfwZ
SEbk3Kht/cE1nK2toNVW+rznS3m+eoV3Zn/DUNwGlZr42hGNs6ETn2jURY78ETqR
mW47xvjf86eIo7vULHJaY6xyarPqkL8DZazOmvY06hUGvGwGBny7gugfXqDG+I8n
cPBsGJGSAmHmVV8o0RCB9UjY+TvSMQRpEDoVlvyrGuglsD8to/4+7UcsuDGlRYN6
jmIOC37mOi/jwRfWL1YUa4MCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEAPDxTH0oQ
JjKXoJgkmQxurB81RfnK/NrswJVzWbOv6ejcbhwh+/ZgJTMc15BrYcxU6vUW1V/i
Z7APU0qJ0icECACML+a2fRI7YdLCTiPIOmY66HY8MZHAn3dGjU5TeiUflC0n0zkP
mxKJe43kcYLNDItbfvUDo/GoxTXrC3EFVZyU0RhFzoVJdODlTHXMVFCzcbQEBrBJ
xKdShCEc8nFMneZcGFeEU488ntZoWzzms8/QpYrKa5S0Sd7umEU2Kwu4HTkvUFg/
CqDUFjhydXxYRsxXBBrEiLOE5BdtJR1sH/QHxIJe23C9iHI2nS1NbLziNEApLwC4
GnSud83VUo9G9w==
-----END CERTIFICATE-----
`
	v := cty.ObjectVal(map[string]cty.Value{
		"type":     cty.StringVal("winrm"),
		"user":     cty.StringVal("Administrator"),
		"password": cty.StringVal("supersecret"),
		"host":     cty.StringVal("127.0.0.1"),
		"port":     cty.StringVal("5985"),
		"https":    cty.True,
		"timeout":  cty.StringVal("30s"),
		"cacert":   cty.StringVal(caCert),
	})

	conf, err := parseConnectionInfo(v)
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
	if conf.CACert != caCert {
		t.Fatalf("expected: %v: got: %v", caCert, conf.CACert)
	}
}

func TestProvisioner_connInfoIpv6(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"type":     cty.StringVal("winrm"),
		"user":     cty.StringVal("Administrator"),
		"password": cty.StringVal("supersecret"),
		"host":     cty.StringVal("::1"),
		"port":     cty.StringVal("5985"),
		"https":    cty.True,
		"timeout":  cty.StringVal("30s"),
	})

	conf, err := parseConnectionInfo(v)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "Administrator" {
		t.Fatalf("expected: %v: got: %v", "Administrator", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("expected: %v: got: %v", "supersecret", conf)
	}
	if conf.Host != "[::1]" {
		t.Fatalf("expected: %v: got: %v", "[::1]", conf)
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

func TestProvisioner_connInfoHostname(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"type":     cty.StringVal("winrm"),
		"user":     cty.StringVal("Administrator"),
		"password": cty.StringVal("supersecret"),
		"host":     cty.StringVal("example.com"),
		"port":     cty.StringVal("5985"),
		"https":    cty.True,
		"timeout":  cty.StringVal("30s"),
	})

	conf, err := parseConnectionInfo(v)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "Administrator" {
		t.Fatalf("expected: %v: got: %v", "Administrator", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("expected: %v: got: %v", "supersecret", conf)
	}
	if conf.Host != "example.com" {
		t.Fatalf("expected: %v: got: %v", "example.com", conf)
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
		Config map[string]cty.Value
		Result string
	}{
		"testSeconds": {
			Config: map[string]cty.Value{
				"timeout": cty.StringVal("90s"),
			},

			Result: "PT1M30S",
		},
		"testMinutes": {
			Config: map[string]cty.Value{
				"timeout": cty.StringVal("5m"),
			},

			Result: "PT5M",
		},
		"testHours": {
			Config: map[string]cty.Value{
				"timeout": cty.StringVal("1h"),
			},

			Result: "PT1H",
		},
	}

	for name, tc := range cases {
		// host is required in the schema
		tc.Config["host"] = cty.StringVal("")

		conf, err := parseConnectionInfo(cty.ObjectVal(tc.Config))
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		result := formatDuration(conf.TimeoutVal)
		if result != tc.Result {
			t.Fatalf("%s: expected: %s got: %s", name, tc.Result, result)
		}
	}
}

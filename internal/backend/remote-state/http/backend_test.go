// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package http

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/zclconf/go-cty/cty"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestHTTPClientFactory(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("http://127.0.0.1:8888/foo"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "http://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
		"retry_max":      cty.StringVal("999"),
		"retry_wait_min": cty.StringVal("15"),
		"retry_wait_max": cty.StringVal("150"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
	if client.Client.RetryMax != 999 {
		t.Fatalf("Expected retry_max \"%d\", got \"%d\"", 999, client.Client.RetryMax)
	}
	if client.Client.RetryWaitMin != 15*time.Second {
		t.Fatalf("Expected retry_wait_min \"%s\", got \"%s\"", 15*time.Second, client.Client.RetryWaitMin)
	}
	if client.Client.RetryWaitMax != 150*time.Second {
		t.Fatalf("Expected retry_wait_max \"%s\", got \"%s\"", 150*time.Second, client.Client.RetryWaitMax)
	}
}

func TestHTTPClientFactoryWithEnv(t *testing.T) {
	// env
	conf := map[string]string{
		"address":        "http://127.0.0.1:8888/foo",
		"update_method":  "BLAH",
		"lock_address":   "http://127.0.0.1:8888/bar",
		"lock_method":    "BLIP",
		"unlock_address": "http://127.0.0.1:8888/baz",
		"unlock_method":  "BLOOP",
		"username":       "user",
		"password":       "pass",
		"retry_max":      "999",
		"retry_wait_min": "15",
		"retry_wait_max": "150",
	}

	defer testWithEnv(t, "TF_HTTP_ADDRESS", conf["address"])()
	defer testWithEnv(t, "TF_HTTP_UPDATE_METHOD", conf["update_method"])()
	defer testWithEnv(t, "TF_HTTP_LOCK_ADDRESS", conf["lock_address"])()
	defer testWithEnv(t, "TF_HTTP_UNLOCK_ADDRESS", conf["unlock_address"])()
	defer testWithEnv(t, "TF_HTTP_LOCK_METHOD", conf["lock_method"])()
	defer testWithEnv(t, "TF_HTTP_UNLOCK_METHOD", conf["unlock_method"])()
	defer testWithEnv(t, "TF_HTTP_USERNAME", conf["username"])()
	defer testWithEnv(t, "TF_HTTP_PASSWORD", conf["password"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_MAX", conf["retry_max"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_WAIT_MIN", conf["retry_wait_min"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_WAIT_MAX", conf["retry_wait_max"])()

	b := backend.TestBackendConfig(t, New(), nil).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, EnvDefaultFunc")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"] || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"], client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"] || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"], client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
	if client.Client.RetryMax != 999 {
		t.Fatalf("Expected retry_max \"%d\", got \"%d\"", 999, client.Client.RetryMax)
	}
	if client.Client.RetryWaitMin != 15*time.Second {
		t.Fatalf("Expected retry_wait_min \"%s\", got \"%s\"", 15*time.Second, client.Client.RetryWaitMin)
	}
	if client.Client.RetryWaitMax != 150*time.Second {
		t.Fatalf("Expected retry_wait_max \"%s\", got \"%s\"", 150*time.Second, client.Client.RetryWaitMax)
	}
}

// testWithEnv sets an environment variable and returns a deferable func to clean up
func testWithEnv(t *testing.T, key string, value string) func() {
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("err: %v", err)
	}

	return func() {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}
func mustMakeCA(t *testing.T, cn string) (certPEM []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("serial: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("pem.Encode: %v", err)
	}

	return buf.Bytes()
}

func mustSubjects(t *testing.T, pool *x509.CertPool) [][]byte {
	t.Helper()

	if pool == nil {
		t.Fatalf("expected non-nil CertPool")
	}

	return pool.Subjects()
}

func subjectPresent(subjects [][]byte, want []byte) bool {
	for _, s := range subjects {
		if bytes.Equal(s, want) {
			return true
		}
	}
	return false
}

func TestConfigureTLS_CAFileOnly(t *testing.T) {
	b := &Backend{}

	ca1 := mustMakeCA(t, "ca-one")

	dir := t.TempDir()
	caPath := dir + "/ca.pem"
	if err := os.WriteFile(caPath, ca1, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = &http.Transport{}

	cfg := cty.ObjectVal(map[string]cty.Value{
		"ca_file":                   cty.StringVal(caPath),
		"skip_cert_verification":    cty.False,
		"client_ca_certificate_pem": cty.NullVal(cty.String),
		"client_certificate_pem":    cty.NullVal(cty.String),
		"client_private_key_pem":    cty.NullVal(cty.String),
	})

	if err := b.configureTLS(client, cfg); err != nil {
		t.Fatalf("configureTLS: %v", err)
	}

	tlsCfg := client.HTTPClient.Transport.(*http.Transport).TLSClientConfig
	if tlsCfg == nil || tlsCfg.RootCAs == nil {
		t.Fatalf("expected RootCAs to be configured")
	}

	block, _ := pem.Decode(ca1)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}

	subjects := mustSubjects(t, tlsCfg.RootCAs)
	if !subjectPresent(subjects, cert.RawSubject) {
		t.Fatalf("expected CA subject from ca_file to be present in RootCAs")
	}
}

func TestConfigureTLS_CAFileAndInlinePEMMerge(t *testing.T) {
	b := &Backend{}

	caFilePEM := mustMakeCA(t, "ca-file")
	caInlinePEM := mustMakeCA(t, "ca-inline")

	dir := t.TempDir()
	caPath := dir + "/ca.pem"
	if err := os.WriteFile(caPath, caFilePEM, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = &http.Transport{}

	cfg := cty.ObjectVal(map[string]cty.Value{
		"ca_file":                   cty.StringVal(caPath),
		"client_ca_certificate_pem": cty.StringVal(string(caInlinePEM)),
		"client_certificate_pem":    cty.NullVal(cty.String),
		"client_private_key_pem":    cty.NullVal(cty.String),
		"skip_cert_verification":    cty.False,
	})

	if err := b.configureTLS(client, cfg); err != nil {
		t.Fatalf("configureTLS: %v", err)
	}

	tlsCfg := client.HTTPClient.Transport.(*http.Transport).TLSClientConfig
	if tlsCfg == nil || tlsCfg.RootCAs == nil {
		t.Fatalf("expected RootCAs to be configured")
	}

	block1, _ := pem.Decode(caFilePEM)
	cert1, err := x509.ParseCertificate(block1.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate file CA: %v", err)
	}

	block2, _ := pem.Decode(caInlinePEM)
	cert2, err := x509.ParseCertificate(block2.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate inline CA: %v", err)
	}

	subjects := mustSubjects(t, tlsCfg.RootCAs)

	if !subjectPresent(subjects, cert1.RawSubject) {
		t.Fatalf("expected file CA subject to be present in RootCAs")
	}
	if !subjectPresent(subjects, cert2.RawSubject) {
		t.Fatalf("expected inline CA subject to be present in RootCAs")
	}
}

func TestConfigureTLS_CAFileNotFound(t *testing.T) {
	b := &Backend{}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = &http.Transport{}

	cfg := cty.ObjectVal(map[string]cty.Value{
		"ca_file":                   cty.StringVal("/path/does/not/exist/ca.pem"),
		"client_ca_certificate_pem": cty.NullVal(cty.String),
		"client_certificate_pem":    cty.NullVal(cty.String),
		"client_private_key_pem":    cty.NullVal(cty.String),
		"skip_cert_verification":    cty.False,
	})

	if err := b.configureTLS(client, cfg); err == nil {
		t.Fatalf("expected error for missing ca_file, got nil")
	}
}

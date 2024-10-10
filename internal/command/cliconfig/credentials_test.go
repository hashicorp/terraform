// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cliconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"
)

func TestCredentialsForHost(t *testing.T) {
	credSrc := &CredentialsSource{
		configured: map[svchost.Hostname]cty.Value{
			"configured.example.com": cty.ObjectVal(map[string]cty.Value{
				"token": cty.StringVal("configured"),
			}),
			"unused.example.com": cty.ObjectVal(map[string]cty.Value{
				"token": cty.StringVal("incorrectly-configured"),
			}),
		},

		// We'll use a static source to stand in for what would normally be
		// a credentials helper program, since we're only testing the logic
		// for choosing when to delegate to the helper here. The logic for
		// interacting with a helper program is tested in the svcauth package.
		helper: svcauth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
			"from-helper.example.com": {
				"token": "from-helper",
			},

			// This should be shadowed by the "configured" entry with the same
			// hostname above.
			"configured.example.com": {
				"token": "incorrectly-from-helper",
			},
		}),
		helperType: "fake",
	}

	testReqAuthHeader := func(t *testing.T, creds svcauth.HostCredentials) string {
		t.Helper()

		if creds == nil {
			return ""
		}

		req, err := http.NewRequest("GET", "http://example.com/", nil)
		if err != nil {
			t.Fatalf("cannot construct HTTP request: %s", err)
		}
		creds.PrepareRequest(req)
		return req.Header.Get("Authorization")
	}

	t.Run("configured", func(t *testing.T) {
		creds, err := credSrc.ForHost(svchost.Hostname("configured.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if got, want := testReqAuthHeader(t, creds), "Bearer configured"; got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("from helper", func(t *testing.T) {
		creds, err := credSrc.ForHost(svchost.Hostname("from-helper.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if got, want := testReqAuthHeader(t, creds), "Bearer from-helper"; got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("not available", func(t *testing.T) {
		creds, err := credSrc.ForHost(svchost.Hostname("unavailable.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if got, want := testReqAuthHeader(t, creds), ""; got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("set in environment", func(t *testing.T) {
		envName := "TF_TOKEN_configured_example_com"
		t.Cleanup(func() {
			os.Unsetenv(envName)
		})

		expectedToken := "configured-by-env"
		os.Setenv(envName, expectedToken)

		creds, err := credSrc.ForHost(svchost.Hostname("configured.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		if got := creds.Token(); got != expectedToken {
			t.Errorf("wrong result\ngot: %s\nwant: %s", got, expectedToken)
		}
	})

	t.Run("punycode name set in environment", func(t *testing.T) {
		envName := "TF_TOKEN_env_xn--eckwd4c7cu47r2wf_com"
		t.Cleanup(func() {
			os.Unsetenv(envName)
		})

		expectedToken := "configured-by-env"
		os.Setenv(envName, expectedToken)

		hostname, _ := svchost.ForComparison("env.ドメイン名例.com")
		creds, err := credSrc.ForHost(hostname)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		if got := creds.Token(); got != expectedToken {
			t.Errorf("wrong result\ngot: %s\nwant: %s", got, expectedToken)
		}
	})

	t.Run("hyphens can be encoded as double underscores", func(t *testing.T) {
		envName := "TF_TOKEN_env_xn____caf__dma_fr"
		expectedToken := "configured-by-fallback"
		t.Cleanup(func() {
			os.Unsetenv(envName)
		})

		os.Setenv(envName, expectedToken)

		hostname, _ := svchost.ForComparison("env.café.fr")
		creds, err := credSrc.ForHost(hostname)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		if got := creds.Token(); got != expectedToken {
			t.Errorf("wrong result\ngot: %s\nwant: %s", got, expectedToken)
		}
	})

	t.Run("periods are ok", func(t *testing.T) {
		envName := "TF_TOKEN_configured.example.com"
		expectedToken := "configured-by-env"
		t.Cleanup(func() {
			os.Unsetenv(envName)
		})

		os.Setenv(envName, expectedToken)

		hostname, _ := svchost.ForComparison("configured.example.com")
		creds, err := credSrc.ForHost(hostname)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		if got := creds.Token(); got != expectedToken {
			t.Errorf("wrong result\ngot: %s\nwant: %s", got, expectedToken)
		}
	})

	t.Run("casing is insensitive", func(t *testing.T) {
		envName := "TF_TOKEN_CONFIGUREDUPPERCASE_EXAMPLE_COM"
		expectedToken := "configured-by-env"

		os.Setenv(envName, expectedToken)
		t.Cleanup(func() {
			os.Unsetenv(envName)
		})

		hostname, _ := svchost.ForComparison("configureduppercase.example.com")
		creds, err := credSrc.ForHost(hostname)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		if got := creds.Token(); got != expectedToken {
			t.Errorf("wrong result\ngot: %s\nwant: %s", got, expectedToken)
		}
	})
}

func TestCredentialsStoreForget(t *testing.T) {
	d := t.TempDir()

	mockCredsFilename := filepath.Join(d, "credentials.tfrc.json")

	cfg := &Config{
		// This simulates there being a credentials block manually configured
		// in some file _other than_ credentials.tfrc.json.
		Credentials: map[string]map[string]interface{}{
			"manually-configured.example.com": {
				"token": "manually-configured",
			},
		},
	}

	// We'll initially use a credentials source with no credentials helper at
	// all, and thus with credentials stored in the credentials file.
	credSrc := cfg.credentialsSource(
		"", nil,
		mockCredsFilename,
	)

	testReqAuthHeader := func(t *testing.T, creds svcauth.HostCredentials) string {
		t.Helper()

		if creds == nil {
			return ""
		}

		req, err := http.NewRequest("GET", "http://example.com/", nil)
		if err != nil {
			t.Fatalf("cannot construct HTTP request: %s", err)
		}
		creds.PrepareRequest(req)
		return req.Header.Get("Authorization")
	}

	// Because these store/forget calls have side-effects, we'll bail out with
	// t.Fatal (or equivalent) as soon as anything unexpected happens.
	// Otherwise downstream tests might fail in confusing ways.
	{
		err := credSrc.StoreForHost(
			svchost.Hostname("manually-configured.example.com"),
			svcauth.HostCredentialsToken("not-manually-configured"),
		)
		if err == nil {
			t.Fatalf("successfully stored for manually-configured; want error")
		}
		if _, ok := err.(ErrUnwritableHostCredentials); !ok {
			t.Fatalf("wrong error type %T; want ErrUnwritableHostCredentials", err)
		}
	}
	{
		err := credSrc.ForgetForHost(
			svchost.Hostname("manually-configured.example.com"),
		)
		if err == nil {
			t.Fatalf("successfully forgot for manually-configured; want error")
		}
		if _, ok := err.(ErrUnwritableHostCredentials); !ok {
			t.Fatalf("wrong error type %T; want ErrUnwritableHostCredentials", err)
		}
	}
	{
		// We don't have a credentials file at all yet, so this first call
		// must create it.
		err := credSrc.StoreForHost(
			svchost.Hostname("stored-locally.example.com"),
			svcauth.HostCredentialsToken("stored-locally"),
		)
		if err != nil {
			t.Fatalf("unexpected error storing locally: %s", err)
		}

		creds, err := credSrc.ForHost(svchost.Hostname("stored-locally.example.com"))
		if err != nil {
			t.Fatalf("failed to read back stored-locally credentials: %s", err)
		}

		if got, want := testReqAuthHeader(t, creds), "Bearer stored-locally"; got != want {
			t.Fatalf("wrong header value for stored-locally\ngot:  %s\nwant: %s", got, want)
		}

		got := readHostsInCredentialsFile(mockCredsFilename)
		want := map[svchost.Hostname]struct{}{
			svchost.Hostname("stored-locally.example.com"): struct{}{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong credentials file content\n%s", diff)
		}
	}

	// Now we'll switch to having a credential helper active.
	// If we were loading the real CLI config from disk here then this
	// entry would already be in cfg.Credentials, but we need to fake that
	// in the test because we're constructing this *Config value directly.
	cfg.Credentials["stored-locally.example.com"] = map[string]interface{}{
		"token": "stored-locally",
	}
	mockHelper := &mockCredentialsHelper{current: make(map[svchost.Hostname]cty.Value)}
	credSrc = cfg.credentialsSource(
		"mock", mockHelper,
		mockCredsFilename,
	)
	{
		err := credSrc.StoreForHost(
			svchost.Hostname("manually-configured.example.com"),
			svcauth.HostCredentialsToken("not-manually-configured"),
		)
		if err == nil {
			t.Fatalf("successfully stored for manually-configured with helper active; want error")
		}
	}
	{
		err := credSrc.StoreForHost(
			svchost.Hostname("stored-in-helper.example.com"),
			svcauth.HostCredentialsToken("stored-in-helper"),
		)
		if err != nil {
			t.Fatalf("unexpected error storing in helper: %s", err)
		}

		creds, err := credSrc.ForHost(svchost.Hostname("stored-in-helper.example.com"))
		if err != nil {
			t.Fatalf("failed to read back stored-in-helper credentials: %s", err)
		}

		if got, want := testReqAuthHeader(t, creds), "Bearer stored-in-helper"; got != want {
			t.Fatalf("wrong header value for stored-in-helper\ngot:  %s\nwant: %s", got, want)
		}

		// Nothing should have changed in the saved credentials file
		got := readHostsInCredentialsFile(mockCredsFilename)
		want := map[svchost.Hostname]struct{}{
			svchost.Hostname("stored-locally.example.com"): struct{}{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong credentials file content\n%s", diff)
		}
	}
	{
		// Because stored-locally is already in the credentials file, a new
		// store should be sent there rather than to the credentials helper.
		err := credSrc.StoreForHost(
			svchost.Hostname("stored-locally.example.com"),
			svcauth.HostCredentialsToken("stored-locally-again"),
		)
		if err != nil {
			t.Fatalf("unexpected error storing locally again: %s", err)
		}

		creds, err := credSrc.ForHost(svchost.Hostname("stored-locally.example.com"))
		if err != nil {
			t.Fatalf("failed to read back stored-locally credentials: %s", err)
		}

		if got, want := testReqAuthHeader(t, creds), "Bearer stored-locally-again"; got != want {
			t.Fatalf("wrong header value for stored-locally\ngot:  %s\nwant: %s", got, want)
		}
	}
	{
		// Forgetting a host already in the credentials file should remove it
		// from the credentials file, not from the helper.
		err := credSrc.ForgetForHost(
			svchost.Hostname("stored-locally.example.com"),
		)
		if err != nil {
			t.Fatalf("unexpected error forgetting locally: %s", err)
		}

		creds, err := credSrc.ForHost(svchost.Hostname("stored-locally.example.com"))
		if err != nil {
			t.Fatalf("failed to read back stored-locally credentials: %s", err)
		}

		if got, want := testReqAuthHeader(t, creds), ""; got != want {
			t.Fatalf("wrong header value for stored-locally\ngot:  %s\nwant: %s", got, want)
		}

		// Should not be present in the credentials file anymore
		got := readHostsInCredentialsFile(mockCredsFilename)
		want := map[svchost.Hostname]struct{}{}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong credentials file content\n%s", diff)
		}
	}
	{
		err := credSrc.ForgetForHost(
			svchost.Hostname("stored-in-helper.example.com"),
		)
		if err != nil {
			t.Fatalf("unexpected error forgetting in helper: %s", err)
		}

		creds, err := credSrc.ForHost(svchost.Hostname("stored-in-helper.example.com"))
		if err != nil {
			t.Fatalf("failed to read back stored-in-helper credentials: %s", err)
		}

		if got, want := testReqAuthHeader(t, creds), ""; got != want {
			t.Fatalf("wrong header value for stored-in-helper\ngot:  %s\nwant: %s", got, want)
		}
	}

	{
		// Finally, the log in our mock helper should show that it was only
		// asked to deal with stored-in-helper, not stored-locally.
		got := mockHelper.log
		want := []mockCredentialsHelperChange{
			{
				Host:   svchost.Hostname("stored-in-helper.example.com"),
				Action: "store",
			},
			{
				Host:   svchost.Hostname("stored-in-helper.example.com"),
				Action: "forget",
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected credentials helper operation log\n%s", diff)
		}
	}
}

type mockCredentialsHelperChange struct {
	Host   svchost.Hostname
	Action string
}

type mockCredentialsHelper struct {
	current map[svchost.Hostname]cty.Value
	log     []mockCredentialsHelperChange
}

// Assertion that mockCredentialsHelper implements svcauth.CredentialsSource
var _ svcauth.CredentialsSource = (*mockCredentialsHelper)(nil)

func (s *mockCredentialsHelper) ForHost(hostname svchost.Hostname) (svcauth.HostCredentials, error) {
	v, ok := s.current[hostname]
	if !ok {
		return nil, nil
	}
	return svcauth.HostCredentialsFromObject(v), nil
}

func (s *mockCredentialsHelper) StoreForHost(hostname svchost.Hostname, new svcauth.HostCredentialsWritable) error {
	s.log = append(s.log, mockCredentialsHelperChange{
		Host:   hostname,
		Action: "store",
	})
	s.current[hostname] = new.ToStore()
	return nil
}

func (s *mockCredentialsHelper) ForgetForHost(hostname svchost.Hostname) error {
	s.log = append(s.log, mockCredentialsHelperChange{
		Host:   hostname,
		Action: "forget",
	})
	delete(s.current, hostname)
	return nil
}

func TestMTLSCredentialsForHost(t *testing.T) {
	certFile, keyFile, caCertFile, err := generateSelfSignedCert(t)
	if err != nil {
		t.Fatalf("failed to generate self-signed certs: %v", err)
	}

	credSrc := &CredentialsSource{
		configured: map[svchost.Hostname]cty.Value{
			"configured.example.com": cty.ObjectVal(map[string]cty.Value{}),
			"only-mtls.example.com":  cty.ObjectVal(map[string]cty.Value{}),
		},
	}

	testReqMTLSAuthHeader := func(t *testing.T, creds svcauth.HostCredentials) *http.Request {
		t.Helper()

		if creds == nil {
			t.Fatal("No credentials found")
		}

		req, err := http.NewRequest("GET", "http://example.com/", nil)
		if err != nil {
			t.Fatalf("cannot construct HTTP request: %s", err)
		}
		creds.PrepareRequest(req)
		return req
	}

	t.Run("mtls credentials from environment", func(t *testing.T) {
		t.Setenv("TF_CLIENT_CERT_configured_example_com", certFile)
		t.Setenv("TF_CLIENT_KEY_configured_example_com", keyFile)
		t.Setenv("TF_CA_CERT_configured_example_com", caCertFile)
		t.Setenv("TF_TOKEN_configured_example_com", "configured-token")

		creds, err := credSrc.ForHost(svchost.Hostname("configured.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		mtlsCreds, ok := creds.(svcauth.HostCredentialsExtended)
		if !ok {
			t.Fatal("expected mTLS credentials")
		}

		// Verify that the TLS configuration is correct
		tlsConfig, err := mtlsCreds.GetTLSConfig()
		if err != nil {
			t.Fatalf("failed to get TLS config: %s", err)
		}

		if len(tlsConfig.Certificates) == 0 {
			t.Fatal("expected at least one certificate in TLS config")
		}

		// Check if CA certificate is loaded correctly
		if tlsConfig.RootCAs == nil {
			t.Fatal("expected RootCAs to be set")
		}

		// Check if the token is correctly set as an authorization header
		req := testReqMTLSAuthHeader(t, mtlsCreds)
		if got, want := req.Header.Get("Authorization"), "Bearer configured-token"; got != want {
			t.Errorf("wrong token header\ngot: %s\nwant: %s", got, want)
		}
	})

	t.Run("mtls credentials without token", func(t *testing.T) {
		t.Setenv("TF_CLIENT_CERT_only__mtls_example_com", certFile)
		t.Setenv("TF_CLIENT_KEY_only__mtls_example_com", keyFile)
		t.Setenv("TF_CA_CERT_only__mtls_example_com", caCertFile)
		creds, err := credSrc.ForHost(svchost.Hostname("only-mtls.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if creds == nil {
			t.Fatal("no credentials found")
		}

		mtlsCreds, ok := creds.(svcauth.HostCredentialsExtended)
		if !ok {
			t.Fatal("expected mTLS credentials")
		}

		// Verify that the TLS configuration is correct
		tlsConfig, err := mtlsCreds.GetTLSConfig()
		if err != nil {
			t.Fatalf("failed to get TLS config: %s", err)
		}

		if len(tlsConfig.Certificates) == 0 {
			t.Fatal("expected at least one certificate in TLS config")
		}

		// Check if CA certificate is loaded correctly
		if tlsConfig.RootCAs == nil {
			t.Fatal("expected RootCAs to be set")
		}

		// Since there's no token, the Authorization header should be empty
		req := testReqMTLSAuthHeader(t, mtlsCreds)
		if got := req.Header.Get("Authorization"); got != "" {
			t.Errorf("expected empty authorization header, got: %s", got)
		}
	})
}

// generateSelfSignedCert generates a self-signed certificate and private key
// and writes them to temporary files. It also generates a CA certificate.
func generateSelfSignedCert(t *testing.T) (certFile, keyFile, caCertFile string, err error) {
	t.Helper()

	// Generate a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create a CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign the CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	// Create a client certificate template
	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client Organization"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IsCA:        false,
	}

	// Sign the client certificate with the CA certificate
	clientCertDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, &caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create client certificate: %v", err)
	}

	// Encode CA certificate to PEM
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Encode client certificate to PEM
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})

	// Encode private key to PEM
	keyPEM, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyPEM})

	// Write the CA certificate to a temporary file
	caCertTempFile, err := os.CreateTemp("", "ca_cert.pem")
	if err != nil {
		t.Fatalf("failed to create CA cert temp file: %v", err)
	}
	defer caCertTempFile.Close()
	if _, err := caCertTempFile.Write(caCertPEM); err != nil {
		t.Fatalf("failed to write CA cert to file: %v", err)
	}

	// Write the client certificate to a temporary file
	certTempFile, err := os.CreateTemp("", "cert.pem")
	if err != nil {
		t.Fatalf("failed to create cert temp file: %v", err)
	}
	defer certTempFile.Close()
	if _, err := certTempFile.Write(clientCertPEM); err != nil {
		t.Fatalf("failed to write cert to file: %v", err)
	}

	// Write the private key to a temporary file
	keyTempFile, err := os.CreateTemp("", "key.pem")
	if err != nil {
		t.Fatalf("failed to create key temp file: %v", err)
	}
	defer keyTempFile.Close()
	if _, err := keyTempFile.Write(keyPEMBytes); err != nil {
		t.Fatalf("failed to write key to file: %v", err)
	}

	return certTempFile.Name(), keyTempFile.Name(), caCertTempFile.Name(), nil
}

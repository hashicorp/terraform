// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package http

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"address": {
						Type:        cty.String,
						Optional:    true, // Must be set but can be set using the TF_HTTP_ADDRESS environment variable
						Description: "The address of the REST endpoint",
					},
					"update_method": {
						Type:        cty.String,
						Optional:    true,
						Description: "HTTP method to use when updating state",
					},
					"lock_address": {
						Type:        cty.String,
						Optional:    true,
						Description: "The address of the lock REST endpoint",
					},
					"unlock_address": {
						Type:        cty.String,
						Optional:    true,
						Description: "The address of the unlock REST endpoint",
					},
					"lock_method": {
						Type:        cty.String,
						Optional:    true,
						Description: "The HTTP method to use when locking",
					},
					"unlock_method": {
						Type:        cty.String,
						Optional:    true,
						Description: "The HTTP method to use when unlocking",
					},
					"username": {
						Type:        cty.String,
						Optional:    true,
						Description: "The username for HTTP basic authentication",
					},
					"password": {
						Type:        cty.String,
						Optional:    true,
						Description: "The password for HTTP basic authentication",
					},
					"skip_cert_verification": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to skip TLS verification",
					},
					"retry_max": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The number of HTTP request retries",
					},
					"retry_wait_min": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The minimum time in seconds to wait between HTTP request attempts",
					},
					"retry_wait_max": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The maximum time in seconds to wait between HTTP request attempts",
					},
					"client_ca_certificate_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded CA certificate chain used by the client to verify server certificates during TLS authentication",
					},
					"client_certificate_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded certificate used by the server to verify the client during mutual TLS (mTLS) authentication",
					},
					"client_private_key_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded private key, required if client_certificate_pem is specified",
					},
				},
			},
		},
	}
}

type Backend struct {
	backendbase.Base

	client *httpClient
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	address := backendbase.GetAttrEnvDefaultFallback(
		configVal, "address",
		"TF_HTTP_ADDRESS", cty.StringVal(""),
	).AsString()
	if address == "" {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("address argument is required"),
		)
	}
	updateURL, err := url.Parse(address)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("failed to parse address URL: %s", err),
		)
	}
	if updateURL.Scheme != "http" && updateURL.Scheme != "https" {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("address must be HTTP or HTTPS"),
		)
	}

	updateMethod := backendbase.GetAttrEnvDefaultFallback(
		configVal, "update_method",
		"TF_HTTP_UPDATE_METHOD", cty.StringVal("POST"),
	).AsString()

	var lockURL *url.URL
	if v := backendbase.GetAttrEnvDefault(configVal, "lock_address", "TF_HTTP_LOCK_ADDRESS"); !v.IsNull() {
		var err error
		lockURL, err = url.Parse(v.AsString())
		if err != nil {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("failed to parse lock_address URL: %s", err),
			)
		}
		if lockURL.Scheme != "http" && lockURL.Scheme != "https" {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("lock_address must be HTTP or HTTPS"),
			)
		}
	}
	lockMethod := backendbase.GetAttrEnvDefaultFallback(
		configVal, "lock_method",
		"TF_HTTP_LOCK_METHOD", cty.StringVal("LOCK"),
	).AsString()

	var unlockURL *url.URL
	if v := backendbase.GetAttrEnvDefault(configVal, "unlock_address", "TF_HTTP_UNLOCK_ADDRESS"); !v.IsNull() {
		var err error
		unlockURL, err = url.Parse(v.AsString())
		if err != nil {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("failed to parse unlock_address URL: %s", err),
			)
		}
		if unlockURL.Scheme != "http" && unlockURL.Scheme != "https" {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("unlock_address must be HTTP or HTTPS"),
			)
		}
	}
	unlockMethod := backendbase.GetAttrEnvDefaultFallback(
		configVal, "unlock_method",
		"TF_HTTP_UNLOCK_METHOD", cty.StringVal("UNLOCK"),
	).AsString()

	retryMax, err := backendbase.IntValue(
		backendbase.GetAttrEnvDefaultFallback(
			configVal, "retry_max",
			"TF_HTTP_RETRY_MAX", cty.NumberIntVal(2),
		),
	)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("invalid retry_max: %s", err),
		)
	}
	retryWaitMin, err := backendbase.IntValue(
		backendbase.GetAttrEnvDefaultFallback(
			configVal, "retry_wait_min",
			"TF_HTTP_RETRY_WAIT_MIN", cty.NumberIntVal(1),
		),
	)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("invalid retry_wait_min: %s", err),
		)
	}
	retryWaitMax, err := backendbase.IntValue(
		backendbase.GetAttrEnvDefaultFallback(
			configVal, "retry_wait_max",
			"TF_HTTP_RETRY_WAIT_MAX", cty.NumberIntVal(30),
		),
	)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("invalid retry_wait_max: %s", err),
		)
	}

	rClient := retryablehttp.NewClient()
	rClient.RetryMax = int(retryMax)
	rClient.RetryWaitMin = time.Duration(retryWaitMin) * time.Second
	rClient.RetryWaitMax = time.Duration(retryWaitMax) * time.Second
	rClient.Logger = log.New(logging.LogOutput(), "", log.Flags())
	if err = b.configureTLS(rClient, configVal); err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	b.client = &httpClient{
		URL:          updateURL,
		UpdateMethod: updateMethod,

		LockURL:      lockURL,
		LockMethod:   lockMethod,
		UnlockURL:    unlockURL,
		UnlockMethod: unlockMethod,

		Username: backendbase.GetAttrEnvDefaultFallback(
			configVal, "username",
			"TF_HTTP_USERNAME", cty.StringVal(""),
		).AsString(),
		Password: backendbase.GetAttrEnvDefaultFallback(
			configVal, "password",
			"TF_HTTP_PASSWORD", cty.StringVal(""),
		).AsString(),

		// accessible only for testing use
		Client: rClient,
	}
	return nil
}

// configureTLS configures TLS when needed; if there are no conditions requiring TLS, no change is made.
func (b *Backend) configureTLS(client *retryablehttp.Client, configVal cty.Value) error {
	// If there are no conditions needing to configure TLS, leave the client untouched
	skipCertVerification := backendbase.MustBoolValue(
		backendbase.GetAttrDefault(configVal, "skip_cert_verification", cty.False),
	)
	clientCACertificatePem := backendbase.GetAttrEnvDefaultFallback(
		configVal, "client_ca_certificate_pem",
		"TF_HTTP_CLIENT_CA_CERTIFICATE_PEM", cty.StringVal(""),
	).AsString()
	clientCertificatePem := backendbase.GetAttrEnvDefaultFallback(
		configVal, "client_certificate_pem",
		"TF_HTTP_CLIENT_CERTIFICATE_PEM", cty.StringVal(""),
	).AsString()
	clientPrivateKeyPem := backendbase.GetAttrEnvDefaultFallback(
		configVal, "client_private_key_pem",
		"TF_HTTP_CLIENT_PRIVATE_KEY_PEM", cty.StringVal(""),
	).AsString()
	if !skipCertVerification && clientCACertificatePem == "" && clientCertificatePem == "" && clientPrivateKeyPem == "" {
		return nil
	}
	if clientCertificatePem != "" && clientPrivateKeyPem == "" {
		return fmt.Errorf("client_certificate_pem is set but client_private_key_pem is not")
	}
	if clientPrivateKeyPem != "" && clientCertificatePem == "" {
		return fmt.Errorf("client_private_key_pem is set but client_certificate_pem is not")
	}

	// TLS configuration is needed; create an object and configure it
	var tlsConfig tls.Config
	client.HTTPClient.Transport.(*http.Transport).TLSClientConfig = &tlsConfig

	if skipCertVerification {
		// ignores TLS verification
		tlsConfig.InsecureSkipVerify = true
	}
	if clientCACertificatePem != "" {
		// trust servers based on a CA
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(clientCACertificatePem)) {
			return errors.New("failed to append certs")
		}
	}
	if clientCertificatePem != "" && clientPrivateKeyPem != "" {
		// attach a client certificate to the TLS handshake (aka mTLS)
		certificate, err := tls.X509KeyPair([]byte(clientCertificatePem), []byte(clientPrivateKeyPem))
		if err != nil {
			return fmt.Errorf("cannot load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(string, bool) error {
	return backend.ErrWorkspacesNotSupported
}

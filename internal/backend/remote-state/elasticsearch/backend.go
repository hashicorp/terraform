// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package elasticsearch

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	defaultIndex = "terraform_remote_state"
)

// New creates a new backend for Elasticsearch remote state.
func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"endpoints": {
						Type:        cty.List(cty.String),
						Optional:    true,
						Description: "Elasticsearch cluster endpoints (e.g. [\"http://localhost:9200\"])",
					},
					"index": {
						Type:        cty.String,
						Optional:    true,
						Description: "Index name for Terraform state storage",
					},
					"username": {
						Type:        cty.String,
						Optional:    true,
						Description: "Username for Elasticsearch authentication",
					},
					"password": {
						Type:        cty.String,
						Optional:    true,
						Description: "Password for Elasticsearch authentication",
						Sensitive:   true,
					},
					"skip_cert_verification": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to skip TLS certificate verification",
					},
					"ca_certificate_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded CA certificate chain used to verify Elasticsearch server certificates",
					},
					"client_certificate_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded certificate for mutual TLS authentication",
					},
					"client_private_key_pem": {
						Type:        cty.String,
						Optional:    true,
						Description: "A PEM-encoded private key for mutual TLS authentication",
						Sensitive:   true,
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"index": {
					EnvVars:  []string{"ELASTICSEARCH_INDEX"},
					Fallback: defaultIndex,
				},
				"username": {
					EnvVars: []string{"ELASTICSEARCH_USERNAME"},
				},
				"password": {
					EnvVars: []string{"ELASTICSEARCH_PASSWORD"},
				},
				"skip_cert_verification": {
					EnvVars:  []string{"ELASTICSEARCH_SKIP_CERT_VERIFICATION"},
					Fallback: "false",
				},
				"ca_certificate_pem": {
					EnvVars: []string{"ELASTICSEARCH_CA_CERTIFICATE_PEM"},
				},
				"client_certificate_pem": {
					EnvVars: []string{"ELASTICSEARCH_CLIENT_CERTIFICATE_PEM"},
				},
				"client_private_key_pem": {
					EnvVars: []string{"ELASTICSEARCH_CLIENT_PRIVATE_KEY_PEM"},
				},
			},
		},
	}
}

type Backend struct {
	backendbase.Base

	// The fields below are set from configure
	client *elasticsearch.Client
	index  string
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	data := backendbase.NewSDKLikeData(configVal)

	// Get endpoints
	endpoints := []string{}
	endpointsAttr := configVal.GetAttr("endpoints")
	if !endpointsAttr.IsNull() && endpointsAttr.Type().IsListType() {
		for it := endpointsAttr.ElementIterator(); it.Next(); {
			_, val := it.Element()
			endpoints = append(endpoints, val.AsString())
		}
	}

	if len(endpoints) == 0 {
		if env := os.Getenv("ELASTICSEARCH_ENDPOINTS"); env != "" {
			endpoints = strings.Split(env, ",")
		}
	}

	// Final fallback
	if len(endpoints) == 0 {
		endpoints = []string{"http://localhost:9200"}
	}

	b.index = data.String("index")
	username := data.String("username")
	password := data.String("password")

	// Configure TLS
	tlsConfig, err := b.configureTLS(&data, configVal)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	// Create Elasticsearch client config
	cfg := elasticsearch.Config{
		Addresses: endpoints,
	}

	if username != "" {
		cfg.Username = username
		cfg.Password = password
	}

	if tlsConfig != nil {
		cfg.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	// Create Elasticsearch client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("failed to create Elasticsearch client: %w", err),
		)
	}

	b.client = client

	// Test connection and ensure index exists
	if err := b.ensureIndex(); err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("failed to initialize Elasticsearch: %w", err),
		)
	}

	return nil
}

// configureTLS configures TLS when needed
func (b *Backend) configureTLS(data *backendbase.SDKLikeData, configVal cty.Value) (*tls.Config, error) {
	skipCertVerification := data.Bool("skip_cert_verification")
	caCertificatePem := data.String("ca_certificate_pem")
	clientCertificatePem := data.String("client_certificate_pem")
	clientPrivateKeyPem := data.String("client_private_key_pem")

	if !skipCertVerification && caCertificatePem == "" && clientCertificatePem == "" && clientPrivateKeyPem == "" {
		return nil, nil
	}

	if clientCertificatePem != "" && clientPrivateKeyPem == "" {
		return nil, fmt.Errorf("client_certificate_pem is set but client_private_key_pem is not")
	}
	if clientPrivateKeyPem != "" && clientCertificatePem == "" {
		return nil, fmt.Errorf("client_private_key_pem is set but client_certificate_pem is not")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{}

	if skipCertVerification {
		tlsConfig.InsecureSkipVerify = true
	}

	if caCertificatePem != "" {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(caCertificatePem)) {
			return nil, errors.New("failed to append CA certificates")
		}
	}

	if clientCertificatePem != "" && clientPrivateKeyPem != "" {
		certificate, err := tls.X509KeyPair([]byte(clientCertificatePem), []byte(clientPrivateKeyPem))
		if err != nil {
			return nil, fmt.Errorf("cannot load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	return tlsConfig, nil
}

// ensureIndex ensures the Elasticsearch index exists
func (b *Backend) ensureIndex() error {
	client := &RemoteClient{
		Client: b.client,
		Index:  b.index,
	}

	return client.ensureIndex()
}

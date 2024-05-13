// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package gcs implements remote storage of state on Google Cloud Storage (GCS).
package gcs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Backend implements "backend".Backend for GCS.
// Schema() and PrepareConfig() are implemented by embedding backendbase.Base.
// Configure(), State(), DeleteState() and States() are implemented explicitly.
type Backend struct {
	backendbase.Base

	storageClient *storage.Client

	bucketName string
	prefix     string

	encryptionKey []byte
	kmsKeyName    string
}

func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bucket": {
						Type:        cty.String,
						Required:    true,
						Description: "The name of the Google Cloud Storage bucket",
					},

					"prefix": {
						Type:        cty.String,
						Optional:    true,
						Description: "The directory where state files will be saved inside the bucket",
					},

					"credentials": {
						Type:        cty.String,
						Optional:    true,
						Description: "Google Cloud JSON Account Key",
					},

					"access_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "An OAuth2 token used for GCP authentication",
					},

					"impersonate_service_account": {
						Type:        cty.String,
						Optional:    true,
						Description: "The service account to impersonate for all Google API Calls",
					},

					"impersonate_service_account_delegates": {
						Type:        cty.List(cty.String),
						Optional:    true,
						Description: "The delegation chain for the impersonated service account",
					},

					"encryption_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "A 32 byte base64 encoded 'customer supplied encryption key' used when reading and writing state files in the bucket.",
					},

					"kms_encryption_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "A Cloud KMS key ('customer managed encryption key') used when reading and writing state files in the bucket. Format should be 'projects/{{project}}/locations/{{location}}/keyRings/{{keyRing}}/cryptoKeys/{{name}}'.",
					},

					"storage_custom_endpoint": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"prefix": {
					Fallback: "",
				},
				"credentials": {
					Fallback: "",
				},
				"access_token": {
					EnvVars: []string{"GOOGLE_OAUTH_ACCESS_TOKEN"},
				},
				"impersonate_service_account": {
					EnvVars: []string{
						"GOOGLE_BACKEND_IMPERSONATE_SERVICE_ACCOUNT",
						"GOOGLE_IMPERSONATE_SERVICE_ACCOUNT",
					},
				},
				"encryption_key": {
					EnvVars: []string{"GOOGLE_ENCRYPTION_KEY"},
				},
				"kms_encryption_key": {
					EnvVars: []string{"GOOGLE_KMS_ENCRYPTION_KEY"},
				},
				"storage_custom_endpoint": {
					EnvVars: []string{
						"GOOGLE_BACKEND_STORAGE_CUSTOM_ENDPOINT",
						"GOOGLE_STORAGE_CUSTOM_ENDPOINT",
					},
				},
			},
		},
	}
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	if b.storageClient != nil {
		return nil
	}

	// TODO: Update the Backend API to pass the real context.Context from
	// the running command.
	ctx := context.TODO()

	data := backendbase.NewSDKLikeData(configVal)

	if data.String("encryption_key") != "" && data.String("kms_encryption_key") != "" {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("can't set both encryption_key and kms_encryption_key"),
		)
	}
	// The above catches the main case where both of the arguments are set to
	// a non-empty value, but we also want to reject the situation where
	// both are present in the configuration regardless of what values were
	// assigned to them. (This check doesn't take the environment variables
	// into account, so must allow neither to be set in the main configuration.)
	if !(configVal.GetAttr("encryption_key").IsNull() || configVal.GetAttr("kms_encryption_key").IsNull()) {
		// This rejects a configuration like:
		//     encryption_key     = ""
		//     kms_encryption_key = ""
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("can't set both encryption_key and kms_encryption_key"),
		)
	}

	b.bucketName = data.String("bucket")
	b.prefix = strings.TrimLeft(data.String("prefix"), "/")
	if b.prefix != "" && !strings.HasSuffix(b.prefix, "/") {
		b.prefix = b.prefix + "/"
	}

	var opts []option.ClientOption
	var credOptions []option.ClientOption

	// Add credential source
	var creds string
	var tokenSource oauth2.TokenSource

	if v := data.String("access_token"); v != "" {
		tokenSource = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: v,
		})
	} else if v := data.String("credentials"); v != "" {
		creds = v
	} else if v := os.Getenv("GOOGLE_BACKEND_CREDENTIALS"); v != "" {
		creds = v
	} else {
		creds = os.Getenv("GOOGLE_CREDENTIALS")
	}

	if tokenSource != nil {
		credOptions = append(credOptions, option.WithTokenSource(tokenSource))
	} else if creds != "" {

		// to mirror how the provider works, we accept the file path or the contents
		contents, err := readPathOrContents(creds)
		if err != nil {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("Error loading credentials: %s", err),
			)
		}

		if !json.Valid([]byte(contents)) {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("the string provided in credentials is neither valid json nor a valid file path"),
			)
		}

		credOptions = append(credOptions, option.WithCredentialsJSON([]byte(contents)))
	}

	// Service Account Impersonation
	if v := data.String("impersonate_service_account"); v != "" {
		ServiceAccount := v
		var delegates []string

		delegatesVal := data.GetAttr("impersonate_service_account_delegates", cty.List(cty.String))
		if !delegatesVal.IsNull() && delegatesVal.LengthInt() != 0 {
			delegates = make([]string, 0, delegatesVal.LengthInt())
			for it := delegatesVal.ElementIterator(); it.Next(); {
				_, v := it.Element()
				if v.IsNull() {
					return backendbase.ErrorAsDiagnostics(
						fmt.Errorf("impersonate_service_account_delegates elements must not be null"),
					)
				}
				delegates = append(delegates, v.AsString())
			}
		}

		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: ServiceAccount,
			Scopes:          []string{storage.ScopeReadWrite},
			Delegates:       delegates,
		}, credOptions...)

		if err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}

		opts = append(opts, option.WithTokenSource(ts))

	} else {
		opts = append(opts, credOptions...)
	}

	opts = append(opts, option.WithUserAgent(httpclient.UserAgentString()))

	// Custom endpoint for storage API
	if storageEndpoint := data.String("storage_custom_endpoint"); storageEndpoint != "" {
		endpoint := option.WithEndpoint(storageEndpoint)
		opts = append(opts, endpoint)
	}
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("storage.NewClient() failed: %v", err),
		)
	}

	b.storageClient = client

	// Customer-supplied encryption
	key := data.String("encryption_key")
	if key != "" {
		kc, err := readPathOrContents(key)
		if err != nil {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("Error loading encryption key: %s", err),
			)
		}

		// The GCS client expects a customer supplied encryption key to be
		// passed in as a 32 byte long byte slice. The byte slice is base64
		// encoded before being passed to the API. We take a base64 encoded key
		// to remain consistent with the GCS docs.
		// https://cloud.google.com/storage/docs/encryption#customer-supplied
		// https://github.com/GoogleCloudPlatform/google-cloud-go/blob/def681/storage/storage.go#L1181
		k, err := base64.StdEncoding.DecodeString(kc)
		if err != nil {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("Error decoding encryption key: %s", err),
			)
		}
		b.encryptionKey = k
	}

	// Customer-managed encryption
	kmsName := data.String("kms_encryption_key")
	if kmsName != "" {
		b.kmsKeyName = kmsName
	}

	return nil
}

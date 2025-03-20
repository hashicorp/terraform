// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"path"
)

var (
	lockFileSuffix = ".lock"
)

func New() backend.Backend {
	return &Backend{}
}

// New creates a new backend for OSS remote state.
func (b *Backend) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			KeyAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The name of the state file stored on the remote backend.",
			},
			BucketAttrName: {
				Type:        cty.String,
				Required:    true,
				Description: "The name of the OCI Object Storage bucket.",
			},
			NamespaceAttrName: {
				Type:        cty.String,
				Required:    true,
				Description: "The namespace of the OCI Object Storage.",
			},
			RegionAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "OCI region where the bucket is located.",
			},
			TenancyOcidAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The OCID of the tenancy.",
			},
			UserOcidAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The OCID of the user.",
			},
			FingerprintAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The fingerprint of the user's API key.",
			},
			PrivateKeyAttrName: {
				Type:        cty.String,
				Sensitive:   true,
				Optional:    true,
				Description: "The private key for API authentication.",
			},
			PrivateKeyPathAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "Path to the private key file.",
			},
			PrivateKeyPasswordAttrName: {
				Type:        cty.String,
				Sensitive:   true,
				Optional:    true,
				Description: "Passphrase for the private key, if required.",
			},
			AuthAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "Authentication method (API key, Instance Principal, Resource Principal, etc.).",
			},

			ConfigFileProfileAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "Profile name from the OCI config file.",
			},
			WorkspaceKeyPrefixAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The prefix applied to the non-default state path inside the bucket.",
			},
			KmsKeyIdAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The OCID of a master encryption key used to call the Key Management service to generate a data encryption key or to encrypt or decrypt a data encryption key.",
			},
			SseCustomerKeyAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The optional header that specifies the base64-encoded 256-bit encryption key to use to encrypt or decrypt the data.",
				Sensitive:   true,
			},
			SseCustomerKeySHA256AttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The optional header that specifies the base64-encoded SHA256 hash of the encryption key. This value is used to check the integrity of the encryption key.",
			},
			SseCustomerAlgorithmAttrName: {
				Type:        cty.String,
				Optional:    true,
				Description: "The optional header that specifies \"AES256\" as the encryption algorithm.",
			},
		},
	}
}

type Backend struct {
	configProvider       ociAuthConfigProvider
	bucket               string
	key                  string
	namespace            string
	workspaceKeyPrefix   string
	kmsKeyID             string
	SSECustomerKey       string
	SSECustomerKeySHA256 string
	SSECustomerAlgorithm string
	client               *RemoteClient
}

func (b *Backend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if keyVal, ok := getBackendAttrWithDefault(obj, KeyAttrName, defaultKeyValue); ok {
		validateStringObjectPath(keyVal.AsString(), cty.GetAttrPath(KeyAttrName), &diags)
	}
	return obj, diags
}
func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if obj.IsNull() {
		diags.Append(tfdiags.AttributeValue(tfdiags.Error, "Invalid Configuration", "Received null configuration for OCI backend.", cty.GetAttrPath(".")))
		return diags
	}

	if bucketVal, ok := getBackendAttr(obj, BucketAttrName); ok {
		b.bucket = bucketVal.AsString()
	}
	if namespaceVal, ok := getBackendAttr(obj, NamespaceAttrName); ok {
		b.namespace = namespaceVal.AsString()
	} else {
		diags.Append(tfdiags.AttributeValue(tfdiags.Error, "Missing Required Attribute", "Bucket name cannot be null", cty.GetAttrPath("namespace")))
	}
	if keyVal, ok := getBackendAttrWithDefault(obj, KeyAttrName, defaultKeyValue); ok {
		b.key = keyVal.AsString()
	}

	if workspaceKeyPrefixVal, ok := getBackendAttrWithDefault(obj, WorkspaceKeyPrefixAttrName, defaultEnvPrefix); ok {
		b.workspaceKeyPrefix = workspaceKeyPrefixVal.AsString()
	}

	if kmsKeyIdVal, ok := getBackendAttr(obj, KmsKeyIdAttrName); ok {
		b.kmsKeyID = kmsKeyIdVal.AsString()
	}
	if customerKeyVal, ok := getBackendAttr(obj, SseCustomerKeyAttrName); ok {
		b.SSECustomerKey = customerKeyVal.AsString()
	}
	if customerKeySHA256Val, ok := getBackendAttr(obj, SseCustomerKeySHA256AttrName); ok {
		b.SSECustomerKeySHA256 = customerKeySHA256Val.AsString()
	}
	if customerAlgorithmVal, ok := getBackendAttrWithDefault(obj, SseCustomerAlgorithmAttrName, DefaultAlgorithm); ok {
		b.SSECustomerAlgorithm = customerAlgorithmVal.AsString()
	}
	b.configProvider = newOciAuthConfigProvider(obj)

	err := b.configureRemoteClient()
	if err != nil {
		diags = append(diags, backendbase.ErrorAsDiagnostics(err)[0])
	}
	return diags
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.key
	}

	return path.Join(b.workspaceKeyPrefix, name, b.key)
}

// getLockFilePath returns the path to the lock file for the given Terraform state.
// For `default.tfstate`, the lock file is stored at `default.tfstate.tflock`.
func (b *Backend) getLockFilePath(name string) string {
	return b.path(name) + lockFileSuffix
}

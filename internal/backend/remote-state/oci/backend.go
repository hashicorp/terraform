// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	lockFileSuffix = ".lock"
)

func New() backend.Backend {
	return &Backend{}
}

// New creates a new backend for oci remote state.
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
	if obj.IsNull() {

		diags.Append(tfdiags.AttributeValue(tfdiags.Error, "Invalid Configuration", "Received null configuration for OCI backend.", cty.GetAttrPath(".")))
		return obj, diags
	}
	if bucketVal := obj.GetAttr(BucketAttrName); bucketVal.IsNull() || bucketVal.AsString() == "" {
		diags = diags.Append(requiredAttributeErrDiag(cty.GetAttrPath(BucketAttrName)))
	} else {
		validateStringBucketName(bucketVal.AsString(), cty.GetAttrPath(BucketAttrName), &diags)
	}
	if namespaceVal := obj.GetAttr(NamespaceAttrName); namespaceVal.IsNull() || namespaceVal.AsString() == "" {
		diags = diags.Append(requiredAttributeErrDiag(cty.GetAttrPath(NamespaceAttrName)))
	}
	if keyVal, ok := getBackendAttrWithDefault(obj, KeyAttrName, defaultKeyValue); ok {
		validateStringObjectPath(keyVal.AsString(), cty.GetAttrPath(KeyAttrName), &diags)
	}
	if workspaceKeyPrefixVal, ok := getBackendAttrWithDefault(obj, WorkspaceKeyPrefixAttrName, defaultWorkspaceEnvPrefix); ok {
		validateStringWorkspacePrefix(workspaceKeyPrefixVal.AsString(), cty.GetAttrPath(WorkspaceKeyPrefixAttrName), &diags)
	}
	authVal, ok := getBackendAttr(obj, AuthAttrName)
	if ok && len(authVal.AsString()) > 0 {

		switch strings.ToLower(authVal.AsString()) {
		case strings.ToLower(AuthAPIKeySetting):
			//Nothing to do
			return obj, diags
		case strings.ToLower(AuthInstancePrincipalSetting), strings.ToLower(AuthInstancePrincipalWithCertsSetting), strings.ToLower(ResourcePrincipal), strings.ToLower(AuthSecurityToken):
			region, _ := getBackendAttr(obj, RegionAttrName)
			if region.AsString() == "" {
				diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
					"Missing region attribute required",
					fmt.Sprintf("The attribute %q is required by the backend for %s authentication.\n\n", RegionAttrName, authVal.AsString()), cty.GetAttrPath(RegionAttrName),
				))
			}
			if strings.ToLower(authVal.AsString()) == strings.ToLower(AuthSecurityToken) {
				profileVal, _ := getBackendAttr(obj, ConfigFileProfileAttrName)
				if profileVal.IsNull() || profileVal.AsString() == "" {
					diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
						"Missing config_file_profile attribute required",
						fmt.Sprintf("The attribute %q is required by the backend for %s authentication.\n\n", ConfigFileProfileAttrName, authVal.AsString()), cty.GetAttrPath(ConfigFileProfileAttrName),
					))
				}
			}
		default:
			diags = diags.Append(tfdiags.AttributeValue(tfdiags.Error,
				"Invalid authentication method",
				fmt.Sprintf("auth must be one of '%s' or '%s' or '%s' or '%s' or '%s' or '%s'", AuthAPIKeySetting, AuthInstancePrincipalSetting, AuthInstancePrincipalWithCertsSetting, AuthSecurityToken, ResourcePrincipal, AuthOKEWorkloadIdentity), cty.GetAttrPath(AuthAttrName),
			))
		}
	}

	customerKey, _ := getBackendAttr(obj, SseCustomerKeyAttrName)
	customerKeySHA, _ := getBackendAttr(obj, SseCustomerKeySHA256AttrName)
	kmsKeyId, _ := getBackendAttr(obj, KmsKeyIdAttrName)
	privateKey, _ := getBackendAttr(obj, PrivateKeyAttrName)
	privateKeyPath, _ := getBackendAttr(obj, PrivateKeyPathAttrName)

	if (!customerKey.IsNull() && len(customerKey.AsString()) > 0) && (customerKeySHA.IsNull() || len(customerKeySHA.AsString()) == 0) {
		diags = diags.Append(attributeErrDiag(
			"Invalid Attribute Combination",
			`  sse_customer_key and its SHA both required.`,
			cty.GetAttrPath(SseCustomerKeySHA256AttrName)))
	}
	if !customerKey.IsNull() && len(customerKey.AsString()) > 0 && !kmsKeyId.IsNull() && len(kmsKeyId.AsString()) > 0 {
		diags = diags.Append(attributeErrDiag(
			"Invalid Attribute Combination",
			`Only one of kms_key_id, sse_customer_key can be set.`,
			cty.GetAttrPath(KmsKeyIdAttrName),
		))
	}
	if !privateKey.IsNull() && len(privateKey.AsString()) > 0 && !privateKeyPath.IsNull() && len(privateKeyPath.AsString()) > 0 {
		diags = diags.Append(attributeErrDiag(
			"Invalid Attribute Combination",
			`Only one of private_key, private_key_path can be set.`,
			cty.GetAttrPath(PrivateKeyPathAttrName),
		))
	}
	return obj, diags
}
func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if bucketVal, ok := getBackendAttr(obj, BucketAttrName); ok {
		b.bucket = bucketVal.AsString()
	}
	if namespaceVal, ok := getBackendAttr(obj, NamespaceAttrName); ok {
		b.namespace = namespaceVal.AsString()
	}
	if keyVal, ok := getBackendAttrWithDefault(obj, KeyAttrName, defaultKeyValue); ok {
		b.key = keyVal.AsString()
	}

	if workspaceKeyPrefixVal, ok := getBackendAttrWithDefault(obj, WorkspaceKeyPrefixAttrName, defaultWorkspaceEnvPrefix); ok {
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

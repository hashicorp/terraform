// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import "time"

const (
	AuthAPIKeySetting                     = "ApiKey"
	AuthInstancePrincipalSetting          = "InstancePrincipal"
	AuthInstancePrincipalWithCertsSetting = "InstancePrincipalWithCerts"
	AuthSecurityToken                     = "SecurityToken"
	AuthOKEWorkloadIdentity               = "OKEWorkloadIdentity"
	ResourcePrincipal                     = "ResourcePrincipal"

	TfEnvPrefix  = "TF_VAR_"
	OciEnvPrefix = "OCI_"

	AuthAttrName               = "auth"
	TenancyOcidAttrName        = "tenancy_ocid"
	UserOcidAttrName           = "user_ocid"
	FingerprintAttrName        = "fingerprint"
	PrivateKeyAttrName         = "private_key"
	PrivateKeyPathAttrName     = "private_key_path"
	PrivateKeyPasswordAttrName = "private_key_password"
	RegionAttrName             = "region"
	WorkspaceKeyPrefixAttrName = "workspace_key_prefix"
	KmsKeyIdAttrName           = "kms_key_id"

	KeyAttrName       = "key"
	defaultKeyValue   = "terraform.tfstate"
	BucketAttrName    = "bucket"
	NamespaceAttrName = "namespace"

	ConfigFileProfileAttrName = "config_file_profile"

	AcceptLocalCerts = "accept_local_certs"

	//	HTTPRequestTimeout specifies the maximum duration for completing an HTTP request.
	HTTPRequestTimeOut    = "HTTP_REQUEST_TIMEOUT"
	DefaultRequestTimeout = 0
	// DialContextConnectionTimeout defines the timeout for establishing a connection during a network dial operation.
	DialContextConnectionTimeout = "DIAL_CONTEXT_CONNECTION_TIMEOUT"
	DefaultConnectionTimeout     = 10 * time.Second
	// TLSHandshakeTimeout indicates the maximum time allowed for the TLS handshake process.
	TLSHandshakeTimeout        = "TLS_HANDSHAKE_TIMEOUT"
	DefaultTLSHandshakeTimeout = 10 * time.Second

	DefaultConfigFileName = "config"
	DefaultConfigDirName  = ".oci"
)

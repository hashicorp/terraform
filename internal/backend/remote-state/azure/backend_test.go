// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	t.Parallel()

	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := map[string]interface{}{
		"storage_account_name": "tfaccount",
		"container_name":       "tfcontainer",
		"key":                  "state",
		"snapshot":             false,
		// Access Key must be Base64
		"access_key": "QUNDRVNTX0tFWQ0K",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if b.containerName != "tfcontainer" {
		t.Fatalf("Incorrect bucketName was populated")
	}
	if b.keyName != "state" {
		t.Fatalf("Incorrect keyName was populated")
	}
	if b.snapshot != false {
		t.Fatalf("Incorrect snapshot was populated")
	}
}

func TestAccBackendAccessKeyBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"access_key":           m.storageAccessKey,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendSASTokenBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	sasToken, err := buildSasToken(m.names.storageAccountName, m.storageAccessKey)
	if err != nil {
		t.Fatalf("Error building SAS Token: %+v", err)
	}

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"sas_token":            *sasToken,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendGithubOIDCBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackendRunningInGitHubActions(t)

	oidcRequestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if oidcRequestToken == "" {
		t.Fatalf("Missing ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	}

	oidcRequestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	if oidcRequestURL == "" {
		t.Fatalf("Missing ACTIONS_ID_TOKEN_REQUEST_URL")
	}

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"use_oidc":             true,
		"oidc_request_token":   oidcRequestToken,
		"oidc_request_url":     oidcRequestURL,
		"tenant_id":            m.tenantId,
		"client_id":            m.clientId,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendADOPipelinesOIDCBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackendRunningInADOPipelines(t)

	oidcRequestToken := os.Getenv("SYSTEM_ACCESSTOKEN")
	if oidcRequestToken == "" {
		t.Fatalf("Missing SYSTEM_ACCESSTOKEN")
	}

	oidcRequestURL := os.Getenv("SYSTEM_OIDCREQUESTURI")
	if oidcRequestURL == "" {
		t.Fatalf("Missing SYSTEM_OIDCREQUESTURI")
	}

	adoPipelineServiceConnectionId := os.Getenv("ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID")
	if adoPipelineServiceConnectionId == "" {
		t.Fatalf("Missing ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID")
	}

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":                    m.subscriptionId,
		"resource_group_name":                m.names.resourceGroup,
		"storage_account_name":               m.names.storageAccountName,
		"container_name":                     m.names.storageContainerName,
		"key":                                m.names.storageKeyName,
		"use_oidc":                           true,
		"oidc_request_token":                 oidcRequestToken,
		"oidc_request_url":                   oidcRequestURL,
		"ado_pipeline_service_connection_id": adoPipelineServiceConnectionId,
		"tenant_id":                          m.tenantId,
		"client_id":                          m.clientId,
		"environment":                        m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendAzureADAuthBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"tenant_id":            m.tenantId,
		"client_id":            m.clientId,
		"client_secret":        m.clientSecret,
		"use_azuread_auth":     true,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendManagedServiceIdentityBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackendRunningInAzure(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"use_msi":              true,
		"tenant_id":            m.tenantId,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendServicePrincipalClientCertificateBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	clientCertPassword := os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD")
	clientCertPath := os.Getenv("ARM_CLIENT_CERTIFICATE_PATH")
	if clientCertPath == "" {
		t.Skip("Skipping since `ARM_CLIENT_CERTIFICATE_PATH` is not specified!")
	}

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":             m.subscriptionId,
		"resource_group_name":         m.names.resourceGroup,
		"storage_account_name":        m.names.storageAccountName,
		"container_name":              m.names.storageContainerName,
		"key":                         m.names.storageKeyName,
		"tenant_id":                   m.tenantId,
		"client_id":                   m.clientId,
		"client_certificate_password": clientCertPassword,
		"client_certificate_path":     clientCertPath,
		"environment":                 m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendServicePrincipalClientSecretBasic(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"tenant_id":            m.tenantId,
		"client_id":            m.clientId,
		"client_secret":        m.clientSecret,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestAccBackendAccessKeyLocked(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"access_key":           m.storageAccessKey,
		"environment":          m.env.Name,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"access_key":           m.storageAccessKey,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)

	backend.TestBackendStateLocksInWS(t, b1, b2, "foo")
	backend.TestBackendStateForceUnlockInWS(t, b1, b2, "foo")
}

func TestAccBackendServicePrincipalLocked(t *testing.T) {
	t.Parallel()

	testAccAzureBackend(t)

	ctx := newCtx()
	m := BuildTestMeta(t, ctx)

	err := m.buildTestResources(ctx)
	if err != nil {
		m.destroyTestResources(ctx)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer m.destroyTestResources(ctx)

	clearARMEnv()

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"tenant_id":            m.tenantId,
		"client_id":            m.clientId,
		"client_secret":        m.clientSecret,
		"environment":          m.env.Name,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"subscription_id":      m.subscriptionId,
		"resource_group_name":  m.names.resourceGroup,
		"storage_account_name": m.names.storageAccountName,
		"container_name":       m.names.storageContainerName,
		"key":                  m.names.storageKeyName,
		"tenant_id":            m.tenantId,
		"client_id":            m.clientId,
		"client_secret":        m.clientSecret,
		"environment":          m.env.Name,
	})).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)

	backend.TestBackendStateLocksInWS(t, b1, b2, "foo")
	backend.TestBackendStateForceUnlockInWS(t, b1, b2, "foo")
}

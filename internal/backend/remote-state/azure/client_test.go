// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
)

var _ remote.Client = new(RemoteClient)
var _ remote.ClientLocker = new(RemoteClient)

func TestRemoteClientAccessKeyBasic(t *testing.T) {
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

	state, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientManagedServiceIdentityBasic(t *testing.T) {
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

	state, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientSasTokenBasic(t *testing.T) {
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

	state, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientServicePrincipalBasic(t *testing.T) {
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

	state, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientAccessKeyLocks(t *testing.T) {
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

	s1, sDiags := b1.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	s2, sDiags := b2.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

func TestRemoteClientServicePrincipalLocks(t *testing.T) {
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

	s1, sDiags := b1.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	s2, sDiags := b2.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

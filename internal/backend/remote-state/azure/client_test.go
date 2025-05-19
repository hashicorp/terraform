// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/jackofallops/giovanni/storage/2023-11-03/blob/blobs"
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

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
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

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
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

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
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

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
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

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
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

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

func TestPutMaintainsMetaData(t *testing.T) {
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

	headerName := "acceptancetest"
	expectedValue := "f3b56bad-33ad-4b93-a600-7a66e9cbd1eb"

	client, err := m.getBlobClient(ctx)
	if err != nil {
		t.Fatalf("Error building Blob Client: %+v", err)
	}

	_, err = client.PutBlockBlob(ctx, m.names.storageContainerName, m.names.storageKeyName, blobs.PutBlockBlobInput{})
	if err != nil {
		t.Fatalf("Error Creating Block Blob: %+v", err)
	}

	blobReference, err := client.GetProperties(ctx, m.names.storageContainerName, m.names.storageKeyName, blobs.GetPropertiesInput{})
	if err != nil {
		t.Fatalf("Error loading MetaData: %+v", err)
	}

	blobReference.MetaData[headerName] = expectedValue
	opts := blobs.SetMetaDataInput{
		MetaData: blobReference.MetaData,
	}
	_, err = client.SetMetaData(ctx, m.names.storageContainerName, m.names.storageKeyName, opts)
	if err != nil {
		t.Fatalf("Error setting MetaData: %+v", err)
	}

	// update the metadata using the Backend
	remoteClient := RemoteClient{
		keyName:       m.names.storageKeyName,
		containerName: m.names.storageContainerName,
		accountName:   m.names.storageAccountName,

		giovanniBlobClient: *client,
	}

	bytes := []byte(randString(20))
	err = remoteClient.Put(bytes)
	if err != nil {
		t.Fatalf("Error putting data: %+v", err)
	}

	// Verify it still exists
	blobReference, err = client.GetProperties(ctx, m.names.storageContainerName, m.names.storageKeyName, blobs.GetPropertiesInput{})
	if err != nil {
		t.Fatalf("Error loading MetaData: %+v", err)
	}

	if blobReference.MetaData[headerName] != expectedValue {
		t.Fatalf("%q was not set to %q in the MetaData: %+v", headerName, expectedValue, blobReference.MetaData)
	}
}

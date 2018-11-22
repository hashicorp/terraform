package azure

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := map[string]interface{}{
		"storage_account_name": "tfaccount",
		"container_name":       "tfcontainer",
		"key":                  "state",
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
}

func TestBackendAccessKeyBasic(t *testing.T) {
	testAccAzureBackend(t)
	rs := acctest.RandString(4)
	res := testResourceNames(rs, "testState")
	armClient := buildTestClient(t, res)

	ctx := context.TODO()
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		armClient.destroyTestResources(ctx, res)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer armClient.destroyTestResources(ctx, res)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"access_key":           res.storageAccountAccessKey,
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestBackendManagedServiceIdentityBasic(t *testing.T) {
	testAccAzureBackendRunningInAzure(t)
	rs := acctest.RandString(4)
	res := testResourceNames(rs, "testState")
	armClient := buildTestClient(t, res)

	ctx := context.TODO()
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		armClient.destroyTestResources(ctx, res)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer armClient.destroyTestResources(ctx, res)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"resource_group_name":  res.resourceGroup,
		"use_msi":              true,
		"arm_subscription_id":  os.Getenv("ARM_SUBSCRIPTION_ID"),
		"arm_tenant_id":        os.Getenv("ARM_TENANT_ID"),
		"environment":          os.Getenv("ARM_ENVIRONMENT"),
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestBackendServicePrincipalBasic(t *testing.T) {
	testAccAzureBackend(t)
	rs := acctest.RandString(4)
	res := testResourceNames(rs, "testState")
	armClient := buildTestClient(t, res)

	ctx := context.TODO()
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		armClient.destroyTestResources(ctx, res)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer armClient.destroyTestResources(ctx, res)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"resource_group_name":  res.resourceGroup,
		"arm_subscription_id":  os.Getenv("ARM_SUBSCRIPTION_ID"),
		"arm_tenant_id":        os.Getenv("ARM_TENANT_ID"),
		"arm_client_id":        os.Getenv("ARM_CLIENT_ID"),
		"arm_client_secret":    os.Getenv("ARM_CLIENT_SECRET"),
		"environment":          os.Getenv("ARM_ENVIRONMENT"),
	})).(*Backend)

	backend.TestBackendStates(t, b)
}

func TestBackendAccessKeyLocked(t *testing.T) {
	testAccAzureBackend(t)
	rs := acctest.RandString(4)
	res := testResourceNames(rs, "testState")
	armClient := buildTestClient(t, res)

	ctx := context.TODO()
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		armClient.destroyTestResources(ctx, res)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer armClient.destroyTestResources(ctx, res)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"access_key":           res.storageAccountAccessKey,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"access_key":           res.storageAccountAccessKey,
	})).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

func TestBackendServicePrincipalLocked(t *testing.T) {
	testAccAzureBackend(t)
	rs := acctest.RandString(4)
	res := testResourceNames(rs, "testState")
	armClient := buildTestClient(t, res)

	ctx := context.TODO()
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		armClient.destroyTestResources(ctx, res)
		t.Fatalf("Error creating Test Resources: %q", err)
	}
	defer armClient.destroyTestResources(ctx, res)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"access_key":           res.storageAccountAccessKey,
		"arm_subscription_id":  os.Getenv("ARM_SUBSCRIPTION_ID"),
		"arm_tenant_id":        os.Getenv("ARM_TENANT_ID"),
		"arm_client_id":        os.Getenv("ARM_CLIENT_ID"),
		"arm_client_secret":    os.Getenv("ARM_CLIENT_SECRET"),
		"environment":          os.Getenv("ARM_ENVIRONMENT"),
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"storage_account_name": res.storageAccountName,
		"container_name":       res.storageContainerName,
		"key":                  res.storageKeyName,
		"access_key":           res.storageAccountAccessKey,
		"arm_subscription_id":  os.Getenv("ARM_SUBSCRIPTION_ID"),
		"arm_tenant_id":        os.Getenv("ARM_TENANT_ID"),
		"arm_client_id":        os.Getenv("ARM_CLIENT_ID"),
		"arm_client_secret":    os.Getenv("ARM_CLIENT_SECRET"),
		"environment":          os.Getenv("ARM_ENVIRONMENT"),
	})).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

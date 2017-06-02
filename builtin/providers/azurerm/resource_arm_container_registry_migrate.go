package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAzureRMContainerRegistryMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AzureRM Container Registry State v0; migrating to v1")
		return migrateAzureRMContainerRegistryStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAzureRMContainerRegistryStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] ARM Container Registry Attributes before Migration: %#v", is.Attributes)

	is.Attributes["sku"] = "Basic"

	log.Printf("[DEBUG] ARM Container Registry Attributes after State Migration: %#v", is.Attributes)

	return is, nil
}

package azurerm

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAzureRMVirtualMachineMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AzureRM Virtual Machine State v0; migrating to v1")
		return migrateAzureRMVirtualMachineStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAzureRMVirtualMachineStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] AzureRM Virtual Machine Attributes before migration: %#v", is.Attributes)
	prefix := "os_profile_windows_config"
	entity := resourceArmVirtualMachine()

	// Read old keys
	reader := &schema.MapFieldReader{
		Schema: entity.Schema,
		Map:    schema.BasicMapReader(is.Attributes),
	}
	result, err := reader.ReadField([]string{prefix})
	if err != nil {
		return nil, err
	}

	oldKeys, ok := result.Value.(*schema.Set)
	if !ok {
		return nil, fmt.Errorf("Got unexpected value from state: %#v", result.Value)
	}

	for _, value := range oldKeys.List() {
		value := value.(map[string]interface{})
		_, exists := value["provision_vm_agent"]
		if !exists {
			value["provision_vm_agent"] = "true"
		}

		_, exists = value["enable_automatic_upgrades"]
		if !exists {
			value["enable_automatic_upgrades"] = "true"
		}
	}

	// Delete old keys
	for k := range is.Attributes {
		if strings.HasPrefix(k, fmt.Sprintf("%s.", prefix)) {
			delete(is.Attributes, k)
		}
	}

	// Write new keys
	writer := schema.MapFieldWriter{
		Schema: entity.Schema,
	}
	if err := writer.WriteField([]string{prefix}, oldKeys); err != nil {
		return is, err
	}
	for k, v := range writer.Map() {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] AzureRM Virtual Machine Attributes after State Migration: %#v", is.Attributes)

	return is, nil
}

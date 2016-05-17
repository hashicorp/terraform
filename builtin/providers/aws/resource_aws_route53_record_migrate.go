package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsRoute53RecordMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Route53 Record State v0; migrating to v1")
		return migrateRoute53RecordStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateRoute53RecordStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	newName := strings.TrimSuffix(is.Attributes["name"], ".")
	is.Attributes["name"] = newName
	log.Printf("[DEBUG] Attributes after migration: %#v, new name: %s", is.Attributes, newName)
	return is, nil
}

package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceSqlUserMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	switch v {
	case 0:
		log.Println("[INFO] Found Google Sql User State v0; migrating to v1")
		is, err := migrateSqlUserStateV0toV1(is)
		if err != nil {
			return is, err
		}
		return is, nil
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateSqlUserStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	name := is.Attributes["name"]
	instance := is.Attributes["instance"]
	is.ID = fmt.Sprintf("%s/%s", instance, name)

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

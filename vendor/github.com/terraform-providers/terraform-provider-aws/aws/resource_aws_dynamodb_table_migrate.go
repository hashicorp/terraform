package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsDynamoDbTableMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS DynamoDB Table State v0; migrating to v1")
		return migrateDynamoDBStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateDynamoDBStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] DynamoDB Table Attributes before Migration: %#v", is.Attributes)

	prefix := "global_secondary_index"
	entity := resourceAwsDynamoDbTable()

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

	log.Printf("[DEBUG] DynamoDB Table Attributes after State Migration: %#v", is.Attributes)

	return is, nil
}

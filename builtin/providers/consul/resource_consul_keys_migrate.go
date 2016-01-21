package consul

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func resourceConsulKeysMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found consul_keys State v0; migrating to v1")
		return resourceConsulKeysMigrateStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func resourceConsulKeysMigrateStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	res := resourceConsulKeys()
	keys, err := readV0Keys(is, res)
	if err != nil {
		return is, err
	}
	if err := clearV0Keys(is); err != nil {
		return is, err
	}
	if err := writeV1Keys(is, res, keys); err != nil {
		return is, err
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

func readV0Keys(
	is *terraform.InstanceState,
	res *schema.Resource,
) (*schema.Set, error) {
	reader := &schema.MapFieldReader{
		Schema: res.Schema,
		Map:    schema.BasicMapReader(is.Attributes),
	}
	result, err := reader.ReadField([]string{"key"})
	if err != nil {
		return nil, err
	}

	oldKeys, ok := result.Value.(*schema.Set)
	if !ok {
		return nil, fmt.Errorf("Got unexpected value from state: %#v", result.Value)
	}
	return oldKeys, nil
}

func clearV0Keys(is *terraform.InstanceState) error {
	for k := range is.Attributes {
		if strings.HasPrefix(k, "key.") {
			delete(is.Attributes, k)
		}
	}
	return nil
}

func writeV1Keys(
	is *terraform.InstanceState,
	res *schema.Resource,
	keys *schema.Set,
) error {
	writer := schema.MapFieldWriter{
		Schema: res.Schema,
	}
	if err := writer.WriteField([]string{"key"}, keys); err != nil {
		return err
	}
	for k, v := range writer.Map() {
		is.Attributes[k] = v
	}

	return nil
}

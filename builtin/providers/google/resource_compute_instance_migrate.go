package google

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

func resourceComputeInstanceMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	switch v {
	case 0:
		log.Println("[INFO] Found Compute Instance State v0; migrating to v1")
		return migrateStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	// Delete old count
	delete(is.Attributes, "metadata.#")

	newMetadata := make(map[string]string)

	for k, v := range is.Attributes {
		if !strings.HasPrefix(k, "metadata.") {
			continue
		}

		// We have a key that looks like "metadata.*" and we know it's not
		// metadata.# because we deleted it above, so it must be metadata.<N>.<key>
		// from the List of Maps. Just need to convert it to a single Map by
		// ditching the '<N>' field.
		kParts := strings.SplitN(k, ".", 3)

		// Sanity check: all three parts should be there and <N> should be a number
		badFormat := false
		if len(kParts) != 3 {
			badFormat = true
		} else if _, err := strconv.Atoi(kParts[1]); err != nil {
			badFormat = true
		}

		if badFormat {
			return is, fmt.Errorf(
				"migration error: found metadata key in unexpected format: %s", k)
		}

		// Rejoin as "metadata.<key>"
		newK := strings.Join([]string{kParts[0], kParts[2]}, ".")
		newMetadata[newK] = v
		delete(is.Attributes, k)
	}

	for k, v := range newMetadata {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

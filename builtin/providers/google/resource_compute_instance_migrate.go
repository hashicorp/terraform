package google

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
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
		is, err := migrateStateV0toV1(is)
		if err != nil {
			return is, err
		}
		fallthrough
	case 1:
		log.Println("[INFO] Found Compute Instance State v1; migrating to v2")
		is, err := migrateStateV1toV2(is)
		if err != nil {
			return is, err
		}
		return is, nil
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

func migrateStateV1toV2(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	// Maps service account index to list of scopes for that sccount
	newScopesMap := make(map[string][]string)

	for k, v := range is.Attributes {
		if !strings.HasPrefix(k, "service_account.") {
			continue
		}

		if k == "service_account.#" {
			continue
		}

		if strings.HasSuffix(k, ".scopes.#") {
			continue
		}

		if strings.HasSuffix(k, ".email") {
			continue
		}

		// Key is now of the form service_account.%d.scopes.%d
		kParts := strings.Split(k, ".")

		// Sanity check: all three parts should be there and <N> should be a number
		badFormat := false
		if len(kParts) != 4 {
			badFormat = true
		} else if _, err := strconv.Atoi(kParts[1]); err != nil {
			badFormat = true
		}

		if badFormat {
			return is, fmt.Errorf(
				"migration error: found scope key in unexpected format: %s", k)
		}

		newScopesMap[kParts[1]] = append(newScopesMap[kParts[1]], v)

		delete(is.Attributes, k)
	}

	for service_acct_index, newScopes := range newScopesMap {
		for _, newScope := range newScopes {
			hash := hashcode.String(canonicalizeServiceScope(newScope))
			newKey := fmt.Sprintf("service_account.%s.scopes.%d", service_acct_index, hash)
			is.Attributes[newKey] = newScope
		}
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

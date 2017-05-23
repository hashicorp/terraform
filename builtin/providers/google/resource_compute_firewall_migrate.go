package google

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

func resourceComputeFirewallMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	switch v {
	case 0:
		log.Println("[INFO] Found Compute Firewall State v0; migrating to v1")
		is, err := migrateFirewallStateV0toV1(is)
		if err != nil {
			return is, err
		}
		return is, nil
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateFirewallStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	idx := 0
	portCount := 0
	newPorts := make(map[string]string)
	keys := make([]string, len(is.Attributes))
	for k, _ := range is.Attributes {
		keys[idx] = k
		idx++

	}
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.HasPrefix(k, "allow.") {
			continue
		}

		if k == "allow.#" {
			continue
		}

		if strings.HasSuffix(k, ".ports.#") {
			continue
		}

		if strings.HasSuffix(k, ".protocol") {
			continue
		}

		// We have a key that looks like "allow.<hash>.ports.*" and we know it's not
		// allow.<hash>.ports.# because we deleted it above, so it must be allow.<hash1>.ports.<hash2>
		// from the Set of Ports. Just need to convert it to a list by
		// replacing second hash with sequential numbers.
		kParts := strings.Split(k, ".")

		// Sanity check: all four parts should be there and <hash> should be a number
		badFormat := false
		if len(kParts) != 4 {
			badFormat = true
		} else if _, err := strconv.Atoi(kParts[1]); err != nil {
			badFormat = true
		}

		if badFormat {
			return is, fmt.Errorf(
				"migration error: found port key in unexpected format: %s", k)
		}
		allowHash, _ := strconv.Atoi(kParts[1])
		newK := fmt.Sprintf("allow.%d.ports.%d", allowHash, portCount)
		portCount++
		newPorts[newK] = is.Attributes[k]
		delete(is.Attributes, k)
	}

	for k, v := range newPorts {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

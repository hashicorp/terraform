package aws

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsCloudFrontDistributionMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS CloudFront Distribution State v0; migrating to v1")
		return migrateCloudFrontDistributionStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateCloudFrontDistributionStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty CloudFront Distribution State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	var fieldIndex = make(map[string]int, 0)
	var hashToIdx = make(map[string]string, 0)
	var newAttributes = make(map[string]string, 0)
	computeHash := schema.HashSchema(&schema.Schema{Type: schema.TypeString})

	var keys []string
	for k, _ := range is.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := is.Attributes[k]
		if strings.HasPrefix(k, "default_cache_behavior") || strings.HasPrefix(k, "cache_behavior") {
			oldKey := strings.Split(k, ".")

			// Skip counts
			if oldKey[1] == "#" {
				continue
			}

			var newKey = make([]string, len(oldKey))
			copy(newKey, oldKey)

			// From default_cache_behavior.1784951082.allowed_methods.0
			// To 	default_cache_behavior.0.allowed_methods.1040875975

			// TypeSet -> TypeList
			if idx, ok := hashToIdx[oldKey[1]]; ok {
				newKey[1] = idx
			} else {
				newIndex := fmt.Sprintf("%d", fieldIndex[oldKey[0]])
				hashToIdx[oldKey[1]] = newIndex
				fieldIndex[oldKey[0]] += 1
				newKey[1] = newIndex
			}

			// TypeList -> TypeSet
			if len(oldKey) == 4 && (oldKey[2] == "allowed_methods" || oldKey[2] == "cached_methods") && oldKey[3] != "#" {
				newKey[3] = strconv.Itoa(computeHash(v))
			}

			newKeyString := strings.Join(newKey, ".")

			log.Printf("[DEBUG] Deleting %s", k)
			delete(is.Attributes, k)
			log.Printf("[DEBUG] Adding %s", newKeyString)
			newAttributes[newKeyString] = v
		}
	}
	for k, v := range newAttributes {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

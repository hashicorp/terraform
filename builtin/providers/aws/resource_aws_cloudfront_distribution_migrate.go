package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var cacheBehaviorExp = regexp.MustCompile(`^cache_behavior\.([0-9]+)\.`)
var cacheBehaviorFormat = "cache_behavior.%d."
var httpMethodExp = regexp.MustCompile(`\.(?P<allowedOrCached>allowed|cached)_methods\.[0-9]+`)
var httpMethodFormat = `.${allowedOrCached}_methods.%d`

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

	var hashToIdx = make(map[string]int, 0)
	var newAttributes = make(map[string]string, 0)

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	for k, v := range is.Attributes {

		// change cache_behavior.1784951082 to cache_behavior.0
		newKey := migrateCacheBehaviorKeyFromSetToList(&hashToIdx, k)

		// change allowed_methods.0 to allowed_methods.1040875975
		newKey = migrateHttpMethodsKeysFromListToSet(newKey, v)

		log.Printf("[DEBUG] Deleting %s", k)
		delete(is.Attributes, k)
		log.Printf("[DEBUG] Adding %s", newKey)
		newAttributes[newKey] = v
	}
	is.Attributes = newAttributes

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

func migrateCacheBehaviorKeyFromSetToList(hashToIdx *map[string]int, k string) string {
	subMatchGroups := cacheBehaviorExp.FindStringSubmatch(k)
	if subMatchGroups != nil {
		var newIndex int
		hash := subMatchGroups[1]
		if idx, ok := (*hashToIdx)[hash]; ok {
			newIndex = idx
		} else {
			newIndex = len(*hashToIdx)
			(*hashToIdx)[hash] = newIndex
		}
		return cacheBehaviorExp.ReplaceAllLiteralString(k, fmt.Sprintf(cacheBehaviorFormat, newIndex))
	}
	return k
}

func migrateHttpMethodsKeysFromListToSet(k string, v string) string {
	computeHash := schema.HashSchema(&schema.Schema{Type: schema.TypeString})
	subMatchGroups := httpMethodExp.FindStringSubmatch(k)
	if subMatchGroups != nil {
		hash := computeHash(v)
		return httpMethodExp.ReplaceAllString(k, fmt.Sprintf(httpMethodFormat, hash))
	}
	return k
}

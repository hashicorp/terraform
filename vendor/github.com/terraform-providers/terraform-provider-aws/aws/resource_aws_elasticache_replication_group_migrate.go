package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsElasticacheReplicationGroupMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Elasticache Replication Group State v0; migrating to v1")
		return migrateAwsElasticacheReplicationGroupStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAwsElasticacheReplicationGroupStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	numCountStr, ok := is.Attributes["cluster_mode.#"]
	if !ok || numCountStr == "0" {
		log.Println("[DEBUG] Empty cluster_mode in InstanceState; no need to migrate.")
		return is, nil
	}

	for k, v := range is.Attributes {
		if !strings.HasPrefix(k, "cluster_mode.") || strings.HasPrefix(k, "cluster_mode.#") {
			continue
		}

		// cluster_mode.HASHCODE.attr
		path := strings.Split(k, ".")
		if len(path) != 3 {
			return is, fmt.Errorf("Found unexpected cluster_mode field: %#v", k)
		}
		hashcode, attr := path[1], path[2]
		if hashcode == "0" {
			// Skip already migrated attribute
			continue
		}

		if attr == "replicas_per_node_group" {
			is.Attributes["cluster_mode.0.replicas_per_node_group"] = v
		}
		if attr == "num_node_groups" {
			is.Attributes["cluster_mode.0.num_node_groups"] = v
		}
		delete(is.Attributes, k)
	}
	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

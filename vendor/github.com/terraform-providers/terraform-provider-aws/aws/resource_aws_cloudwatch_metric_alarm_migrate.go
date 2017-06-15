package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsCloudWatchMetricAlarmMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS CloudWatch Metric Alarm State v0; migrating to v1")
		return migrateCloudWatchMetricAlarmStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateCloudWatchMetricAlarmStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	is.Attributes["treat_missing_data"] = "missing"

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

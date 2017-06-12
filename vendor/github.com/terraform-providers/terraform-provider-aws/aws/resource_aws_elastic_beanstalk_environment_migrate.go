package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsElasticBeanstalkEnvironmentMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Elastic Beanstalk Environment State v0; migrating to v1")
		return migrateBeanstalkEnvironmentStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateBeanstalkEnvironmentStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty Elastic Beanstalk Environment State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	if is.Attributes["tier"] == "" {
		is.Attributes["tier"] = "WebServer"
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}

package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsEcsTaskDefinitionMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	conn := meta.(*AWSClient).ecsconn

	switch v {
	case 0:
		log.Println("[INFO] Found AWS ECS Task Definition State v0; migrating to v1")
		return migrateEcsTaskDefinitionStateV0toV1(is, conn)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateEcsTaskDefinitionStateV0toV1(is *terraform.InstanceState, conn *ecs.ECS) (*terraform.InstanceState, error) {
	arn := is.Attributes["arn"]

	// We need to pull definitions from the API b/c they're unrecoverable from the checksum
	td, err := conn.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}

	b, err := jsonutil.BuildJSON(td.TaskDefinition.ContainerDefinitions)
	if err != nil {
		return nil, err
	}

	is.Attributes["container_definitions"] = string(b)

	return is, nil
}

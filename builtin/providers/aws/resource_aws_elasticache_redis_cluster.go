package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticacheRedisCluster() *schema.Resource {

	resourceSchema := resourceAwsElasticacheReplicationGroupCommon()

	resourceSchema["automatic_failover_enabled"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
	}

	resourceSchema["number_cache_clusters"] = &schema.Schema{
		Type:     schema.TypeInt,
		Computed: true,
		ForceNew: true,
	}

	resourceSchema["num_node_groups"] = &schema.Schema{
		Type:     schema.TypeInt,
		Required: true,
		ForceNew: true,
	}

	resourceSchema["replicas_per_node_group"] = &schema.Schema{
		Type:     schema.TypeInt,
		Required: true,
		ForceNew: true,
	}

	return &schema.Resource{
		Create: resourceAwsElasticacheRedisClusterCreate,
		Read:   resourceAwsElasticacheReplicationGroupRead,
		Update: resourceAwsElasticacheReplicationGroupUpdate,
		Delete: resourceAwsElasticacheReplicationGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: resourceSchema,
	}
}

func resourceAwsElasticacheRedisClusterCreate(d *schema.ResourceData, meta interface{}) error {

	params := resourceAwsElasticacheReplicationGroupCreateSetup(d, meta)

	if v, ok := d.GetOk("num_node_groups"); ok {
		params.NumNodeGroups = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("replicas_per_node_group"); ok {
		params.ReplicasPerNodeGroup = aws.Int64(int64(v.(int)))
	}

	return resourceAwsElasticacheReplicationGroupCreateCommon(d, meta, params)
}

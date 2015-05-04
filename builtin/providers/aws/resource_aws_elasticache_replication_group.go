package aws

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticacheReplicationGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheReplicationGroupCreate,
		Read:   resourceAwsElasticacheReplicationGroupRead,
		Update: resourceAwsElasticacheReplicationGroupUpdate,
		Delete: resourceAwsElasticacheReplicationGroupDelete,

		Schema: map[string]*schema.Schema{
			"replication_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cache_node_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"automatic_failover": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"num_cache_clusters": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "redis",
			},
			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsElasticacheReplicationGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	replicationGroupId := d.Get("replication_group_id").(string)
	description := d.Get("description").(string)
	cacheNodeType := d.Get("cache_node_type").(string)
	automaticFailover := d.Get("automatic_failover").(bool)
	numCacheClusters := d.Get("num_cache_clusters").(int)
	parameterGroupName := d.Get("parameter_group_name").(string)
	engine := d.Get("engine").(string)
	engineVersion := d.Get("engine_version").(string)

	req := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupID:		aws.String(replicationGroupId),
		ReplicationGroupDescription:	aws.String(description),
		CacheNodeType:            	aws.String(cacheNodeType),
		AutomaticFailoverEnabled: 	aws.Boolean(automaticFailover),
		NumCacheClusters:         	aws.Long(int64(numCacheClusters)),
		CacheParameterGroupName:	aws.String(parameterGroupName),
		Engine:				aws.String(engine),
		EngineVersion:			aws.String(engineVersion),
	}

	_, err := conn.CreateReplicationGroup(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache replication group: %s", err)
	}

	d.SetId(replicationGroupId)

	return nil
}

func resourceAwsElasticacheReplicationGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	req := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupID: aws.String(d.Id()),
	}

	res, err := conn.DescribeReplicationGroups(req)
	if err != nil {
		return err
	}

	if len(res.ReplicationGroups) == 1 {
		c := res.ReplicationGroups[0]
		d.Set("replication_group_id", c.ReplicationGroupID)
		d.Set("description", c.Description)
		d.Set("automatic_failover", c.AutomaticFailover)
	}

	return nil
}

func resourceAwsElasticacheReplicationGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.ModifyReplicationGroupInput{
		ApplyImmediately:   aws.Boolean(true),
		ReplicationGroupID: aws.String(d.Id()),
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		req.ReplicationGroupDescription = aws.String(description)
	}

	if d.HasChange("automatic_failover") {
		automaticFailover := d.Get("automatic_failover").(bool)
		req.AutomaticFailoverEnabled = aws.Boolean(automaticFailover)
	}

	_, err := conn.ModifyReplicationGroup(req)
	if err != nil {
		d.Partial(true)
		return fmt.Errorf("Error updating Elasticache replication group: %s", err)
	}

	return resourceAwsElasticacheReplicationGroupRead(d, meta)
}

func resourceAwsElasticacheReplicationGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupID: aws.String(d.Id()),
	}

	_, err := conn.DeleteReplicationGroupRequest(req)
	if err != nil {
		return fmt.Errorf("Error deleting Elasticache replication group: %s", err)
	}

	return nil
}

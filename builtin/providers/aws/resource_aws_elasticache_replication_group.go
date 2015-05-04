package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
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

	var hasParameterGroup bool

	replicationGroupId := d.Get("replication_group_id").(string)
	description := d.Get("description").(string)
	cacheNodeType := d.Get("cache_node_type").(string)
	automaticFailover := d.Get("automatic_failover").(bool)
	numCacheClusters := d.Get("num_cache_clusters").(int)
	parameterGroupName, hasParameterGroup := d.GetOk("parameter_group_name")
	engine := d.Get("engine").(string)
	engineVersion := d.Get("engine_version").(string)

	req := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupID:		aws.String(replicationGroupId),
		ReplicationGroupDescription:	aws.String(description),
		CacheNodeType:            	aws.String(cacheNodeType),
		AutomaticFailoverEnabled: 	aws.Boolean(automaticFailover),
		NumCacheClusters:         	aws.Long(int64(numCacheClusters)),
		Engine:				aws.String(engine),
		EngineVersion:			aws.String(engineVersion),
	}

	if hasParameterGroup {
		req.CacheParameterGroupName = aws.String(parameterGroupName.(string))
	}

	_, err := conn.CreateReplicationGroup(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache replication group: %s", err)
	}

	pending := []string{"creating"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     "available",
		Refresh:    ReplicationGroupStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    20 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to be created: %s", d.Id(), sterr)
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

	_, err := conn.DeleteReplicationGroup(req)
	if err != nil {
		return fmt.Errorf("Error deleting Elasticache replication group: %s", err)
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting"},
		Target:     "",
		Refresh:    ReplicationGroupStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    20 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for replication group (%s) to delete: %s", d.Id(), sterr)
	}

	return nil
}

func ReplicationGroupStateRefreshFunc(conn *elasticache.ElastiCache, replicationGroupID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		// log.Printf("[XXX] Given state: %s", givenState)
		resp, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupID: aws.String(replicationGroupID),
		})
		if err != nil {
			apierr := err.(aws.APIError)
			log.Printf("[DEBUG] message: %v, code: %v", apierr.Message, apierr.Code)
			if apierr.Message == fmt.Sprintf("ReplicationGroup not found: %v", replicationGroupID) {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROR] ReplicationGroupStateRefreshFunc: %s", err)
			return nil, "", err
		}

		c := resp.ReplicationGroups[0]
		log.Printf("[DEBUG] status: %v", *c.Status)

		// return the current state if it's in the pending array
		for _, p := range pending {
			s := *c.Status
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *c.Status)
				return c, p, nil
			}
		}

		// return given state if it's not in pending
		if givenState != "" {
			return c, givenState, nil
		}
		log.Printf("[DEBUG] current status: %v", *c.Status)
		return c, *c.Status, nil
	}
}

package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
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
				ForceNew: true,
			},
			"automatic_failover": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"num_cache_clusters": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"primary_cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
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
			"primary_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"preferred_cache_cluster_azs": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsElasticacheReplicationGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	replicationGroupID := d.Get("replication_group_id").(string)
	description := d.Get("description").(string)
	cacheNodeType := d.Get("cache_node_type").(string)
	automaticFailover := d.Get("automatic_failover").(bool)
	numCacheClusters := d.Get("num_cache_clusters").(int)
	primaryClusterID := d.Get("primary_cluster_id").(string)
	engine := d.Get("engine").(string)
	engineVersion := d.Get("engine_version").(string)
	securityNameSet := d.Get("security_group_names").(*schema.Set)
	securityIDSet := d.Get("security_group_ids").(*schema.Set)
	subnetGroupName := d.Get("subnet_group_name").(string)
	prefferedCacheClusterAZs := d.Get("preferred_cache_cluster_azs").(*schema.Set)

	securityNames := expandStringList(securityNameSet.List())
	securityIds := expandStringList(securityIDSet.List())
	prefferedAZs := expandStringList(prefferedCacheClusterAZs.List())

	req := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupID:          aws.String(replicationGroupID),
		ReplicationGroupDescription: aws.String(description),
		CacheNodeType:               aws.String(cacheNodeType),
		AutomaticFailoverEnabled:    aws.Bool(automaticFailover),
		NumCacheClusters:            aws.Int64(int64(numCacheClusters)),
		PrimaryClusterID:            aws.String(primaryClusterID),
		Engine:                      aws.String(engine),
		CacheSubnetGroupName:        aws.String(subnetGroupName),
		EngineVersion:               aws.String(engineVersion),
		CacheSecurityGroupNames:     securityNames,
		SecurityGroupIDs:            securityIds,
		PreferredCacheClusterAZs:    prefferedAZs,
	}

	// parameter groups are optional and can be defaulted by AWS
	if v, ok := d.GetOk("parameter_group_name"); ok {
		req.CacheParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("maintenance_window"); ok {
		req.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	_, err := conn.CreateReplicationGroup(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache replication group: %s", err)
	}

	d.SetId(replicationGroupID)

	pending := []string{"creating"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     "available",
		Refresh:    replicationGroupStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    60 * time.Minute,
		Delay:      20 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to be created: %s", d.Id(), sterr)
	}

	return nil
}

func resourceAwsElasticacheReplicationGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupID: aws.String(d.Id()),
	}

	res, err := conn.DescribeReplicationGroups(req)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ReplicationGroupNotFoundFault" {
			// Update state to indicate the replication group no longer exists.
			d.SetId("")
			return nil
		}

		return err
	}

	if len(res.ReplicationGroups) == 1 {
		c := res.ReplicationGroups[0]
		d.Set("replication_group_id", c.ReplicationGroupID)
		d.Set("description", c.Description)
		d.Set("automatic_failover", c.AutomaticFailover)
		d.Set("num_cache_clusters", len(c.MemberClusters))
		d.Set("primary_endpoint", res.ReplicationGroups[0].NodeGroups[0].PrimaryEndpoint.Address)
	}

	return nil
}

func resourceAwsElasticacheReplicationGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.ModifyReplicationGroupInput{
		ApplyImmediately:   aws.Bool(true),
		ReplicationGroupID: aws.String(d.Id()),
	}

	if d.HasChange("automatic_failover") {
		automaticFailover := d.Get("automatic_failover").(bool)
		req.AutomaticFailoverEnabled = aws.Bool(automaticFailover)
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		req.ReplicationGroupDescription = aws.String(description)
	}

	if d.HasChange("engine_version") {
		engineVersion := d.Get("engine_version").(string)
		req.EngineVersion = aws.String(engineVersion)
	}

	if d.HasChange("security_group_ids") {
		securityIDSet := d.Get("security_group_ids").(*schema.Set)
		securityIds := expandStringList(securityIDSet.List())
		req.SecurityGroupIDs = securityIds
	}

	if d.HasChange("security_group_names") {
		securityNameSet := d.Get("security_group_names").(*schema.Set)
		securityNames := expandStringList(securityNameSet.List())
		req.CacheSecurityGroupNames = securityNames
	}

	_, err := conn.ModifyReplicationGroup(req)
	if err != nil {
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
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ReplicationGroupNotFoundFault" {
			// Update state to indicate the replication group no longer exists.
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error deleting Elasticache replication group: %s", err)
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting"},
		Target:     "",
		Refresh:    replicationGroupStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    15 * time.Minute,
		Delay:      20 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for replication group (%s) to delete: %s", d.Id(), sterr)
	}

	return nil
}

func replicationGroupStateRefreshFunc(conn *elasticache.ElastiCache, replicationGroupID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupID: aws.String(replicationGroupID),
		})
		if err != nil {
			ec2err, ok := err.(awserr.Error)

			if ok {
				log.Printf("[DEBUG] message: %v, code: %v", ec2err.Message(), ec2err.Code())
				if ec2err.Code() == "ReplicationGroupNotFoundFault" {
					log.Printf("[DEBUG] Detect deletion")
					return nil, "", nil
				}
			}

			log.Printf("[ERROR] replicationGroupStateRefreshFunc: %s", err)
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

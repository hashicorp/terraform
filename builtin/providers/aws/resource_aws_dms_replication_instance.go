package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDmsReplicationInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDmsReplicationInstanceCreate,
		Read:   resourceAwsDmsReplicationInstanceRead,
		Update: resourceAwsDmsReplicationInstanceUpdate,
		Delete: resourceAwsDmsReplicationInstanceDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"allocated_storage": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(5, 6144),
			},
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"multi_az": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"preferred_maintenance_window": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validateOnceAWeekWindowFormat,
			},
			"publicly_accessible": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"replication_instance_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"replication_instance_class": {
				Type:     schema.TypeString,
				Required: true,
				// Valid Values: dms.t2.micro | dms.t2.small | dms.t2.medium | dms.t2.large | dms.c4.large |
				// dms.c4.xlarge | dms.c4.2xlarge | dms.c4.4xlarge
			},
			"replication_instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDmsReplicationInstanceId,
			},
			"replication_instance_private_ips": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"replication_instance_public_ips": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"replication_subnet_group_id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsDmsReplicationInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.CreateReplicationInstanceInput{
		AutoMinorVersionUpgrade:       aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		PubliclyAccessible:            aws.Bool(d.Get("publicly_accessible").(bool)),
		ReplicationInstanceClass:      aws.String(d.Get("replication_instance_class").(string)),
		ReplicationInstanceIdentifier: aws.String(d.Get("replication_instance_id").(string)),
		Tags: dmsTagsFromMap(d.Get("tags").(map[string]interface{})),
	}

	// WARNING: GetOk returns the zero value for the type if the key is omitted in config. This means for optional
	// keys that the zero value is valid we cannot know if the zero value was in the config and cannot allow the API
	// to set the default value. See GitHub Issue #5694 https://github.com/hashicorp/terraform/issues/5694

	if v, ok := d.GetOk("allocated_storage"); ok {
		request.AllocatedStorage = aws.Int64(int64(v.(int)))
	}
	if v, ok := d.GetOk("engine_version"); ok {
		request.EngineVersion = aws.String(v.(string))
	}
	if v, ok := d.GetOk("kms_key_arn"); ok {
		request.KmsKeyId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("preferred_maintenance_window"); ok {
		request.PreferredMaintenanceWindow = aws.String(v.(string))
	}
	if v, ok := d.GetOk("replication_subnet_group_id"); ok {
		request.ReplicationSubnetGroupIdentifier = aws.String(v.(string))
	}
	if v, ok := d.GetOk("vpc_security_group_ids"); ok {
		request.VpcSecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}

	az, azSet := d.GetOk("availability_zone")
	if azSet {
		request.AvailabilityZone = aws.String(az.(string))
	}

	if multiAz, ok := d.GetOk("multi_az"); ok {
		request.MultiAZ = aws.Bool(multiAz.(bool))

		if multiAz.(bool) && azSet {
			return fmt.Errorf("Cannot set availability_zone if multi_az is set to true")
		}
	}

	log.Println("[DEBUG] DMS create replication instance:", request)

	_, err := conn.CreateReplicationInstance(request)
	if err != nil {
		return err
	}

	d.SetId(d.Get("replication_instance_id").(string))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    resourceAwsDmsReplicationInstanceStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDmsReplicationInstanceRead(d, meta)
}

func resourceAwsDmsReplicationInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	response, err := conn.DescribeReplicationInstances(&dms.DescribeReplicationInstancesInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-instance-id"),
				Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
			},
		},
	})
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Instance %q Not Found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = resourceAwsDmsReplicationInstanceSetState(d, response.ReplicationInstances[0])
	if err != nil {
		return err
	}

	tagsResp, err := conn.ListTagsForResource(&dms.ListTagsForResourceInput{
		ResourceArn: aws.String(d.Get("replication_instance_arn").(string)),
	})
	if err != nil {
		return err
	}
	d.Set("tags", dmsTagsToMap(tagsResp.TagList))

	return nil
}

func resourceAwsDmsReplicationInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	request := &dms.ModifyReplicationInstanceInput{
		ApplyImmediately:       aws.Bool(d.Get("apply_immediately").(bool)),
		ReplicationInstanceArn: aws.String(d.Get("replication_instance_arn").(string)),
	}
	hasChanges := false

	if d.HasChange("auto_minor_version_upgrade") {
		request.AutoMinorVersionUpgrade = aws.Bool(d.Get("auto_minor_version_upgrade").(bool))
		hasChanges = true
	}

	if d.HasChange("allocated_storage") {
		if v, ok := d.GetOk("allocated_storage"); ok {
			request.AllocatedStorage = aws.Int64(int64(v.(int)))
			hasChanges = true
		}
	}

	if d.HasChange("engine_version") {
		if v, ok := d.GetOk("engine_version"); ok {
			request.ReplicationInstanceClass = aws.String(v.(string))
			hasChanges = true
		}
	}

	if d.HasChange("multi_az") {
		if v, ok := d.GetOk("multi_az"); ok {
			request.MultiAZ = aws.Bool(v.(bool))
			hasChanges = true
		}
	}

	if d.HasChange("preferred_maintenance_window") {
		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			request.PreferredMaintenanceWindow = aws.String(v.(string))
			hasChanges = true
		}
	}

	if d.HasChange("replication_instance_class") {
		if v, ok := d.GetOk("replication_instance_class"); ok {
			request.ReplicationInstanceClass = aws.String(v.(string))
			hasChanges = true
		}
	}

	if d.HasChange("vpc_security_group_ids") {
		if v, ok := d.GetOk("vpc_security_group_ids"); ok {
			request.VpcSecurityGroupIds = expandStringList(v.(*schema.Set).List())
			hasChanges = true
		}
	}

	if d.HasChange("tags") {
		err := dmsSetTags(d.Get("replication_instance_arn").(string), d, meta)
		if err != nil {
			return err
		}
	}

	if hasChanges {
		conn := meta.(*AWSClient).dmsconn

		_, err := conn.ModifyReplicationInstance(request)
		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"available"},
			Refresh:    resourceAwsDmsReplicationInstanceStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

		return resourceAwsDmsReplicationInstanceRead(d, meta)
	}

	return nil
}

func resourceAwsDmsReplicationInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.DeleteReplicationInstanceInput{
		ReplicationInstanceArn: aws.String(d.Get("replication_instance_arn").(string)),
	}

	log.Printf("[DEBUG] DMS delete replication instance: %#v", request)

	_, err := conn.DeleteReplicationInstance(request)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDmsReplicationInstanceStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsDmsReplicationInstanceSetState(d *schema.ResourceData, instance *dms.ReplicationInstance) error {
	d.SetId(*instance.ReplicationInstanceIdentifier)

	d.Set("replication_instance_id", instance.ReplicationInstanceIdentifier)
	d.Set("allocated_storage", instance.AllocatedStorage)
	d.Set("auto_minor_version_upgrade", instance.AutoMinorVersionUpgrade)
	d.Set("availability_zone", instance.AvailabilityZone)
	d.Set("engine_version", instance.EngineVersion)
	d.Set("kms_key_arn", instance.KmsKeyId)
	d.Set("multi_az", instance.MultiAZ)
	d.Set("preferred_maintenance_window", instance.PreferredMaintenanceWindow)
	d.Set("publicly_accessible", instance.PubliclyAccessible)
	d.Set("replication_instance_arn", instance.ReplicationInstanceArn)
	d.Set("replication_instance_class", instance.ReplicationInstanceClass)
	d.Set("replication_subnet_group_id", instance.ReplicationSubnetGroup.ReplicationSubnetGroupIdentifier)

	vpc_security_group_ids := []string{}
	for _, sg := range instance.VpcSecurityGroups {
		vpc_security_group_ids = append(vpc_security_group_ids, aws.StringValue(sg.VpcSecurityGroupId))
	}

	d.Set("vpc_security_group_ids", vpc_security_group_ids)

	private_ip_addresses := []string{}
	for _, ip := range instance.ReplicationInstancePrivateIpAddresses {
		private_ip_addresses = append(private_ip_addresses, aws.StringValue(ip))
	}

	d.Set("replication_instance_private_ips", private_ip_addresses)

	public_ip_addresses := []string{}
	for _, ip := range instance.ReplicationInstancePublicIpAddresses {
		public_ip_addresses = append(public_ip_addresses, aws.StringValue(ip))
	}

	d.Set("replication_instance_public_ips", public_ip_addresses)

	return nil
}

func resourceAwsDmsReplicationInstanceStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).dmsconn

		v, err := conn.DescribeReplicationInstances(&dms.DescribeReplicationInstancesInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-instance-id"),
					Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
				},
			},
		})
		if err != nil {
			if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
				return nil, "", nil
			}
			log.Printf("Error on retrieving DMS Replication Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.ReplicationInstances == nil {
			return nil, "", fmt.Errorf("Error on retrieving DMS Replication Instance when waiting for State")
		}

		return v, *v.ReplicationInstances[0].ReplicationInstanceStatus, nil
	}
}

package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNeptuneClusterInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNeptuneClusterInstanceCreate,
		Read:   resourceAwsNeptuneClusterInstanceRead,
		Update: resourceAwsNeptuneClusterInstanceUpdate,
		Delete: resourceAwsNeptuneClusterInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(90 * time.Minute),
			Update: schema.DefaultTimeout(90 * time.Minute),
			Delete: schema.DefaultTimeout(90 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dbi_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "neptune",
				ForceNew:     true,
				ValidateFunc: validateNeptuneEngine(),
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"identifier": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"identifier_prefix"},
				ValidateFunc:  validateNeptuneIdentifier,
			},

			"identifier_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"identifier"},
				ValidateFunc:  validateNeptuneIdentifierPrefix,
			},

			"instance_class": {
				Type:     schema.TypeString,
				Required: true,
			},

			"kms_key_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"neptune_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default.neptune1",
			},

			"neptune_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  8182,
				ForceNew: true,
			},

			"preferred_backup_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},

			"promotion_tier": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"tags": tagsSchema(),

			"writer": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceAwsNeptuneClusterInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	tags := tagsFromMapNeptune(d.Get("tags").(map[string]interface{}))

	createOpts := &neptune.CreateDBInstanceInput{
		DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
		DBClusterIdentifier:     aws.String(d.Get("cluster_identifier").(string)),
		Engine:                  aws.String(d.Get("engine").(string)),
		PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
		PromotionTier:           aws.Int64(int64(d.Get("promotion_tier").(int))),
		AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		Tags:                    tags,
	}

	if attr, ok := d.GetOk("availability_zone"); ok {
		createOpts.AvailabilityZone = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("neptune_parameter_group_name"); ok {
		createOpts.DBParameterGroupName = aws.String(attr.(string))
	}

	if v, ok := d.GetOk("identifier"); ok {
		createOpts.DBInstanceIdentifier = aws.String(v.(string))
	} else {
		if v, ok := d.GetOk("identifier_prefix"); ok {
			createOpts.DBInstanceIdentifier = aws.String(resource.PrefixedUniqueId(v.(string)))
		} else {
			createOpts.DBInstanceIdentifier = aws.String(resource.PrefixedUniqueId("tf-"))
		}
	}

	if attr, ok := d.GetOk("neptune_subnet_group_name"); ok {
		createOpts.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("engine_version"); ok {
		createOpts.EngineVersion = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_backup_window"); ok {
		createOpts.PreferredBackupWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_maintenance_window"); ok {
		createOpts.PreferredMaintenanceWindow = aws.String(attr.(string))
	}

	log.Printf("[DEBUG] Creating Neptune Instance: %s", createOpts)

	var resp *neptune.CreateDBInstanceOutput
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = conn.CreateDBInstance(createOpts)
		if err != nil {
			if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating Neptune Instance: %s", err)
	}

	d.SetId(aws.StringValue(resp.DBInstance.DBInstanceIdentifier))

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsNeptuneClusterInstanceCreateUpdatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsNeptuneInstanceStateRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsNeptuneClusterInstanceRead(d, meta)
}

func resourceAwsNeptuneClusterInstanceRead(d *schema.ResourceData, meta interface{}) error {
	db, err := resourceAwsNeptuneInstanceRetrieve(d.Id(), meta.(*AWSClient).neptuneconn)
	if err != nil {
		return fmt.Errorf("Error on retrieving Neptune Cluster Instance (%s): %s", d.Id(), err)
	}

	if db == nil {
		log.Printf("[WARN] Neptune Cluster Instance (%s): not found, removing from state.", d.Id())
		d.SetId("")
		return nil
	}

	if db.DBClusterIdentifier == nil {
		return fmt.Errorf("Cluster identifier is missing from instance (%s)", d.Id())
	}

	conn := meta.(*AWSClient).neptuneconn
	resp, err := conn.DescribeDBClusters(&neptune.DescribeDBClustersInput{
		DBClusterIdentifier: db.DBClusterIdentifier,
	})

	var dbc *neptune.DBCluster
	for _, c := range resp.DBClusters {
		if aws.StringValue(c.DBClusterIdentifier) == aws.StringValue(db.DBClusterIdentifier) {
			dbc = c
		}
	}
	if dbc == nil {
		return fmt.Errorf("Error finding Neptune Cluster (%s) for Cluster Instance (%s): %s",
			aws.StringValue(db.DBClusterIdentifier), aws.StringValue(db.DBInstanceIdentifier), err)
	}
	for _, m := range dbc.DBClusterMembers {
		if aws.StringValue(db.DBInstanceIdentifier) == aws.StringValue(m.DBInstanceIdentifier) {
			if aws.BoolValue(m.IsClusterWriter) {
				d.Set("writer", true)
			} else {
				d.Set("writer", false)
			}
		}
	}

	if db.Endpoint != nil {
		address := aws.StringValue(db.Endpoint.Address)
		port := int(aws.Int64Value(db.Endpoint.Port))
		d.Set("address", address)
		d.Set("endpoint", fmt.Sprintf("%s:%d", address, port))
		d.Set("port", port)
	}

	if db.DBSubnetGroup != nil {
		d.Set("neptune_subnet_group_name", db.DBSubnetGroup.DBSubnetGroupName)
	}

	d.Set("arn", db.DBInstanceArn)
	d.Set("auto_minor_version_upgrade", db.AutoMinorVersionUpgrade)
	d.Set("availability_zone", db.AvailabilityZone)
	d.Set("cluster_identifier", db.DBClusterIdentifier)
	d.Set("dbi_resource_id", db.DbiResourceId)
	d.Set("engine_version", db.EngineVersion)
	d.Set("engine", db.Engine)
	d.Set("identifier", db.DBInstanceIdentifier)
	d.Set("instance_class", db.DBInstanceClass)
	d.Set("kms_key_arn", db.KmsKeyId)
	d.Set("preferred_backup_window", db.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", db.PreferredMaintenanceWindow)
	d.Set("promotion_tier", db.PromotionTier)
	d.Set("publicly_accessible", db.PubliclyAccessible)
	d.Set("storage_encrypted", db.StorageEncrypted)

	if len(db.DBParameterGroups) > 0 {
		d.Set("neptune_parameter_group_name", db.DBParameterGroups[0].DBParameterGroupName)
	}

	if err := saveTagsNeptune(conn, d, aws.StringValue(db.DBInstanceArn)); err != nil {
		return fmt.Errorf("Failed to save tags for Neptune Cluster Instance (%s): %s", aws.StringValue(db.DBInstanceIdentifier), err)
	}

	return nil
}

func resourceAwsNeptuneClusterInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	requestUpdate := false

	req := &neptune.ModifyDBInstanceInput{
		ApplyImmediately:     aws.Bool(d.Get("apply_immediately").(bool)),
		DBInstanceIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("neptune_parameter_group_name") {
		req.DBParameterGroupName = aws.String(d.Get("neptune_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("instance_class") {
		req.DBInstanceClass = aws.String(d.Get("instance_class").(string))
		requestUpdate = true
	}

	if d.HasChange("preferred_backup_window") {
		d.SetPartial("preferred_backup_window")
		req.PreferredBackupWindow = aws.String(d.Get("preferred_backup_window").(string))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		d.SetPartial("preferred_maintenance_window")
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("auto_minor_version_upgrade") {
		d.SetPartial("auto_minor_version_upgrade")
		req.AutoMinorVersionUpgrade = aws.Bool(d.Get("auto_minor_version_upgrade").(bool))
		requestUpdate = true
	}

	if d.HasChange("promotion_tier") {
		d.SetPartial("promotion_tier")
		req.PromotionTier = aws.Int64(int64(d.Get("promotion_tier").(int)))
		requestUpdate = true
	}

	log.Printf("[DEBUG] Send Neptune Instance Modification request: %#v", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] Neptune Instance Modification request: %#v", req)
		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err := conn.ModifyDBInstance(req)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error modifying Neptune Instance %s: %s", d.Id(), err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    resourceAwsNeptuneClusterInstanceCreateUpdatePendingStates,
			Target:     []string{"available"},
			Refresh:    resourceAwsNeptuneInstanceStateRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second,
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

	}

	if err := setTagsNeptune(conn, d, d.Get("arn").(string)); err != nil {
		return err
	}

	return resourceAwsNeptuneClusterInstanceRead(d, meta)
}

func resourceAwsNeptuneClusterInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	log.Printf("[DEBUG] Neptune Cluster Instance destroy: %v", d.Id())

	opts := neptune.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	log.Printf("[DEBUG] Neptune Cluster Instance destroy configuration: %s", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		if isAWSErr(err, neptune.ErrCodeDBInstanceNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting Neptune cluster instance %q: %s", d.Id(), err)
	}

	log.Println("[INFO] Waiting for Neptune Cluster Instance to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsNeptuneClusterInstanceDeletePendingStates,
		Target:     []string{},
		Refresh:    resourceAwsNeptuneInstanceStateRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	_, err := stateConf.WaitForState()
	return err

}

var resourceAwsNeptuneClusterInstanceCreateUpdatePendingStates = []string{
	"backing-up",
	"creating",
	"maintenance",
	"modifying",
	"rebooting",
	"renaming",
	"starting",
	"upgrading",
}

var resourceAwsNeptuneClusterInstanceDeletePendingStates = []string{
	"modifying",
	"deleting",
}

func resourceAwsNeptuneInstanceStateRefreshFunc(id string, conn *neptune.Neptune) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsNeptuneInstanceRetrieve(id, conn)

		if err != nil {
			log.Printf("Error on retrieving Neptune Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.DBInstanceStatus != nil {
			log.Printf("[DEBUG] Neptune Instance status for instance %s: %s", id, aws.StringValue(v.DBInstanceStatus))
		}

		return v, aws.StringValue(v.DBInstanceStatus), nil
	}
}

func resourceAwsNeptuneInstanceRetrieve(id string, conn *neptune.Neptune) (*neptune.DBInstance, error) {
	opts := neptune.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(id),
	}

	log.Printf("[DEBUG] Neptune Instance describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)
	if err != nil {
		if isAWSErr(err, neptune.ErrCodeDBInstanceNotFoundFault, "") {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving Neptune Instances: %s", err)
	}

	if len(resp.DBInstances) != 1 ||
		aws.StringValue(resp.DBInstances[0].DBInstanceIdentifier) != id {
		return nil, nil
	}

	return resp.DBInstances[0], nil
}

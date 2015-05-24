package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/service/rds"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbFromSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbFromSnapshotCreate,
		Read:   resourceAwsDbFromSnapshotRead,
		Update: resourceAwsDbFromSnapshotUpdate,
		Delete: resourceAwsDbFromSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"allocated_storage": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"auto_minor_version_upgrade": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
			},

			"storage_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"db_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"snapshot_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"iops": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"license_model": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"multi_az": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"publicly_accessible": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"db_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"option_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbFromSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	opts := rds.RestoreDBInstanceFromDBSnapshotInput{
		AutoMinorVersionUpgrade: aws.Boolean(d.Get("auto_minor_version_upgrade").(bool)),
		DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
		DBInstanceIdentifier:    aws.String(d.Get("db_identifier").(string)),
		DBSnapshotIdentifier:    aws.String(d.Get("snapshot_identifier").(string)),
		Tags:                    tags,
	}

	if attr, ok := d.GetOk("multi_az"); ok {
		opts.MultiAZ = aws.Boolean(attr.(bool))
	}

	if attr, ok := d.GetOk("license_model"); ok {
		opts.LicenseModel = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("storage_type"); ok {
		opts.StorageType = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		opts.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("iops"); ok {
		opts.IOPS = aws.Long(int64(attr.(int)))
	}

	if attr, ok := d.GetOk("port"); ok {
		opts.Port = aws.Long(int64(attr.(int)))
	}

	if attr, ok := d.GetOk("availability_zone"); ok {
		opts.AvailabilityZone = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("publicly_accessible"); ok {
		opts.PubliclyAccessible = aws.Boolean(attr.(bool))
	}

	log.Printf("[DEBUG] DB Instance create configuration: %#v", opts)
	var err error
	_, err = conn.RestoreDBInstanceFromDBSnapshot(&opts)
	if err != nil {
		return fmt.Errorf("Error creating DB Instance: %s", err)
	}

	d.SetId(d.Get("db_identifier").(string))

	log.Printf("[INFO] DB Instance ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for DB Instance to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     "available",
		Refresh:    resourceAwsDbFromSnapshotStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	resp, err := stateConf.WaitForState()
	if err != nil {
		return err
	}
	fmt.Println(resp)

	return resourceAwsDbFromSnapshotRead(d, meta)
}

func resourceAwsDbFromSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsDbFromSnapshotRetrieve(d, meta)

	if err != nil {
		return err
	}
	if v == nil {
		d.SetId("")
		return nil
	}

	d.Set("username", v.MasterUsername)
	d.Set("engine", v.Engine)
	d.Set("engine_version", v.EngineVersion)
	d.Set("allocated_storage", v.AllocatedStorage)
	d.Set("storage_type", v.StorageType)
	d.Set("availability_zone", v.AvailabilityZone)
	d.Set("license_model", v.LicenseModel)

	// list tags for resource
	// set tags
	conn := meta.(*AWSClient).rdsconn
	arn, err := buildRDSARN(d, meta)
	if err != nil {
		name := "<empty>"
		if v.DBInstanceIdentifier != nil && *v.DBInstanceIdentifier != "" {
			name = *v.DBInstanceIdentifier
		}

		log.Printf("[DEBUG] Error building ARN for DB Instance, not setting Tags for DB %s", name)
	} else {
		resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: aws.String(arn),
		})

		if err != nil {
			log.Printf("[DEBUG] Error retreiving tags for ARN: %s", arn)
		}

		var dt []*rds.Tag
		if len(resp.TagList) > 0 {
			dt = resp.TagList
		}
		d.Set("tags", tagsToMapRDS(dt))
	}

	return nil
}

func resourceAwsDbFromSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	finalSnapshot := d.Get("final_snapshot_identifier").(string)
	if finalSnapshot == "" {
		opts.SkipFinalSnapshot = aws.Boolean(true)
	} else {
		opts.FinalDBSnapshotIdentifier = aws.String(finalSnapshot)
	}

	log.Printf("[DEBUG] DB Instance destroy configuration: %v", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		return err
	}

	log.Println(
		"[INFO] Waiting for DB Instance to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating", "backing-up",
			"modifying", "deleting", "available"},
		Target:     "",
		Refresh:    resourceAwsDbFromSnapshotStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func resourceAwsDbFromSnapshotUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	d.Partial(true)

	req := &rds.ModifyDBInstanceInput{
		ApplyImmediately:     aws.Boolean(d.Get("apply_immediately").(bool)),
		DBInstanceIdentifier: aws.String(d.Id()),
	}
	d.SetPartial("apply_immediately")

	requestUpdate := false
	if d.HasChange("allocated_storage") {
		d.SetPartial("allocated_storage")
		req.AllocatedStorage = aws.Long(int64(d.Get("allocated_storage").(int)))
		requestUpdate = true
	}
	if d.HasChange("backup_retention_period") {
		d.SetPartial("backup_retention_period")
		req.BackupRetentionPeriod = aws.Long(int64(d.Get("backup_retention_period").(int)))
		requestUpdate = true
	}
	if d.HasChange("instance_class") {
		d.SetPartial("instance_class")
		req.DBInstanceClass = aws.String(d.Get("instance_class").(string))
		requestUpdate = true
	}
	if d.HasChange("parameter_group_name") {
		d.SetPartial("parameter_group_name")
		req.DBParameterGroupName = aws.String(d.Get("parameter_group_name").(string))
		requestUpdate = true
	}
	if d.HasChange("engine_version") {
		d.SetPartial("engine_version")
		req.EngineVersion = aws.String(d.Get("engine_version").(string))
		requestUpdate = true
	}
	if d.HasChange("iops") {
		d.SetPartial("iops")
		req.IOPS = aws.Long(int64(d.Get("iops").(int)))
		requestUpdate = true
	}
	if d.HasChange("backup_window") {
		d.SetPartial("backup_window")
		req.PreferredBackupWindow = aws.String(d.Get("backup_window").(string))
		requestUpdate = true
	}
	if d.HasChange("maintenance_window") {
		d.SetPartial("maintenance_window")
		req.PreferredMaintenanceWindow = aws.String(d.Get("maintenance_window").(string))
		requestUpdate = true
	}
	if d.HasChange("password") {
		d.SetPartial("password")
		req.MasterUserPassword = aws.String(d.Get("password").(string))
		requestUpdate = true
	}
	if d.HasChange("multi_az") {
		d.SetPartial("multi_az")
		req.MultiAZ = aws.Boolean(d.Get("multi_az").(bool))
		requestUpdate = true
	}
	if d.HasChange("storage_type") {
		d.SetPartial("storage_type")
		req.StorageType = aws.String(d.Get("storage_type").(string))
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			req.VPCSecurityGroupIDs = s
		}
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			req.DBSecurityGroups = s
		}
		requestUpdate = true
	}

	log.Printf("[DEBUG] Send DB Instance Modification request: %#v", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] DB Instance Modification request: %#v", req)
		_, err := conn.ModifyDBInstance(req)
		if err != nil {
			return fmt.Errorf("Error modifying DB Instance %s: %s", d.Id(), err)
		}
	}

	// seperate request to promote a database
	if d.HasChange("replicate_source_db") {
		if d.Get("replicate_source_db").(string) == "" {
			// promote
			opts := rds.PromoteReadReplicaInput{
				DBInstanceIdentifier: aws.String(d.Id()),
			}
			attr := d.Get("backup_retention_period")
			opts.BackupRetentionPeriod = aws.Long(int64(attr.(int)))
			if attr, ok := d.GetOk("backup_window"); ok {
				opts.PreferredBackupWindow = aws.String(attr.(string))
			}
			_, err := conn.PromoteReadReplica(&opts)
			if err != nil {
				return fmt.Errorf("Error promoting database: %#v", err)
			}
			d.Set("replicate_source_db", "")
		} else {
			return fmt.Errorf("cannot elect new source database for replication")
		}
	}

	if arn, err := buildRDSARN(d, meta); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}
	d.Partial(false)
	return resourceAwsDbFromSnapshotRead(d, meta)
}

func resourceAwsDbFromSnapshotRetrieve(
	d *schema.ResourceData, meta interface{}) (*rds.DBInstance, error) {
	conn := meta.(*AWSClient).rdsconn

	opts := rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] DB Snapshot describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)

	if err != nil {
		dbsnapshoterr, ok := err.(awserr.Error)
		if ok && dbsnapshoterr.Code() == "DBSnapshotNotFound" {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DB Snapshot: %s", err)
	}

	if len(resp.DBInstances) != 1 ||
		*resp.DBInstances[0].DBInstanceIdentifier != d.Id() {
		if err != nil {
			return nil, nil
		}
	}

	return resp.DBInstances[0], nil
}

func resourceAwsDbFromSnapshotStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsDbFromSnapshotRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving DB Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.DBInstanceStatus != nil {
			log.Printf("[DEBUG] DB Snapshot status for instance %s: %s", d.Id(), *v.DBInstanceStatus)
		}

		return v, *v.DBInstanceStatus, nil
	}
}

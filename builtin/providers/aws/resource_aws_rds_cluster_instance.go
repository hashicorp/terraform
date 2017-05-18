package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSClusterInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterInstanceCreate,
		Read:   resourceAwsRDSClusterInstanceRead,
		Update: resourceAwsRDSClusterInstanceUpdate,
		Delete: resourceAwsRDSClusterInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(90 * time.Minute),
			Update: schema.DefaultTimeout(90 * time.Minute),
			Delete: schema.DefaultTimeout(90 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"identifier": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"identifier_prefix"},
				ValidateFunc:  validateRdsIdentifier,
			},
			"identifier_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateRdsIdentifierPrefix,
			},

			"db_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"writer": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"instance_class": {
				Type:     schema.TypeString,
				Required: true,
			},

			"db_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			// apply_immediately is used to determine when the update modifications
			// take place.
			// See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"monitoring_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					if v != nil {
						value := v.(string)
						return strings.ToLower(value)
					}
					return ""
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},

			"preferred_backup_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},

			"monitoring_interval": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"promotion_tier": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRDSClusterInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	createOpts := &rds.CreateDBInstanceInput{
		DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
		DBClusterIdentifier:     aws.String(d.Get("cluster_identifier").(string)),
		Engine:                  aws.String("aurora"),
		PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
		PromotionTier:           aws.Int64(int64(d.Get("promotion_tier").(int))),
		AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		Tags: tags,
	}

	if attr, ok := d.GetOk("db_parameter_group_name"); ok {
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

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		createOpts.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("monitoring_role_arn"); ok {
		createOpts.MonitoringRoleArn = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_backup_window"); ok {
		createOpts.PreferredBackupWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_maintenance_window"); ok {
		createOpts.PreferredMaintenanceWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("monitoring_interval"); ok {
		createOpts.MonitoringInterval = aws.Int64(int64(attr.(int)))
	}

	log.Printf("[DEBUG] Creating RDS DB Instance opts: %s", createOpts)
	resp, err := conn.CreateDBInstance(createOpts)
	if err != nil {
		return err
	}

	d.SetId(*resp.DBInstance.DBInstanceIdentifier)

	// reuse db_instance refresh func
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     []string{"available"},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceRead(d *schema.ResourceData, meta interface{}) error {
	db, err := resourceAwsDbInstanceRetrieve(d, meta)
	// Errors from this helper are always reportable
	if err != nil {
		return fmt.Errorf("[WARN] Error on retrieving RDS Cluster Instance (%s): %s", d.Id(), err)
	}
	// A nil response means "not found"
	if db == nil {
		log.Printf("[WARN] RDS Cluster Instance (%s): not found, removing from state.", d.Id())
		d.SetId("")
		return nil
	}

	// Retrieve DB Cluster information, to determine if this Instance is a writer
	conn := meta.(*AWSClient).rdsconn
	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: db.DBClusterIdentifier,
	})

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if *c.DBClusterIdentifier == *db.DBClusterIdentifier {
			dbc = c
		}
	}

	if dbc == nil {
		return fmt.Errorf("[WARN] Error finding RDS Cluster (%s) for Cluster Instance (%s): %s",
			*db.DBClusterIdentifier, *db.DBInstanceIdentifier, err)
	}

	for _, m := range dbc.DBClusterMembers {
		if *db.DBInstanceIdentifier == *m.DBInstanceIdentifier {
			if *m.IsClusterWriter == true {
				d.Set("writer", true)
			} else {
				d.Set("writer", false)
			}
		}
	}

	if db.Endpoint != nil {
		d.Set("endpoint", db.Endpoint.Address)
		d.Set("port", db.Endpoint.Port)
	}

	d.Set("publicly_accessible", db.PubliclyAccessible)
	d.Set("cluster_identifier", db.DBClusterIdentifier)
	d.Set("instance_class", db.DBInstanceClass)
	d.Set("identifier", db.DBInstanceIdentifier)
	d.Set("storage_encrypted", db.StorageEncrypted)
	d.Set("kms_key_id", db.KmsKeyId)
	d.Set("auto_minor_version_upgrade", db.AutoMinorVersionUpgrade)
	d.Set("promotion_tier", db.PromotionTier)
	d.Set("preferred_backup_window", db.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", db.PreferredMaintenanceWindow)

	if db.MonitoringInterval != nil {
		d.Set("monitoring_interval", db.MonitoringInterval)
	}

	if db.MonitoringRoleArn != nil {
		d.Set("monitoring_role_arn", db.MonitoringRoleArn)
	}

	if len(db.DBParameterGroups) > 0 {
		d.Set("db_parameter_group_name", db.DBParameterGroups[0].DBParameterGroupName)
	}

	// Fetch and save tags
	arn, err := buildRDSARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		log.Printf("[DEBUG] Error building ARN for RDS Cluster Instance (%s), not setting Tags", *db.DBInstanceIdentifier)
	} else {
		if err := saveTagsRDS(conn, d, arn); err != nil {
			log.Printf("[WARN] Failed to save tags for RDS Cluster Instance (%s): %s", *db.DBClusterIdentifier, err)
		}
	}

	return nil
}

func resourceAwsRDSClusterInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	requestUpdate := false

	req := &rds.ModifyDBInstanceInput{
		ApplyImmediately:     aws.Bool(d.Get("apply_immediately").(bool)),
		DBInstanceIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("db_parameter_group_name") {
		req.DBParameterGroupName = aws.String(d.Get("db_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("instance_class") {
		req.DBInstanceClass = aws.String(d.Get("instance_class").(string))
		requestUpdate = true
	}

	if d.HasChange("monitoring_role_arn") {
		d.SetPartial("monitoring_role_arn")
		req.MonitoringRoleArn = aws.String(d.Get("monitoring_role_arn").(string))
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

	if d.HasChange("monitoring_interval") {
		d.SetPartial("monitoring_interval")
		req.MonitoringInterval = aws.Int64(int64(d.Get("monitoring_interval").(int)))
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

	log.Printf("[DEBUG] Send DB Instance Modification request: %#v", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] DB Instance Modification request: %#v", req)
		_, err := conn.ModifyDBInstance(req)
		if err != nil {
			return fmt.Errorf("Error modifying DB Instance %s: %s", d.Id(), err)
		}

		// reuse db_instance refresh func
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"creating", "backing-up", "modifying"},
			Target:     []string{"available"},
			Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

	}

	if arn, err := buildRDSARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		}
	}

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] RDS Cluster Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	log.Printf("[DEBUG] RDS Cluster Instance destroy configuration: %s", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		return err
	}

	// re-uses db_instance refresh func
	log.Println("[INFO] Waiting for RDS Cluster Instance to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"modifying", "deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil

}

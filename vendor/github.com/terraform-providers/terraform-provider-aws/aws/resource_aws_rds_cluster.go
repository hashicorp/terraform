package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterCreate,
		Read:   resourceAwsRDSClusterRead,
		Update: resourceAwsRDSClusterUpdate,
		Delete: resourceAwsRDSClusterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsRdsClusterImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(120 * time.Minute),
			Update: schema.DefaultTimeout(120 * time.Minute),
			Delete: schema.DefaultTimeout(120 * time.Minute),
		},

		Schema: map[string]*schema.Schema{

			"availability_zones": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"cluster_identifier": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cluster_identifier_prefix"},
				ValidateFunc:  validateRdsIdentifier,
			},
			"cluster_identifier_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateRdsIdentifierPrefix,
			},

			"cluster_members": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"db_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"db_cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"reader_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"hosted_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "aurora",
				ForceNew:     true,
				ValidateFunc: validateRdsEngine,
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"final_snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only alphanumeric characters and hyphens allowed in %q", k))
					}
					if regexp.MustCompile(`--`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot contain two consecutive hyphens", k))
					}
					if regexp.MustCompile(`-$`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot end in a hyphen", k))
					}
					return
				},
			},

			"skip_final_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"master_username": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},

			"master_password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"snapshot_identifier": {
				Type:     schema.TypeString,
				Computed: false,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"port": {
				Type:     schema.TypeInt,
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

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
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

			"backup_retention_period": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value > 35 {
						es = append(es, fmt.Errorf(
							"backup retention period cannot be more than 35 days"))
					}
					return
				},
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"replication_source_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"iam_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"iam_database_authentication_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"cluster_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"source_region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRdsClusterImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither skip_final_snapshot nor final_snapshot_identifier can be fetched
	// from any API call, so we need to default skip_final_snapshot to true so
	// that final_snapshot_identifier is not required
	d.Set("skip_final_snapshot", true)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsRDSClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	var identifier string
	if v, ok := d.GetOk("cluster_identifier"); ok {
		identifier = v.(string)
	} else {
		if v, ok := d.GetOk("cluster_identifier_prefix"); ok {
			identifier = resource.PrefixedUniqueId(v.(string))
		} else {
			identifier = resource.PrefixedUniqueId("tf-")
		}

		d.Set("cluster_identifier", identifier)
	}

	if _, ok := d.GetOk("snapshot_identifier"); ok {
		opts := rds.RestoreDBClusterFromSnapshotInput{
			DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
			SnapshotIdentifier:  aws.String(d.Get("snapshot_identifier").(string)),
			Engine:              aws.String(d.Get("engine").(string)),
			Tags:                tags,
		}

		if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
			opts.AvailabilityZones = expandStringList(attr.List())
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("database_name"); ok {
			opts.DatabaseName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		// Check if any of the parameters that require a cluster modification after creation are set
		var clusterUpdate bool
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			clusterUpdate = true
			opts.VpcSecurityGroupIds = expandStringList(attr.List())
		}

		if _, ok := d.GetOk("db_cluster_parameter_group_name"); ok {
			clusterUpdate = true
		}

		if _, ok := d.GetOk("backup_retention_period"); ok {
			clusterUpdate = true
		}

		log.Printf("[DEBUG] RDS Cluster restore from snapshot configuration: %s", opts)
		_, err := conn.RestoreDBClusterFromSnapshot(&opts)
		if err != nil {
			return fmt.Errorf("Error creating RDS Cluster: %s", err)
		}

		if clusterUpdate {
			log.Printf("[INFO] RDS Cluster is restoring from snapshot with default db_cluster_parameter_group_name, backup_retention_period and vpc_security_group_ids" +
				"but custom values should be set, will now update after snapshot is restored!")

			d.SetId(d.Get("cluster_identifier").(string))

			log.Printf("[INFO] RDS Cluster ID: %s", d.Id())

			log.Println("[INFO] Waiting for RDS Cluster to be available")

			stateConf := &resource.StateChangeConf{
				Pending:    resourceAwsRdsClusterCreatePendingStates,
				Target:     []string{"available"},
				Refresh:    resourceAwsRDSClusterStateRefreshFunc(d, meta),
				Timeout:    d.Timeout(schema.TimeoutCreate),
				MinTimeout: 10 * time.Second,
				Delay:      30 * time.Second,
			}

			// Wait, catching any errors
			_, err := stateConf.WaitForState()
			if err != nil {
				return err
			}

			err = resourceAwsRDSClusterUpdate(d, meta)
			if err != nil {
				return err
			}
		}
	} else if _, ok := d.GetOk("replication_source_identifier"); ok {
		createOpts := &rds.CreateDBClusterInput{
			DBClusterIdentifier:         aws.String(d.Get("cluster_identifier").(string)),
			Engine:                      aws.String(d.Get("engine").(string)),
			StorageEncrypted:            aws.Bool(d.Get("storage_encrypted").(bool)),
			ReplicationSourceIdentifier: aws.String(d.Get("replication_source_identifier").(string)),
			Tags: tags,
		}

		if attr, ok := d.GetOk("port"); ok {
			createOpts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			createOpts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_cluster_parameter_group_name"); ok {
			createOpts.DBClusterParameterGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("engine_version"); ok {
			createOpts.EngineVersion = aws.String(attr.(string))
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			createOpts.VpcSecurityGroupIds = expandStringList(attr.List())
		}

		if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
			createOpts.AvailabilityZones = expandStringList(attr.List())
		}

		if v, ok := d.GetOk("backup_retention_period"); ok {
			createOpts.BackupRetentionPeriod = aws.Int64(int64(v.(int)))
		}

		if v, ok := d.GetOk("preferred_backup_window"); ok {
			createOpts.PreferredBackupWindow = aws.String(v.(string))
		}

		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			createOpts.PreferredMaintenanceWindow = aws.String(v.(string))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			createOpts.KmsKeyId = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("source_region"); ok {
			createOpts.SourceRegion = aws.String(attr.(string))
		}

		log.Printf("[DEBUG] Create RDS Cluster as read replica: %s", createOpts)
		resp, err := conn.CreateDBCluster(createOpts)
		if err != nil {
			log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
			return err
		}

		log.Printf("[DEBUG]: RDS Cluster create response: %s", resp)

	} else {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_rds_cluster: %s: "master_password": required field is not set`, d.Get("database_name").(string))
		}

		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_rds_cluster: %s: "master_username": required field is not set`, d.Get("database_name").(string))
		}

		createOpts := &rds.CreateDBClusterInput{
			DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
			Engine:              aws.String(d.Get("engine").(string)),
			MasterUserPassword:  aws.String(d.Get("master_password").(string)),
			MasterUsername:      aws.String(d.Get("master_username").(string)),
			StorageEncrypted:    aws.Bool(d.Get("storage_encrypted").(bool)),
			Tags:                tags,
		}

		if v := d.Get("database_name"); v.(string) != "" {
			createOpts.DatabaseName = aws.String(v.(string))
		}

		if attr, ok := d.GetOk("port"); ok {
			createOpts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			createOpts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_cluster_parameter_group_name"); ok {
			createOpts.DBClusterParameterGroupName = aws.String(attr.(string))
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			createOpts.VpcSecurityGroupIds = expandStringList(attr.List())
		}

		if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
			createOpts.AvailabilityZones = expandStringList(attr.List())
		}

		if v, ok := d.GetOk("backup_retention_period"); ok {
			createOpts.BackupRetentionPeriod = aws.Int64(int64(v.(int)))
		}

		if v, ok := d.GetOk("preferred_backup_window"); ok {
			createOpts.PreferredBackupWindow = aws.String(v.(string))
		}

		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			createOpts.PreferredMaintenanceWindow = aws.String(v.(string))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			createOpts.KmsKeyId = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("iam_database_authentication_enabled"); ok {
			createOpts.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] RDS Cluster create options: %s", createOpts)
		resp, err := conn.CreateDBCluster(createOpts)
		if err != nil {
			log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
			return err
		}

		log.Printf("[DEBUG]: RDS Cluster create response: %s", resp)
	}

	d.SetId(d.Get("cluster_identifier").(string))

	log.Printf("[INFO] RDS Cluster ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for RDS Cluster to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsRdsClusterCreatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for RDS Cluster state to be \"available\": %s", err)
	}

	if v, ok := d.GetOk("iam_roles"); ok {
		for _, role := range v.(*schema.Set).List() {
			err := setIamRoleToRdsCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}
	}

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if "DBClusterNotFoundFault" == awsErr.Code() {
				d.SetId("")
				log.Printf("[DEBUG] RDS Cluster (%s) not found", d.Id())
				return nil
			}
		}
		log.Printf("[DEBUG] Error describing RDS Cluster (%s)", d.Id())
		return err
	}

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if *c.DBClusterIdentifier == d.Id() {
			dbc = c
		}
	}

	if dbc == nil {
		log.Printf("[WARN] RDS Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	return flattenAwsRdsClusterResource(d, meta, dbc)
}

func flattenAwsRdsClusterResource(d *schema.ResourceData, meta interface{}, dbc *rds.DBCluster) error {
	conn := meta.(*AWSClient).rdsconn

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("[DEBUG] Error saving AvailabilityZones to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	// Only set the DatabaseName if it is not nil. There is a known API bug where
	// RDS accepts a DatabaseName but does not return it, causing a perpetual
	// diff.
	//	See https://github.com/hashicorp/terraform/issues/4671 for backstory
	if dbc.DatabaseName != nil {
		d.Set("database_name", dbc.DatabaseName)
	}

	d.Set("cluster_identifier", dbc.DBClusterIdentifier)
	d.Set("cluster_resource_id", dbc.DbClusterResourceId)
	d.Set("db_subnet_group_name", dbc.DBSubnetGroup)
	d.Set("db_cluster_parameter_group_name", dbc.DBClusterParameterGroup)
	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine", dbc.Engine)
	d.Set("engine_version", dbc.EngineVersion)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	d.Set("storage_encrypted", dbc.StorageEncrypted)
	d.Set("backup_retention_period", dbc.BackupRetentionPeriod)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)
	d.Set("kms_key_id", dbc.KmsKeyId)
	d.Set("reader_endpoint", dbc.ReaderEndpoint)
	d.Set("replication_source_identifier", dbc.ReplicationSourceIdentifier)
	d.Set("iam_database_authentication_enabled", dbc.IAMDatabaseAuthenticationEnabled)
	d.Set("hosted_zone_id", dbc.HostedZoneId)

	var vpcg []string
	for _, g := range dbc.VpcSecurityGroups {
		vpcg = append(vpcg, *g.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("[DEBUG] Error saving VPC Security Group IDs to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	var cm []string
	for _, m := range dbc.DBClusterMembers {
		cm = append(cm, *m.DBInstanceIdentifier)
	}
	if err := d.Set("cluster_members", cm); err != nil {
		return fmt.Errorf("[DEBUG] Error saving RDS Cluster Members to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	var roles []string
	for _, r := range dbc.AssociatedRoles {
		roles = append(roles, *r.RoleArn)
	}

	if err := d.Set("iam_roles", roles); err != nil {
		return fmt.Errorf("[DEBUG] Error saving IAM Roles to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	// Fetch and save tags
	arn, err := buildRDSClusterARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		log.Printf("[DEBUG] Error building ARN for RDS Cluster (%s), not setting Tags", *dbc.DBClusterIdentifier)
	} else {
		if err := saveTagsRDS(conn, d, arn); err != nil {
			log.Printf("[WARN] Failed to save tags for RDS Cluster (%s): %s", *dbc.DBClusterIdentifier, err)
		}
	}

	return nil
}

func resourceAwsRDSClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	requestUpdate := false

	req := &rds.ModifyDBClusterInput{
		ApplyImmediately:    aws.Bool(d.Get("apply_immediately").(bool)),
		DBClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.VpcSecurityGroupIds = expandStringList(attr.List())
		} else {
			req.VpcSecurityGroupIds = []*string{}
		}
		requestUpdate = true
	}

	if d.HasChange("preferred_backup_window") {
		req.PreferredBackupWindow = aws.String(d.Get("preferred_backup_window").(string))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("backup_retention_period") {
		req.BackupRetentionPeriod = aws.Int64(int64(d.Get("backup_retention_period").(int)))
		requestUpdate = true
	}

	if d.HasChange("db_cluster_parameter_group_name") {
		d.SetPartial("db_cluster_parameter_group_name")
		req.DBClusterParameterGroupName = aws.String(d.Get("db_cluster_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("iam_database_authentication_enabled") {
		req.EnableIAMDatabaseAuthentication = aws.Bool(d.Get("iam_database_authentication_enabled").(bool))
		requestUpdate = true
	}

	if requestUpdate {
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err := conn.ModifyDBCluster(req)
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if ok && awsErr.Code() == rds.ErrCodeInvalidDBClusterStateFault {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Failed to modify RDS Cluster (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("iam_roles") {
		oraw, nraw := d.GetChange("iam_roles")
		if oraw == nil {
			oraw = new(schema.Set)
		}
		if nraw == nil {
			nraw = new(schema.Set)
		}

		os := oraw.(*schema.Set)
		ns := nraw.(*schema.Set)
		removeRoles := os.Difference(ns)
		enableRoles := ns.Difference(os)

		for _, role := range enableRoles.List() {
			err := setIamRoleToRdsCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}

		for _, role := range removeRoles.List() {
			err := removeIamRoleFromRdsCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}
	}

	if arn, err := buildRDSClusterARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	log.Printf("[DEBUG] Destroying RDS Cluster (%s)", d.Id())

	deleteOpts := rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalSnapshot = aws.Bool(skipFinalSnapshot)

	if skipFinalSnapshot == false {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalDBSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("RDS Cluster FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] RDS Cluster delete options: %s", deleteOpts)

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteDBCluster(&deleteOpts)
		if err != nil {
			if isAWSErr(err, rds.ErrCodeInvalidDBClusterStateFault, "is not currently in the available state") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, rds.ErrCodeDBClusterNotFoundFault, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("RDS Cluster cannot be deleted: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsRdsClusterDeletePendingStates,
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error deleting RDS Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSClusterStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).rdsconn

		resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(d.Id()),
		})

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if "DBClusterNotFoundFault" == awsErr.Code() {
					return 42, "destroyed", nil
				}
			}
			log.Printf("[WARN] Error on retrieving DB Cluster (%s) when waiting: %s", d.Id(), err)
			return nil, "", err
		}

		var dbc *rds.DBCluster

		for _, c := range resp.DBClusters {
			if *c.DBClusterIdentifier == d.Id() {
				dbc = c
			}
		}

		if dbc == nil {
			return 42, "destroyed", nil
		}

		if dbc.Status != nil {
			log.Printf("[DEBUG] DB Cluster status (%s): %s", d.Id(), *dbc.Status)
		}

		return dbc, *dbc.Status, nil
	}
}

func buildRDSClusterARN(identifier, partition, accountid, region string) (string, error) {
	if partition == "" {
		return "", fmt.Errorf("Unable to construct RDS Cluster ARN because of missing AWS partition")
	}
	if accountid == "" {
		return "", fmt.Errorf("Unable to construct RDS Cluster ARN because of missing AWS Account ID")
	}

	arn := fmt.Sprintf("arn:%s:rds:%s:%s:cluster:%s", partition, region, accountid, identifier)
	return arn, nil

}

func setIamRoleToRdsCluster(clusterIdentifier string, roleArn string, conn *rds.RDS) error {
	params := &rds.AddRoleToDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.AddRoleToDBCluster(params)
	if err != nil {
		return err
	}

	return nil
}

func removeIamRoleFromRdsCluster(clusterIdentifier string, roleArn string, conn *rds.RDS) error {
	params := &rds.RemoveRoleFromDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.RemoveRoleFromDBCluster(params)
	if err != nil {
		return err
	}

	return nil
}

var resourceAwsRdsClusterCreatePendingStates = []string{
	"creating",
	"backing-up",
	"modifying",
	"preparing-data-migration",
	"migrating",
	"resetting-master-credentials",
}

var resourceAwsRdsClusterDeletePendingStates = []string{
	"available",
	"deleting",
	"backing-up",
	"modifying",
}

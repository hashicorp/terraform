package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/docdb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDocDBCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDocDBClusterCreate,
		Read:   resourceAwsDocDBClusterRead,
		Update: resourceAwsDocDBClusterUpdate,
		Delete: resourceAwsDocDBClusterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsDocDBClusterImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(120 * time.Minute),
			Update: schema.DefaultTimeout(120 * time.Minute),
			Delete: schema.DefaultTimeout(120 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

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
				ValidateFunc:  validateDocDBIdentifier,
			},
			"cluster_identifier_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cluster_identifier"},
				ValidateFunc:  validateDocDBIdentifierPrefix,
			},

			"cluster_members": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
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
				Default:      "docdb",
				ForceNew:     true,
				ValidateFunc: validateDocDBEngine(),
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
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
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"port": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      27017,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1150, 65535),
			},

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
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validation.IntAtMost(35),
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"cluster_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enabled_cloudwatch_logs_exports": {
				Type:     schema.TypeList,
				Computed: false,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"audit",
					}, false),
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDocDBClusterImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither skip_final_snapshot nor final_snapshot_identifier can be fetched
	// from any API call, so we need to default skip_final_snapshot to true so
	// that final_snapshot_identifier is not required
	d.Set("skip_final_snapshot", true)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsDocDBClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn
	tags := tagsFromMapDocDB(d.Get("tags").(map[string]interface{}))

	// Some API calls (e.g. RestoreDBClusterFromSnapshot do not support all
	// parameters to correctly apply all settings in one pass. For missing
	// parameters or unsupported configurations, we may need to call
	// ModifyDBInstance afterwadocdb to prevent Terraform operators from API
	// errors or needing to double apply.
	var requiresModifyDbCluster bool
	modifyDbClusterInput := &docdb.ModifyDBClusterInput{
		ApplyImmediately: aws.Bool(true),
	}

	var identifier string
	if v, ok := d.GetOk("cluster_identifier"); ok {
		identifier = v.(string)
	} else if v, ok := d.GetOk("cluster_identifier_prefix"); ok {
		identifier = resource.PrefixedUniqueId(v.(string))
	} else {
		identifier = resource.PrefixedUniqueId("tf-")
	}

	if _, ok := d.GetOk("snapshot_identifier"); ok {
		opts := docdb.RestoreDBClusterFromSnapshotInput{
			DBClusterIdentifier: aws.String(identifier),
			Engine:              aws.String(d.Get("engine").(string)),
			SnapshotIdentifier:  aws.String(d.Get("snapshot_identifier").(string)),
			Tags:                tags,
		}

		if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
			opts.AvailabilityZones = expandStringList(attr.List())
		}

		if attr, ok := d.GetOk("backup_retention_period"); ok {
			modifyDbClusterInput.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
			requiresModifyDbCluster = true
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_cluster_parameter_group_name"); ok {
			modifyDbClusterInput.DBClusterParameterGroupName = aws.String(attr.(string))
			requiresModifyDbCluster = true
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			opts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOk("engine_version"); ok {
			opts.EngineVersion = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			opts.KmsKeyId = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("preferred_backup_window"); ok {
			modifyDbClusterInput.PreferredBackupWindow = aws.String(attr.(string))
			requiresModifyDbCluster = true
		}

		if attr, ok := d.GetOk("preferred_maintenance_window"); ok {
			modifyDbClusterInput.PreferredMaintenanceWindow = aws.String(attr.(string))
			requiresModifyDbCluster = true
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			opts.VpcSecurityGroupIds = expandStringList(attr.List())
		}

		log.Printf("[DEBUG] DocDB Cluster restore from snapshot configuration: %s", opts)
		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err := conn.RestoreDBClusterFromSnapshot(&opts)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error creating DocDB Cluster: %s", err)
		}
	} else {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_docdb_cluster: %s: "master_password": required field is not set`, identifier)
		}

		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_docdb_cluster: %s: "master_username": required field is not set`, identifier)
		}

		createOpts := &docdb.CreateDBClusterInput{
			DBClusterIdentifier: aws.String(identifier),
			Engine:              aws.String(d.Get("engine").(string)),
			MasterUserPassword:  aws.String(d.Get("master_password").(string)),
			MasterUsername:      aws.String(d.Get("master_username").(string)),
			Tags:                tags,
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

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			createOpts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOkExists("storage_encrypted"); ok {
			createOpts.StorageEncrypted = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] DocDB Cluster create options: %s", createOpts)
		var resp *docdb.CreateDBClusterOutput
		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			var err error
			resp, err = conn.CreateDBCluster(createOpts)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error creating DocDB cluster: %s", err)
		}

		log.Printf("[DEBUG]: DocDB Cluster create response: %s", resp)
	}

	d.SetId(identifier)

	log.Printf("[INFO] DocDB Cluster ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for DocDB Cluster to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsDocDBClusterCreatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsDocDBClusterStateRefreshFunc(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for DocDB Cluster state to be \"available\": %s", err)
	}

	if requiresModifyDbCluster {
		modifyDbClusterInput.DBClusterIdentifier = aws.String(d.Id())

		log.Printf("[INFO] DocDB Cluster (%s) configuration requires ModifyDBCluster: %s", d.Id(), modifyDbClusterInput)
		_, err := conn.ModifyDBCluster(modifyDbClusterInput)
		if err != nil {
			return fmt.Errorf("error modifying DocDB Cluster (%s): %s", d.Id(), err)
		}

		log.Printf("[INFO] Waiting for DocDB Cluster (%s) to be available", d.Id())
		err = waitForDocDBClusterUpdate(conn, d.Id(), d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return fmt.Errorf("error waiting for DocDB Cluster (%s) to be available: %s", d.Id(), err)
		}
	}

	return resourceAwsDocDBClusterRead(d, meta)
}

func resourceAwsDocDBClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn

	input := &docdb.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Describing DocDB Cluster: %s", input)
	resp, err := conn.DescribeDBClusters(input)

	if isAWSErr(err, docdb.ErrCodeDBClusterNotFoundFault, "") {
		log.Printf("[WARN] DocDB Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error describing DocDB Cluster (%s): %s", d.Id(), err)
	}

	if resp == nil {
		return fmt.Errorf("Error retrieving DocDB cluster: empty response for: %s", input)
	}

	var dbc *docdb.DBCluster
	for _, c := range resp.DBClusters {
		if aws.StringValue(c.DBClusterIdentifier) == d.Id() {
			dbc = c
			break
		}
	}

	if dbc == nil {
		log.Printf("[WARN] DocDB Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}

	d.Set("arn", dbc.DBClusterArn)
	d.Set("backup_retention_period", dbc.BackupRetentionPeriod)
	d.Set("cluster_identifier", dbc.DBClusterIdentifier)

	var cm []string
	for _, m := range dbc.DBClusterMembers {
		cm = append(cm, aws.StringValue(m.DBInstanceIdentifier))
	}
	if err := d.Set("cluster_members", cm); err != nil {
		return fmt.Errorf("error setting cluster_members: %s", err)
	}

	d.Set("cluster_resource_id", dbc.DbClusterResourceId)

	d.Set("db_cluster_parameter_group_name", dbc.DBClusterParameterGroup)
	d.Set("db_subnet_group_name", dbc.DBSubnetGroup)

	if err := d.Set("enabled_cloudwatch_logs_exports", aws.StringValueSlice(dbc.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("error setting enabled_cloudwatch_logs_exports: %s", err)
	}

	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine_version", dbc.EngineVersion)
	d.Set("engine", dbc.Engine)
	d.Set("hosted_zone_id", dbc.HostedZoneId)

	d.Set("kms_key_id", dbc.KmsKeyId)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)
	d.Set("reader_endpoint", dbc.ReaderEndpoint)
	d.Set("storage_encrypted", dbc.StorageEncrypted)

	var vpcg []string
	for _, g := range dbc.VpcSecurityGroups {
		vpcg = append(vpcg, aws.StringValue(g.VpcSecurityGroupId))
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("error setting vpc_security_group_ids: %s", err)
	}

	// Fetch and save tags
	if err := saveTagsDocDB(conn, d, aws.StringValue(dbc.DBClusterArn)); err != nil {
		log.Printf("[WARN] Failed to save tags for DocDB Cluster (%s): %s", aws.StringValue(dbc.DBClusterIdentifier), err)
	}

	return nil
}

func resourceAwsDocDBClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn
	requestUpdate := false

	req := &docdb.ModifyDBClusterInput{
		ApplyImmediately:    aws.Bool(d.Get("apply_immediately").(bool)),
		DBClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
		requestUpdate = true
	}

	if d.HasChange("engine_version") {
		req.EngineVersion = aws.String(d.Get("engine_version").(string))
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

	if d.HasChange("enabled_cloudwatch_logs_exports") {
		d.SetPartial("enabled_cloudwatch_logs_exports")
		req.CloudwatchLogsExportConfiguration = buildDocDBCloudwatchLogsExportConfiguration(d)
		requestUpdate = true
	}

	if requestUpdate {
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err := conn.ModifyDBCluster(req)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}

				if isAWSErr(err, docdb.ErrCodeInvalidDBClusterStateFault, "is not currently in the available state") {
					return resource.RetryableError(err)
				}

				if isAWSErr(err, docdb.ErrCodeInvalidDBClusterStateFault, "DB cluster is not available for modification") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Failed to modify DocDB Cluster (%s): %s", d.Id(), err)
		}

		log.Printf("[INFO] Waiting for DocDB Cluster (%s) to be available", d.Id())
		err = waitForDocDBClusterUpdate(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return fmt.Errorf("error waiting for DocDB Cluster (%s) to be available: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags") {
		if err := setTagsDocDB(conn, d); err != nil {
			return err
		}

		d.SetPartial("tags")
	}

	return resourceAwsDocDBClusterRead(d, meta)
}

func resourceAwsDocDBClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn
	log.Printf("[DEBUG] Destroying DocDB Cluster (%s)", d.Id())

	deleteOpts := docdb.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalSnapshot = aws.Bool(skipFinalSnapshot)

	if !skipFinalSnapshot {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalDBSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("DocDB Cluster FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] DocDB Cluster delete options: %s", deleteOpts)

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteDBCluster(&deleteOpts)
		if err != nil {
			if isAWSErr(err, docdb.ErrCodeInvalidDBClusterStateFault, "is not currently in the available state") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, docdb.ErrCodeDBClusterNotFoundFault, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("DocDB Cluster cannot be deleted: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsDocDBClusterDeletePendingStates,
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsDocDBClusterStateRefreshFunc(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting DocDB Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsDocDBClusterStateRefreshFunc(conn *docdb.DocDB, dbClusterIdentifier string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDBClusters(&docdb.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(dbClusterIdentifier),
		})

		if isAWSErr(err, docdb.ErrCodeDBClusterNotFoundFault, "") {
			return 42, "destroyed", nil
		}

		if err != nil {
			return nil, "", err
		}

		var dbc *docdb.DBCluster

		for _, c := range resp.DBClusters {
			if *c.DBClusterIdentifier == dbClusterIdentifier {
				dbc = c
			}
		}

		if dbc == nil {
			return 42, "destroyed", nil
		}

		if dbc.Status != nil {
			log.Printf("[DEBUG] DB Cluster status (%s): %s", dbClusterIdentifier, *dbc.Status)
		}

		return dbc, aws.StringValue(dbc.Status), nil
	}
}

var resourceAwsDocDBClusterCreatePendingStates = []string{
	"creating",
	"backing-up",
	"modifying",
	"preparing-data-migration",
	"migrating",
	"resetting-master-credentials",
}

var resourceAwsDocDBClusterDeletePendingStates = []string{
	"available",
	"deleting",
	"backing-up",
	"modifying",
}

var resourceAwsDocDBClusterUpdatePendingStates = []string{
	"backing-up",
	"modifying",
	"resetting-master-credentials",
	"upgrading",
}

func waitForDocDBClusterUpdate(conn *docdb.DocDB, id string, timeout time.Duration) error {
	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsDocDBClusterUpdatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsDocDBClusterStateRefreshFunc(conn, id),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	return err
}

func buildDocDBCloudwatchLogsExportConfiguration(d *schema.ResourceData) *docdb.CloudwatchLogsExportConfiguration {

	oraw, nraw := d.GetChange("enabled_cloudwatch_logs_exports")
	o := oraw.([]interface{})
	n := nraw.([]interface{})

	create, disable := diffCloudwatchLogsExportConfiguration(o, n)

	return &docdb.CloudwatchLogsExportConfiguration{
		EnableLogTypes:  expandStringList(create),
		DisableLogTypes: expandStringList(disable),
	}
}

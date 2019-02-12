package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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

			"backtrack_window": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 259200),
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
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cluster_identifier"},
				ValidateFunc:  validateRdsIdentifierPrefix,
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

			"deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"global_cluster_identifier": {
				Type:     schema.TypeString,
				Optional: true,
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
				ValidateFunc: validateRdsEngine(),
			},

			"engine_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "provisioned",
				ValidateFunc: validation.StringInSlice([]string{
					"global",
					"parallelquery",
					"provisioned",
					"serverless",
				}, false),
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"scaling_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto_pause": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"max_capacity": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  16,
						},
						"min_capacity": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  2,
						},
						"seconds_until_auto_pause": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      300,
							ValidateFunc: validation.IntBetween(300, 86400),
						},
					},
				},
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Allow configuration to be unset when using engine_mode serverless, as its required to be true
					// InvalidParameterCombination: Aurora Serverless DB clusters are always encrypted at rest. Encryption can't be disabled.
					if d.Get("engine_mode").(string) != "serverless" {
						return false
					}
					if new != "false" {
						return false
					}
					return true
				},
			},

			"s3_import": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ConflictsWith: []string{
					"snapshot_identifier",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"bucket_prefix": {
							Type:     schema.TypeString,
							Required: false,
							Optional: true,
							ForceNew: true,
						},
						"ingestion_role": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"source_engine": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"source_engine_version": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
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
				ForceNew: true,
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

			"enabled_cloudwatch_logs_exports": {
				Type:     schema.TypeList,
				Computed: false,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"audit",
						"error",
						"general",
						"slowquery",
					}, false),
				},
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

	// Some API calls (e.g. RestoreDBClusterFromSnapshot do not support all
	// parameters to correctly apply all settings in one pass. For missing
	// parameters or unsupported configurations, we may need to call
	// ModifyDBInstance afterwards to prevent Terraform operators from API
	// errors or needing to double apply.
	var requiresModifyDbCluster bool
	modifyDbClusterInput := &rds.ModifyDBClusterInput{
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
		opts := rds.RestoreDBClusterFromSnapshotInput{
			DBClusterIdentifier:  aws.String(identifier),
			DeletionProtection:   aws.Bool(d.Get("deletion_protection").(bool)),
			Engine:               aws.String(d.Get("engine").(string)),
			EngineMode:           aws.String(d.Get("engine_mode").(string)),
			ScalingConfiguration: expandRdsScalingConfiguration(d.Get("scaling_configuration").([]interface{})),
			SnapshotIdentifier:   aws.String(d.Get("snapshot_identifier").(string)),
			Tags:                 tags,
		}

		if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
			opts.AvailabilityZones = expandStringList(attr.List())
		}

		// Need to check value > 0 due to:
		// InvalidParameterValue: Backtrack is not enabled for the aurora-postgresql engine.
		if v, ok := d.GetOk("backtrack_window"); ok && v.(int) > 0 {
			opts.BacktrackWindow = aws.Int64(int64(v.(int)))
		}

		if attr, ok := d.GetOk("backup_retention_period"); ok {
			modifyDbClusterInput.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
			requiresModifyDbCluster = true
		}

		if attr, ok := d.GetOk("database_name"); ok {
			opts.DatabaseName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_cluster_parameter_group_name"); ok {
			opts.DBClusterParameterGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
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

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))
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

		log.Printf("[DEBUG] RDS Cluster restore from snapshot configuration: %s", opts)
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
			return fmt.Errorf("Error creating RDS Cluster: %s", err)
		}
	} else if _, ok := d.GetOk("replication_source_identifier"); ok {
		createOpts := &rds.CreateDBClusterInput{
			DBClusterIdentifier:         aws.String(identifier),
			DeletionProtection:          aws.Bool(d.Get("deletion_protection").(bool)),
			Engine:                      aws.String(d.Get("engine").(string)),
			EngineMode:                  aws.String(d.Get("engine_mode").(string)),
			ReplicationSourceIdentifier: aws.String(d.Get("replication_source_identifier").(string)),
			ScalingConfiguration:        expandRdsScalingConfiguration(d.Get("scaling_configuration").([]interface{})),
			Tags:                        tags,
		}

		// Need to check value > 0 due to:
		// InvalidParameterValue: Backtrack is not enabled for the aurora-postgresql engine.
		if v, ok := d.GetOk("backtrack_window"); ok && v.(int) > 0 {
			createOpts.BacktrackWindow = aws.Int64(int64(v.(int)))
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

		if attr, ok := d.GetOkExists("storage_encrypted"); ok {
			createOpts.StorageEncrypted = aws.Bool(attr.(bool))
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			createOpts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		log.Printf("[DEBUG] Create RDS Cluster as read replica: %s", createOpts)
		var resp *rds.CreateDBClusterOutput
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
			return fmt.Errorf("error creating RDS cluster: %s", err)
		}

		log.Printf("[DEBUG]: RDS Cluster create response: %s", resp)

	} else if v, ok := d.GetOk("s3_import"); ok {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "master_password": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "master_username": required field is not set`, d.Get("name").(string))
		}
		s3_bucket := v.([]interface{})[0].(map[string]interface{})
		createOpts := &rds.RestoreDBClusterFromS3Input{
			DBClusterIdentifier: aws.String(identifier),
			DeletionProtection:  aws.Bool(d.Get("deletion_protection").(bool)),
			Engine:              aws.String(d.Get("engine").(string)),
			MasterUsername:      aws.String(d.Get("master_username").(string)),
			MasterUserPassword:  aws.String(d.Get("master_password").(string)),
			S3BucketName:        aws.String(s3_bucket["bucket_name"].(string)),
			S3IngestionRoleArn:  aws.String(s3_bucket["ingestion_role"].(string)),
			S3Prefix:            aws.String(s3_bucket["bucket_prefix"].(string)),
			SourceEngine:        aws.String(s3_bucket["source_engine"].(string)),
			SourceEngineVersion: aws.String(s3_bucket["source_engine_version"].(string)),
			Tags:                tags,
		}

		// Need to check value > 0 due to:
		// InvalidParameterValue: Backtrack is not enabled for the aurora-postgresql engine.
		if v, ok := d.GetOk("backtrack_window"); ok && v.(int) > 0 {
			createOpts.BacktrackWindow = aws.Int64(int64(v.(int)))
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

		if attr, ok := d.GetOk("iam_database_authentication_enabled"); ok {
			createOpts.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			createOpts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOkExists("storage_encrypted"); ok {
			createOpts.StorageEncrypted = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] RDS Cluster restore options: %s", createOpts)
		// Retry for IAM/S3 eventual consistency
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			resp, err := conn.RestoreDBClusterFromS3(createOpts)
			if err != nil {
				// InvalidParameterValue: Files from the specified Amazon S3 bucket cannot be downloaded.
				// Make sure that you have created an AWS Identity and Access Management (IAM) role that lets Amazon RDS access Amazon S3 for you.
				if isAWSErr(err, "InvalidParameterValue", "Files from the specified Amazon S3 bucket cannot be downloaded") {
					return resource.RetryableError(err)
				}
				if isAWSErr(err, "InvalidParameterValue", "S3_SNAPSHOT_INGESTION") {
					return resource.RetryableError(err)
				}
				if isAWSErr(err, "InvalidParameterValue", "S3 bucket cannot be found") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			log.Printf("[DEBUG]: RDS Cluster create response: %s", resp)
			return nil
		})

		if err != nil {
			log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
			return err
		}

	} else {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_rds_cluster: %s: "master_password": required field is not set`, d.Get("database_name").(string))
		}

		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_rds_cluster: %s: "master_username": required field is not set`, d.Get("database_name").(string))
		}

		createOpts := &rds.CreateDBClusterInput{
			DBClusterIdentifier:  aws.String(identifier),
			DeletionProtection:   aws.Bool(d.Get("deletion_protection").(bool)),
			Engine:               aws.String(d.Get("engine").(string)),
			EngineMode:           aws.String(d.Get("engine_mode").(string)),
			MasterUserPassword:   aws.String(d.Get("master_password").(string)),
			MasterUsername:       aws.String(d.Get("master_username").(string)),
			ScalingConfiguration: expandRdsScalingConfiguration(d.Get("scaling_configuration").([]interface{})),
			Tags:                 tags,
		}

		// Need to check value > 0 due to:
		// InvalidParameterValue: Backtrack is not enabled for the aurora-postgresql engine.
		if v, ok := d.GetOk("backtrack_window"); ok && v.(int) > 0 {
			createOpts.BacktrackWindow = aws.Int64(int64(v.(int)))
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

		if attr, ok := d.GetOk("engine_version"); ok {
			createOpts.EngineVersion = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("global_cluster_identifier"); ok {
			createOpts.GlobalClusterIdentifier = aws.String(attr.(string))
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

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			createOpts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOk("scaling_configuration"); ok && len(attr.([]interface{})) > 0 {
			createOpts.ScalingConfiguration = expandRdsScalingConfiguration(attr.([]interface{}))
		}

		if attr, ok := d.GetOkExists("storage_encrypted"); ok {
			createOpts.StorageEncrypted = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] RDS Cluster create options: %s", createOpts)
		var resp *rds.CreateDBClusterOutput
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
			return fmt.Errorf("error creating RDS cluster: %s", err)
		}

		log.Printf("[DEBUG]: RDS Cluster create response: %s", resp)
	}

	d.SetId(identifier)

	log.Printf("[INFO] RDS Cluster ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for RDS Cluster to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsRdsClusterCreatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for RDS Cluster state to be \"available\": %s", err)
	}

	if v, ok := d.GetOk("iam_roles"); ok {
		for _, role := range v.(*schema.Set).List() {
			err := setIamRoleToRdsCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}
	}

	if requiresModifyDbCluster {
		modifyDbClusterInput.DBClusterIdentifier = aws.String(d.Id())

		log.Printf("[INFO] RDS Cluster (%s) configuration requires ModifyDBCluster: %s", d.Id(), modifyDbClusterInput)
		_, err := conn.ModifyDBCluster(modifyDbClusterInput)
		if err != nil {
			return fmt.Errorf("error modifying RDS Cluster (%s): %s", d.Id(), err)
		}

		log.Printf("[INFO] Waiting for RDS Cluster (%s) to be available", d.Id())
		err = waitForRDSClusterUpdate(conn, d.Id(), d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return fmt.Errorf("error waiting for RDS Cluster (%s) to be available: %s", d.Id(), err)
		}
	}

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Describing RDS Cluster: %s", input)
	resp, err := conn.DescribeDBClusters(input)

	if isAWSErr(err, rds.ErrCodeDBClusterNotFoundFault, "") {
		log.Printf("[WARN] RDS Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error describing RDS Cluster (%s): %s", d.Id(), err)
	}

	if resp == nil {
		return fmt.Errorf("Error retrieving RDS cluster: empty response for: %s", input)
	}

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if aws.StringValue(c.DBClusterIdentifier) == d.Id() {
			dbc = c
			break
		}
	}

	if dbc == nil {
		log.Printf("[WARN] RDS Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}

	d.Set("arn", dbc.DBClusterArn)
	d.Set("backtrack_window", int(aws.Int64Value(dbc.BacktrackWindow)))
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

	// Only set the DatabaseName if it is not nil. There is a known API bug where
	// RDS accepts a DatabaseName but does not return it, causing a perpetual
	// diff.
	//	See https://github.com/hashicorp/terraform/issues/4671 for backstory
	if dbc.DatabaseName != nil {
		d.Set("database_name", dbc.DatabaseName)
	}

	d.Set("db_cluster_parameter_group_name", dbc.DBClusterParameterGroup)
	d.Set("db_subnet_group_name", dbc.DBSubnetGroup)
	d.Set("deletion_protection", dbc.DeletionProtection)

	if err := d.Set("enabled_cloudwatch_logs_exports", aws.StringValueSlice(dbc.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("error setting enabled_cloudwatch_logs_exports: %s", err)
	}

	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine_mode", dbc.EngineMode)
	d.Set("engine_version", dbc.EngineVersion)
	d.Set("engine", dbc.Engine)
	d.Set("hosted_zone_id", dbc.HostedZoneId)
	d.Set("iam_database_authentication_enabled", dbc.IAMDatabaseAuthenticationEnabled)

	var roles []string
	for _, r := range dbc.AssociatedRoles {
		roles = append(roles, aws.StringValue(r.RoleArn))
	}
	if err := d.Set("iam_roles", roles); err != nil {
		return fmt.Errorf("error setting iam_roles: %s", err)
	}

	d.Set("kms_key_id", dbc.KmsKeyId)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)
	d.Set("reader_endpoint", dbc.ReaderEndpoint)
	d.Set("replication_source_identifier", dbc.ReplicationSourceIdentifier)

	if err := d.Set("scaling_configuration", flattenRdsScalingConfigurationInfo(dbc.ScalingConfigurationInfo)); err != nil {
		return fmt.Errorf("error setting scaling_configuration: %s", err)
	}

	d.Set("storage_encrypted", dbc.StorageEncrypted)

	var vpcg []string
	for _, g := range dbc.VpcSecurityGroups {
		vpcg = append(vpcg, aws.StringValue(g.VpcSecurityGroupId))
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("error setting vpc_security_group_ids: %s", err)
	}

	// Fetch and save tags
	if err := saveTagsRDS(conn, d, aws.StringValue(dbc.DBClusterArn)); err != nil {
		log.Printf("[WARN] Failed to save tags for RDS Cluster (%s): %s", aws.StringValue(dbc.DBClusterIdentifier), err)
	}

	// Fetch and save Global Cluster if engine mode global
	d.Set("global_cluster_identifier", "")

	if aws.StringValue(dbc.EngineMode) == "global" {
		globalCluster, err := rdsDescribeGlobalClusterFromDbClusterARN(conn, aws.StringValue(dbc.DBClusterArn))

		if err != nil {
			return fmt.Errorf("error reading RDS Global Cluster information for DB Cluster (%s): %s", d.Id(), err)
		}

		if globalCluster != nil {
			d.Set("global_cluster_identifier", globalCluster.GlobalClusterIdentifier)
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

	if d.HasChange("backtrack_window") {
		req.BacktrackWindow = aws.Int64(int64(d.Get("backtrack_window").(int)))
		requestUpdate = true
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

	if d.HasChange("deletion_protection") {
		req.DeletionProtection = aws.Bool(d.Get("deletion_protection").(bool))
		requestUpdate = true
	}

	if d.HasChange("iam_database_authentication_enabled") {
		req.EnableIAMDatabaseAuthentication = aws.Bool(d.Get("iam_database_authentication_enabled").(bool))
		requestUpdate = true
	}

	if d.HasChange("enabled_cloudwatch_logs_exports") {
		d.SetPartial("enabled_cloudwatch_logs_exports")
		req.CloudwatchLogsExportConfiguration = buildCloudwatchLogsExportConfiguration(d)
		requestUpdate = true
	}

	if d.HasChange("scaling_configuration") {
		d.SetPartial("scaling_configuration")
		req.ScalingConfiguration = expandRdsScalingConfiguration(d.Get("scaling_configuration").([]interface{}))
		requestUpdate = true
	}

	if requestUpdate {
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err := conn.ModifyDBCluster(req)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}

				if isAWSErr(err, rds.ErrCodeInvalidDBClusterStateFault, "Cannot modify engine version without a primary instance in DB cluster") {
					return resource.NonRetryableError(err)
				}

				if isAWSErr(err, rds.ErrCodeInvalidDBClusterStateFault, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Failed to modify RDS Cluster (%s): %s", d.Id(), err)
		}

		log.Printf("[INFO] Waiting for RDS Cluster (%s) to be available", d.Id())
		err = waitForRDSClusterUpdate(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return fmt.Errorf("error waiting for RDS Cluster (%s) to be available: %s", d.Id(), err)
		}
	}

	if d.HasChange("global_cluster_identifier") {
		oRaw, nRaw := d.GetChange("global_cluster_identifier")
		o := oRaw.(string)
		n := nRaw.(string)

		if o == "" {
			return errors.New("Existing RDS Clusters cannot be added to an existing RDS Global Cluster")
		}

		if n != "" {
			return errors.New("Existing RDS Clusters cannot be migrated between existing RDS Global Clusters")
		}

		input := &rds.RemoveFromGlobalClusterInput{
			DbClusterIdentifier:     aws.String(d.Get("arn").(string)),
			GlobalClusterIdentifier: aws.String(o),
		}

		log.Printf("[DEBUG] Removing RDS Cluster from RDS Global Cluster: %s", input)
		if _, err := conn.RemoveFromGlobalCluster(input); err != nil {
			return fmt.Errorf("error removing RDS Cluster (%s) from RDS Global Cluster: %s", d.Id(), err)
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

	if d.HasChange("tags") {
		if err := setTagsRDS(conn, d, d.Get("arn").(string)); err != nil {
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

	// Automatically remove from global cluster to bypass this error on deletion:
	// InvalidDBClusterStateFault: This cluster is a part of a global cluster, please remove it from globalcluster first
	if d.Get("global_cluster_identifier").(string) != "" {
		input := &rds.RemoveFromGlobalClusterInput{
			DbClusterIdentifier:     aws.String(d.Get("arn").(string)),
			GlobalClusterIdentifier: aws.String(d.Get("global_cluster_identifier").(string)),
		}

		log.Printf("[DEBUG] Removing RDS Cluster from RDS Global Cluster: %s", input)
		_, err := conn.RemoveFromGlobalCluster(input)

		if err != nil && !isAWSErr(err, rds.ErrCodeGlobalClusterNotFoundFault, "") {
			return fmt.Errorf("error removing RDS Cluster (%s) from RDS Global Cluster: %s", d.Id(), err)
		}
	}

	deleteOpts := rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalSnapshot = aws.Bool(skipFinalSnapshot)

	if !skipFinalSnapshot {
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
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting RDS Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSClusterStateRefreshFunc(conn *rds.RDS, dbClusterIdentifier string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(dbClusterIdentifier),
		})

		if isAWSErr(err, rds.ErrCodeDBClusterNotFoundFault, "") {
			return 42, "destroyed", nil
		}

		if err != nil {
			return nil, "", err
		}

		var dbc *rds.DBCluster

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

func setIamRoleToRdsCluster(clusterIdentifier string, roleArn string, conn *rds.RDS) error {
	params := &rds.AddRoleToDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.AddRoleToDBCluster(params)
	return err
}

func removeIamRoleFromRdsCluster(clusterIdentifier string, roleArn string, conn *rds.RDS) error {
	params := &rds.RemoveRoleFromDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.RemoveRoleFromDBCluster(params)
	return err
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

var resourceAwsRdsClusterUpdatePendingStates = []string{
	"backing-up",
	"modifying",
	"resetting-master-credentials",
	"upgrading",
}

func waitForRDSClusterUpdate(conn *rds.RDS, id string, timeout time.Duration) error {
	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsRdsClusterUpdatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(conn, id),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	return err
}

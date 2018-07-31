package aws

import (
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

func resourceAwsDbInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbInstanceCreate,
		Read:   resourceAwsDbInstanceRead,
		Update: resourceAwsDbInstanceUpdate,
		Delete: resourceAwsDbInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsDbInstanceImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(80 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"username": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"engine": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
			},

			"engine_version": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: suppressAwsDbEngineVersionDiffs,
			},

			"character_set_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"allocated_storage": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"storage_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

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

			"instance_class": {
				Type:     schema.TypeString,
				Required: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"backup_retention_period": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"backup_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},

			"iops": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"license_model": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"maintenance_window": {
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

			"multi_az": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"security_group_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
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

			"s3_import": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ConflictsWith: []string{
					"snapshot_identifier",
					"replicate_source_db",
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

			"skip_final_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"copy_tags_to_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"db_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"hosted_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
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

			"replicate_source_db": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"replicas": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"snapshot_identifier": {
				Type:     schema.TypeString,
				Computed: false,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"allow_major_version_upgrade": {
				Type:     schema.TypeBool,
				Computed: false,
				Optional: true,
			},

			"monitoring_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"monitoring_interval": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"option_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"timezone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"iam_database_authentication_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ca_cert_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enabled_cloudwatch_logs_exports": {
				Type:     schema.TypeList,
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

func resourceAwsDbInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	var identifier string
	if v, ok := d.GetOk("identifier"); ok {
		identifier = v.(string)
	} else {
		if v, ok := d.GetOk("identifier_prefix"); ok {
			identifier = resource.PrefixedUniqueId(v.(string))
		} else {
			identifier = resource.UniqueId()
		}

		// SQL Server identifier size is max 15 chars, so truncate
		if engine := d.Get("engine").(string); engine != "" {
			if strings.Contains(strings.ToLower(engine), "sqlserver") {
				identifier = identifier[:15]
			}
		}
		d.Set("identifier", identifier)
	}

	if v, ok := d.GetOk("replicate_source_db"); ok {
		opts := rds.CreateDBInstanceReadReplicaInput{
			SourceDBInstanceIdentifier: aws.String(v.(string)),
			CopyTagsToSnapshot:         aws.Bool(d.Get("copy_tags_to_snapshot").(bool)),
			DBInstanceClass:            aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier:       aws.String(identifier),
			PubliclyAccessible:         aws.Bool(d.Get("publicly_accessible").(bool)),
			Tags:                       tags,
		}
		if attr, ok := d.GetOk("iops"); ok {
			opts.Iops = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("storage_type"); ok {
			opts.StorageType = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			opts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			opts.KmsKeyId = aws.String(attr.(string))
			if arnParts := strings.Split(v.(string), ":"); len(arnParts) >= 4 {
				opts.SourceRegion = aws.String(arnParts[3])
			}
		}

		if attr, ok := d.GetOk("monitoring_role_arn"); ok {
			opts.MonitoringRoleArn = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("monitoring_interval"); ok {
			opts.MonitoringInterval = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))
		}

		log.Printf("[DEBUG] DB Instance Replica create configuration: %#v", opts)
		_, err := conn.CreateDBInstanceReadReplica(&opts)
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}
	} else if v, ok := d.GetOk("s3_import"); ok {

		if _, ok := d.GetOk("allocated_storage"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "allocated_storage": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("engine"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "engine": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("password"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "password": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("username"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "username": required field is not set`, d.Get("name").(string))
		}

		s3_bucket := v.([]interface{})[0].(map[string]interface{})
		opts := rds.RestoreDBInstanceFromS3Input{
			AllocatedStorage:        aws.Int64(int64(d.Get("allocated_storage").(int))),
			AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
			CopyTagsToSnapshot:      aws.Bool(d.Get("copy_tags_to_snapshot").(bool)),
			DBName:                  aws.String(d.Get("name").(string)),
			DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier:    aws.String(d.Get("identifier").(string)),
			Engine:                  aws.String(d.Get("engine").(string)),
			EngineVersion:           aws.String(d.Get("engine_version").(string)),
			S3BucketName:            aws.String(s3_bucket["bucket_name"].(string)),
			S3Prefix:                aws.String(s3_bucket["bucket_prefix"].(string)),
			S3IngestionRoleArn:      aws.String(s3_bucket["ingestion_role"].(string)),
			MasterUsername:          aws.String(d.Get("username").(string)),
			MasterUserPassword:      aws.String(d.Get("password").(string)),
			PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
			StorageEncrypted:        aws.Bool(d.Get("storage_encrypted").(bool)),
			SourceEngine:            aws.String(s3_bucket["source_engine"].(string)),
			SourceEngineVersion:     aws.String(s3_bucket["source_engine_version"].(string)),
			Tags:                    tags,
		}

		if attr, ok := d.GetOk("multi_az"); ok {
			opts.MultiAZ = aws.Bool(attr.(bool))

		}

		if _, ok := d.GetOk("character_set_name"); ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "character_set_name" doesn't work with with restores"`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("timezone"); ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "timezone" doesn't work with with restores"`, d.Get("name").(string))
		}

		attr := d.Get("backup_retention_period")
		opts.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))

		if attr, ok := d.GetOk("maintenance_window"); ok {
			opts.PreferredMaintenanceWindow = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("backup_window"); ok {
			opts.PreferredBackupWindow = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("license_model"); ok {
			opts.LicenseModel = aws.String(attr.(string))
		}
		if attr, ok := d.GetOk("parameter_group_name"); ok {
			opts.DBParameterGroupName = aws.String(attr.(string))
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			opts.VpcSecurityGroupIds = s
		}

		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			opts.DBSecurityGroups = s
		}
		if attr, ok := d.GetOk("storage_type"); ok {
			opts.StorageType = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("iops"); ok {
			opts.Iops = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("monitoring_role_arn"); ok {
			opts.MonitoringRoleArn = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("monitoring_interval"); ok {
			opts.MonitoringInterval = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			opts.KmsKeyId = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("iam_database_authentication_enabled"); ok {
			opts.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] DB Instance S3 Restore configuration: %#v", opts)
		var err error
		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err = conn.RestoreDBInstanceFromS3(&opts)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "ENHANCED_MONITORING") {
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
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}

		d.SetId(d.Get("identifier").(string))

		log.Printf("[INFO] DB Instance ID: %s", d.Id())

		log.Println(
			"[INFO] Waiting for DB Instance to be available")

		stateConf := &resource.StateChangeConf{
			Pending:    resourceAwsDbInstanceCreatePendingStates,
			Target:     []string{"available", "storage-optimization"},
			Refresh:    resourceAwsDbInstanceStateRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

		return resourceAwsDbInstanceRead(d, meta)
	} else if _, ok := d.GetOk("snapshot_identifier"); ok {
		opts := rds.RestoreDBInstanceFromDBSnapshotInput{
			DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier:    aws.String(d.Get("identifier").(string)),
			DBSnapshotIdentifier:    aws.String(d.Get("snapshot_identifier").(string)),
			AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
			PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
			Tags:                    tags,
			CopyTagsToSnapshot:      aws.Bool(d.Get("copy_tags_to_snapshot").(bool)),
		}

		if attr, ok := d.GetOk("name"); ok {
			// "Note: This parameter [DBName] doesn't apply to the MySQL, PostgreSQL, or MariaDB engines."
			// https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_RestoreDBInstanceFromDBSnapshot.html
			switch strings.ToLower(d.Get("engine").(string)) {
			case "mysql", "postgres", "mariadb":
				// skip
			default:
				opts.DBName = aws.String(attr.(string))
			}
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			opts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOk("engine"); ok {
			opts.Engine = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("iops"); ok {
			opts.Iops = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("license_model"); ok {
			opts.LicenseModel = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("multi_az"); ok {
			opts.MultiAZ = aws.Bool(attr.(bool))
		}

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))

		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}
		if attr, ok := d.GetOk("tde_credential_arn"); ok {
			opts.TdeCredentialArn = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("storage_type"); ok {
			opts.StorageType = aws.String(attr.(string))
		}

		log.Printf("[DEBUG] DB Instance restore from snapshot configuration: %s", opts)
		_, err := conn.RestoreDBInstanceFromDBSnapshot(&opts)
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}

		var sgUpdate bool
		var passwordUpdate bool

		if _, ok := d.GetOk("password"); ok {
			passwordUpdate = true
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			sgUpdate = true
		}
		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			sgUpdate = true
		}
		if sgUpdate || passwordUpdate {
			log.Printf("[INFO] DB is restoring from snapshot with default security, but custom security should be set, will now update after snapshot is restored!")

			// wait for instance to get up and then modify security
			d.SetId(d.Get("identifier").(string))

			log.Printf("[INFO] DB Instance ID: %s", d.Id())

			log.Println(
				"[INFO] Waiting for DB Instance to be available")

			stateConf := &resource.StateChangeConf{
				Pending:    resourceAwsDbInstanceCreatePendingStates,
				Target:     []string{"available", "storage-optimization"},
				Refresh:    resourceAwsDbInstanceStateRefreshFunc(d.Id(), conn),
				Timeout:    d.Timeout(schema.TimeoutCreate),
				MinTimeout: 10 * time.Second,
				Delay:      30 * time.Second, // Wait 30 secs before starting
			}

			// Wait, catching any errors
			_, err := stateConf.WaitForState()
			if err != nil {
				return err
			}

			err = resourceAwsDbInstanceUpdate(d, meta)
			if err != nil {
				return err
			}

		}
	} else {
		if _, ok := d.GetOk("allocated_storage"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "allocated_storage": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("engine"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "engine": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("password"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "password": required field is not set`, d.Get("name").(string))
		}
		if _, ok := d.GetOk("username"); !ok {
			return fmt.Errorf(`provider.aws: aws_db_instance: %s: "username": required field is not set`, d.Get("name").(string))
		}
		opts := rds.CreateDBInstanceInput{
			AllocatedStorage:        aws.Int64(int64(d.Get("allocated_storage").(int))),
			DBName:                  aws.String(d.Get("name").(string)),
			DBInstanceClass:         aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier:    aws.String(d.Get("identifier").(string)),
			MasterUsername:          aws.String(d.Get("username").(string)),
			MasterUserPassword:      aws.String(d.Get("password").(string)),
			Engine:                  aws.String(d.Get("engine").(string)),
			EngineVersion:           aws.String(d.Get("engine_version").(string)),
			StorageEncrypted:        aws.Bool(d.Get("storage_encrypted").(bool)),
			AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
			PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
			Tags:                    tags,
			CopyTagsToSnapshot:      aws.Bool(d.Get("copy_tags_to_snapshot").(bool)),
		}

		attr := d.Get("backup_retention_period")
		opts.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
		if attr, ok := d.GetOk("multi_az"); ok {
			opts.MultiAZ = aws.Bool(attr.(bool))

		}

		if attr, ok := d.GetOk("character_set_name"); ok {
			opts.CharacterSetName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("timezone"); ok {
			opts.Timezone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("maintenance_window"); ok {
			opts.PreferredMaintenanceWindow = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("backup_window"); ok {
			opts.PreferredBackupWindow = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("license_model"); ok {
			opts.LicenseModel = aws.String(attr.(string))
		}
		if attr, ok := d.GetOk("parameter_group_name"); ok {
			opts.DBParameterGroupName = aws.String(attr.(string))
		}

		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			opts.VpcSecurityGroupIds = s
		}

		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			opts.DBSecurityGroups = s
		}
		if attr, ok := d.GetOk("storage_type"); ok {
			opts.StorageType = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("enabled_cloudwatch_logs_exports"); ok && len(attr.([]interface{})) > 0 {
			opts.EnableCloudwatchLogsExports = expandStringList(attr.([]interface{}))
		}

		if attr, ok := d.GetOk("iops"); ok {
			opts.Iops = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("monitoring_role_arn"); ok {
			opts.MonitoringRoleArn = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("monitoring_interval"); ok {
			opts.MonitoringInterval = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("option_group_name"); ok {
			opts.OptionGroupName = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("kms_key_id"); ok {
			opts.KmsKeyId = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("iam_database_authentication_enabled"); ok {
			opts.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] DB Instance create configuration: %#v", opts)
		var err error
		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err = conn.CreateDBInstance(&opts)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "ENHANCED_MONITORING") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			if isAWSErr(err, "InvalidParameterValue", "") {
				return fmt.Errorf("Error creating DB Instance: %s, %+v", err, opts)
			}
			return fmt.Errorf("Error creating DB Instance: %s", err)

		}
	}

	d.SetId(d.Get("identifier").(string))

	log.Printf("[INFO] DB Instance ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for DB Instance to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsDbInstanceCreatePendingStates,
		Target:     []string{"available", "storage-optimization"},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDbInstanceRead(d, meta)
}

func resourceAwsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsDbInstanceRetrieve(d.Id(), meta.(*AWSClient).rdsconn)

	if err != nil {
		return err
	}
	if v == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", v.DBName)
	d.Set("identifier", v.DBInstanceIdentifier)
	d.Set("resource_id", v.DbiResourceId)
	d.Set("username", v.MasterUsername)
	d.Set("engine", v.Engine)
	d.Set("engine_version", v.EngineVersion)
	d.Set("allocated_storage", v.AllocatedStorage)
	d.Set("iops", v.Iops)
	d.Set("copy_tags_to_snapshot", v.CopyTagsToSnapshot)
	d.Set("auto_minor_version_upgrade", v.AutoMinorVersionUpgrade)
	d.Set("storage_type", v.StorageType)
	d.Set("instance_class", v.DBInstanceClass)
	d.Set("availability_zone", v.AvailabilityZone)
	d.Set("backup_retention_period", v.BackupRetentionPeriod)
	d.Set("backup_window", v.PreferredBackupWindow)
	d.Set("license_model", v.LicenseModel)
	d.Set("maintenance_window", v.PreferredMaintenanceWindow)
	d.Set("publicly_accessible", v.PubliclyAccessible)
	d.Set("multi_az", v.MultiAZ)
	d.Set("kms_key_id", v.KmsKeyId)
	d.Set("port", v.DbInstancePort)
	d.Set("iam_database_authentication_enabled", v.IAMDatabaseAuthenticationEnabled)
	if v.DBSubnetGroup != nil {
		d.Set("db_subnet_group_name", v.DBSubnetGroup.DBSubnetGroupName)
	}

	if v.CharacterSetName != nil {
		d.Set("character_set_name", v.CharacterSetName)
	}

	d.Set("timezone", v.Timezone)

	if len(v.DBParameterGroups) > 0 {
		d.Set("parameter_group_name", v.DBParameterGroups[0].DBParameterGroupName)
	}

	if v.Endpoint != nil {
		d.Set("port", v.Endpoint.Port)
		d.Set("address", v.Endpoint.Address)
		d.Set("hosted_zone_id", v.Endpoint.HostedZoneId)
		if v.Endpoint.Address != nil && v.Endpoint.Port != nil {
			d.Set("endpoint",
				fmt.Sprintf("%s:%d", *v.Endpoint.Address, *v.Endpoint.Port))
		}
	}

	d.Set("status", v.DBInstanceStatus)
	d.Set("storage_encrypted", v.StorageEncrypted)
	if v.OptionGroupMemberships != nil {
		d.Set("option_group_name", v.OptionGroupMemberships[0].OptionGroupName)
	}

	if v.MonitoringInterval != nil {
		d.Set("monitoring_interval", v.MonitoringInterval)
	}

	if v.MonitoringRoleArn != nil {
		d.Set("monitoring_role_arn", v.MonitoringRoleArn)
	}

	if err := d.Set("enabled_cloudwatch_logs_exports", flattenStringList(v.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("error setting enabled_cloudwatch_logs_exports: %s", err)
	}

	// list tags for resource
	// set tags
	conn := meta.(*AWSClient).rdsconn

	arn := aws.StringValue(v.DBInstanceArn)
	d.Set("arn", arn)
	resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		return fmt.Errorf("Error retrieving tags for ARN: %s", arn)
	}

	var dt []*rds.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}
	d.Set("tags", tagsToMapRDS(dt))

	// Create an empty schema.Set to hold all vpc security group ids
	ids := &schema.Set{
		F: schema.HashString,
	}
	for _, v := range v.VpcSecurityGroups {
		ids.Add(*v.VpcSecurityGroupId)
	}
	d.Set("vpc_security_group_ids", ids)

	// Create an empty schema.Set to hold all security group names
	sgn := &schema.Set{
		F: schema.HashString,
	}
	for _, v := range v.DBSecurityGroups {
		sgn.Add(*v.DBSecurityGroupName)
	}
	d.Set("security_group_names", sgn)
	// replica things

	var replicas []string
	for _, v := range v.ReadReplicaDBInstanceIdentifiers {
		replicas = append(replicas, *v)
	}
	if err := d.Set("replicas", replicas); err != nil {
		return fmt.Errorf("[DEBUG] Error setting replicas attribute: %#v, error: %#v", replicas, err)
	}

	d.Set("replicate_source_db", v.ReadReplicaSourceDBInstanceIdentifier)

	d.Set("ca_cert_identifier", v.CACertificateIdentifier)

	return nil
}

func resourceAwsDbInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	opts.SkipFinalSnapshot = aws.Bool(skipFinalSnapshot)

	if skipFinalSnapshot == false {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			opts.FinalDBSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("DB Instance FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] DB Instance destroy configuration: %v", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		return err
	}

	log.Println("[INFO] Waiting for DB Instance to be destroyed")
	return waitUntilAwsDbInstanceIsDeleted(d.Id(), conn, d.Timeout(schema.TimeoutDelete))
}

func waitUntilAwsDbInstanceIsDeleted(id string, conn *rds.RDS, timeout time.Duration) error {
	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsDbInstanceDeletePendingStates,
		Target:     []string{},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(id, conn),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDbInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	d.Partial(true)

	req := &rds.ModifyDBInstanceInput{
		ApplyImmediately:     aws.Bool(d.Get("apply_immediately").(bool)),
		DBInstanceIdentifier: aws.String(d.Id()),
	}
	d.SetPartial("apply_immediately")

	if !d.Get("apply_immediately").(bool) {
		log.Println("[INFO] Only settings updating, instance changes will be applied in next maintenance window")
	}

	requestUpdate := false
	if d.HasChange("allocated_storage") || d.HasChange("iops") {
		d.SetPartial("allocated_storage")
		d.SetPartial("iops")
		req.Iops = aws.Int64(int64(d.Get("iops").(int)))
		req.AllocatedStorage = aws.Int64(int64(d.Get("allocated_storage").(int)))
		requestUpdate = true
	}
	if d.HasChange("allow_major_version_upgrade") {
		d.SetPartial("allow_major_version_upgrade")
		req.AllowMajorVersionUpgrade = aws.Bool(d.Get("allow_major_version_upgrade").(bool))
		requestUpdate = true
	}
	if d.HasChange("backup_retention_period") {
		d.SetPartial("backup_retention_period")
		req.BackupRetentionPeriod = aws.Int64(int64(d.Get("backup_retention_period").(int)))
		requestUpdate = true
	}
	if d.HasChange("copy_tags_to_snapshot") {
		d.SetPartial("copy_tags_to_snapshot")
		req.CopyTagsToSnapshot = aws.Bool(d.Get("copy_tags_to_snapshot").(bool))
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
		req.AllowMajorVersionUpgrade = aws.Bool(d.Get("allow_major_version_upgrade").(bool))
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
		req.MultiAZ = aws.Bool(d.Get("multi_az").(bool))
		requestUpdate = true
	}
	if d.HasChange("publicly_accessible") {
		d.SetPartial("publicly_accessible")
		req.PubliclyAccessible = aws.Bool(d.Get("publicly_accessible").(bool))
		requestUpdate = true
	}
	if d.HasChange("storage_type") {
		d.SetPartial("storage_type")
		req.StorageType = aws.String(d.Get("storage_type").(string))
		requestUpdate = true

		if *req.StorageType == "io1" {
			req.Iops = aws.Int64(int64(d.Get("iops").(int)))
		}
	}
	if d.HasChange("auto_minor_version_upgrade") {
		d.SetPartial("auto_minor_version_upgrade")
		req.AutoMinorVersionUpgrade = aws.Bool(d.Get("auto_minor_version_upgrade").(bool))
		requestUpdate = true
	}

	if d.HasChange("monitoring_role_arn") {
		d.SetPartial("monitoring_role_arn")
		req.MonitoringRoleArn = aws.String(d.Get("monitoring_role_arn").(string))
		requestUpdate = true
	}

	if d.HasChange("monitoring_interval") {
		d.SetPartial("monitoring_interval")
		req.MonitoringInterval = aws.Int64(int64(d.Get("monitoring_interval").(int)))
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			req.VpcSecurityGroupIds = s
		}
		requestUpdate = true
	}

	if d.HasChange("security_group_names") {
		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			var s []*string
			for _, v := range attr.List() {
				s = append(s, aws.String(v.(string)))
			}
			req.DBSecurityGroups = s
		}
		requestUpdate = true
	}

	if d.HasChange("option_group_name") {
		d.SetPartial("option_group_name")
		req.OptionGroupName = aws.String(d.Get("option_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("port") {
		d.SetPartial("port")
		req.DBPortNumber = aws.Int64(int64(d.Get("port").(int)))
		requestUpdate = true
	}
	if d.HasChange("db_subnet_group_name") && !d.IsNewResource() {
		d.SetPartial("db_subnet_group_name")
		req.DBSubnetGroupName = aws.String(d.Get("db_subnet_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("enabled_cloudwatch_logs_exports") && !d.IsNewResource() {
		d.SetPartial("enabled_cloudwatch_logs_exports")
		req.CloudwatchLogsExportConfiguration = buildCloudwatchLogsExportConfiguration(d)
		requestUpdate = true
	}

	if d.HasChange("iam_database_authentication_enabled") {
		req.EnableIAMDatabaseAuthentication = aws.Bool(d.Get("iam_database_authentication_enabled").(bool))
		requestUpdate = true
	}

	log.Printf("[DEBUG] Send DB Instance Modification request: %t", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] DB Instance Modification request: %s", req)
		_, err := conn.ModifyDBInstance(req)
		if err != nil {
			return fmt.Errorf("Error modifying DB Instance %s: %s", d.Id(), err)
		}

		log.Println("[INFO] Waiting for DB Instance to be available")

		stateConf := &resource.StateChangeConf{
			Pending:    resourceAwsDbInstanceUpdatePendingStates,
			Target:     []string{"available", "storage-optimization"},
			Refresh:    resourceAwsDbInstanceStateRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, dbStateErr := stateConf.WaitForState()
		if dbStateErr != nil {
			return dbStateErr
		}
	}

	// separate request to promote a database
	if d.HasChange("replicate_source_db") {
		if d.Get("replicate_source_db").(string) == "" {
			// promote
			opts := rds.PromoteReadReplicaInput{
				DBInstanceIdentifier: aws.String(d.Id()),
			}
			attr := d.Get("backup_retention_period")
			opts.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
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

	if err := setTagsRDS(conn, d, d.Get("arn").(string)); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsDbInstanceRead(d, meta)
}

// resourceAwsDbInstanceRetrieve fetches DBInstance information from the AWS
// API. It returns an error if there is a communication problem or unexpected
// error with AWS. When the DBInstance is not found, it returns no error and a
// nil pointer.
func resourceAwsDbInstanceRetrieve(id string, conn *rds.RDS) (*rds.DBInstance, error) {
	opts := rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(id),
	}

	log.Printf("[DEBUG] DB Instance describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)
	if err != nil {
		if isAWSErr(err, rds.ErrCodeDBInstanceNotFoundFault, "") {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DB Instances: %s", err)
	}

	if len(resp.DBInstances) != 1 ||
		*resp.DBInstances[0].DBInstanceIdentifier != id {
		if err != nil {
			return nil, nil
		}
	}

	return resp.DBInstances[0], nil
}

func resourceAwsDbInstanceImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither skip_final_snapshot nor final_snapshot_identifier can be fetched
	// from any API call, so we need to default skip_final_snapshot to true so
	// that final_snapshot_identifier is not required
	d.Set("skip_final_snapshot", true)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsDbInstanceStateRefreshFunc(id string, conn *rds.RDS) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsDbInstanceRetrieve(id, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.DBInstanceStatus != nil {
			log.Printf("[DEBUG] DB Instance status for instance %s: %s", id, *v.DBInstanceStatus)
		}

		return v, *v.DBInstanceStatus, nil
	}
}

func buildCloudwatchLogsExportConfiguration(d *schema.ResourceData) *rds.CloudwatchLogsExportConfiguration {

	oraw, nraw := d.GetChange("enabled_cloudwatch_logs_exports")
	o := oraw.([]interface{})
	n := nraw.([]interface{})

	create, disable := diffCloudwatchLogsExportConfiguration(o, n)

	return &rds.CloudwatchLogsExportConfiguration{
		EnableLogTypes:  expandStringList(create),
		DisableLogTypes: expandStringList(disable),
	}
}

func diffCloudwatchLogsExportConfiguration(old, new []interface{}) ([]interface{}, []interface{}) {
	create := make([]interface{}, 0)
	disable := make([]interface{}, 0)

	for _, n := range new {
		if _, contains := sliceContainsString(old, n.(string)); !contains {
			create = append(create, n)
		}
	}

	for _, o := range old {
		if _, contains := sliceContainsString(new, o.(string)); !contains {
			disable = append(disable, o)
		}
	}

	return create, disable
}

// Database instance status: http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Status.html
var resourceAwsDbInstanceCreatePendingStates = []string{
	"backing-up",
	"configuring-enhanced-monitoring",
	"configuring-log-exports",
	"creating",
	"maintenance",
	"modifying",
	"rebooting",
	"renaming",
	"resetting-master-credentials",
	"starting",
	"stopping",
	"upgrading",
}

var resourceAwsDbInstanceDeletePendingStates = []string{
	"available",
	"backing-up",
	"configuring-enhanced-monitoring",
	"configuring-log-exports",
	"creating",
	"deleting",
	"incompatible-parameters",
	"modifying",
	"starting",
	"stopping",
	"storage-full",
	"storage-optimization",
}

var resourceAwsDbInstanceUpdatePendingStates = []string{
	"backing-up",
	"configuring-enhanced-monitoring",
	"configuring-log-exports",
	"creating",
	"maintenance",
	"modifying",
	"moving-to-vpc",
	"rebooting",
	"renaming",
	"resetting-master-credentials",
	"starting",
	"stopping",
	"storage-full",
	"upgrading",
}

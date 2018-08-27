package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDbInstance() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDbInstanceRead,

		Schema: map[string]*schema.Schema{
			"db_instance_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"allocated_storage": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"backup_retention_period": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"db_cluster_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_instance_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_instance_class": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_parameter_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"db_security_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"db_subnet_group": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_instance_port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"enabled_cloudwatch_logs_exports": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"hosted_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"iops": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"license_model": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"master_username": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"monitoring_interval": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"monitoring_role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"multi_az": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"option_group_memberships": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"preferred_backup_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"storage_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"timezone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_security_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"replicate_source_db": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ca_cert_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	opts := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(d.Get("db_instance_identifier").(string)),
	}

	log.Printf("[DEBUG] Reading DB Instance: %s", opts)

	resp, err := conn.DescribeDBInstances(opts)
	if err != nil {
		return err
	}

	if len(resp.DBInstances) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}
	if len(resp.DBInstances) > 1 {
		return fmt.Errorf("Your query returned more than one result. Please try a more specific search criteria.")
	}

	dbInstance := *resp.DBInstances[0]

	d.SetId(d.Get("db_instance_identifier").(string))

	d.Set("allocated_storage", dbInstance.AllocatedStorage)
	d.Set("auto_minor_upgrade_enabled", dbInstance.AutoMinorVersionUpgrade)
	d.Set("availability_zone", dbInstance.AvailabilityZone)
	d.Set("backup_retention_period", dbInstance.BackupRetentionPeriod)
	d.Set("db_cluster_identifier", dbInstance.DBClusterIdentifier)
	d.Set("db_instance_arn", dbInstance.DBInstanceArn)
	d.Set("db_instance_class", dbInstance.DBInstanceClass)
	d.Set("db_name", dbInstance.DBName)

	var parameterGroups []string
	for _, v := range dbInstance.DBParameterGroups {
		parameterGroups = append(parameterGroups, *v.DBParameterGroupName)
	}
	if err := d.Set("db_parameter_groups", parameterGroups); err != nil {
		return fmt.Errorf("Error setting db_parameter_groups attribute: %#v, error: %#v", parameterGroups, err)
	}

	var dbSecurityGroups []string
	for _, v := range dbInstance.DBSecurityGroups {
		dbSecurityGroups = append(dbSecurityGroups, *v.DBSecurityGroupName)
	}
	if err := d.Set("db_security_groups", dbSecurityGroups); err != nil {
		return fmt.Errorf("Error setting db_security_groups attribute: %#v, error: %#v", dbSecurityGroups, err)
	}

	if dbInstance.DBSubnetGroup != nil {
		d.Set("db_subnet_group", dbInstance.DBSubnetGroup.DBSubnetGroupName)
	} else {
		d.Set("db_subnet_group", "")
	}

	d.Set("db_instance_port", dbInstance.DbInstancePort)
	d.Set("engine", dbInstance.Engine)
	d.Set("engine_version", dbInstance.EngineVersion)
	d.Set("iops", dbInstance.Iops)
	d.Set("kms_key_id", dbInstance.KmsKeyId)
	d.Set("license_model", dbInstance.LicenseModel)
	d.Set("master_username", dbInstance.MasterUsername)
	d.Set("monitoring_interval", dbInstance.MonitoringInterval)
	d.Set("monitoring_role_arn", dbInstance.MonitoringRoleArn)
	d.Set("address", dbInstance.Endpoint.Address)
	d.Set("port", dbInstance.Endpoint.Port)
	d.Set("hosted_zone_id", dbInstance.Endpoint.HostedZoneId)
	d.Set("endpoint", fmt.Sprintf("%s:%d", *dbInstance.Endpoint.Address, *dbInstance.Endpoint.Port))

	if err := d.Set("enabled_cloudwatch_logs_exports", aws.StringValueSlice(dbInstance.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("error setting enabled_cloudwatch_logs_exports: %#v", err)
	}

	var optionGroups []string
	for _, v := range dbInstance.OptionGroupMemberships {
		optionGroups = append(optionGroups, *v.OptionGroupName)
	}
	if err := d.Set("option_group_memberships", optionGroups); err != nil {
		return fmt.Errorf("Error setting option_group_memberships attribute: %#v, error: %#v", optionGroups, err)
	}

	d.Set("preferred_backup_window", dbInstance.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbInstance.PreferredMaintenanceWindow)
	d.Set("publicly_accessible", dbInstance.PubliclyAccessible)
	d.Set("storage_encrypted", dbInstance.StorageEncrypted)
	d.Set("storage_type", dbInstance.StorageType)
	d.Set("timezone", dbInstance.Timezone)
	d.Set("replicate_source_db", dbInstance.ReadReplicaSourceDBInstanceIdentifier)
	d.Set("ca_cert_identifier", dbInstance.CACertificateIdentifier)

	var vpcSecurityGroups []string
	for _, v := range dbInstance.VpcSecurityGroups {
		vpcSecurityGroups = append(vpcSecurityGroups, *v.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_groups", vpcSecurityGroups); err != nil {
		return fmt.Errorf("Error setting vpc_security_groups attribute: %#v, error: %#v", vpcSecurityGroups, err)
	}

	return nil
}

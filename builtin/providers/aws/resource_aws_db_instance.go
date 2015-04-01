package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/rds"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbInstanceCreate,
		Read:   resourceAwsDbInstanceRead,
		Delete: resourceAwsDbInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"storage_encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"allocated_storage": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"storage_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"backup_retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  1,
			},

			"backup_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"iops": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"maintenance_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"multi_az": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
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

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"final_snapshot_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"db_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDbInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	opts := rds.CreateDBInstanceMessage{
		AllocatedStorage:     aws.Integer(d.Get("allocated_storage").(int)),
		DBInstanceClass:      aws.String(d.Get("instance_class").(string)),
		DBInstanceIdentifier: aws.String(d.Get("identifier").(string)),
		DBName:               aws.String(d.Get("name").(string)),
		MasterUsername:       aws.String(d.Get("username").(string)),
		MasterUserPassword:   aws.String(d.Get("password").(string)),
		Engine:               aws.String(d.Get("engine").(string)),
		EngineVersion:        aws.String(d.Get("engine_version").(string)),
		StorageEncrypted:     aws.Boolean(d.Get("storage_encrypted").(bool)),
	}

	if attr, ok := d.GetOk("storage_type"); ok {
		opts.StorageType = aws.String(attr.(string))
	}

	attr := d.Get("backup_retention_period")
	opts.BackupRetentionPeriod = aws.Integer(attr.(int))

	if attr, ok := d.GetOk("iops"); ok {
		opts.IOPS = aws.Integer(attr.(int))
	}

	if attr, ok := d.GetOk("port"); ok {
		opts.Port = aws.Integer(attr.(int))
	}

	if attr, ok := d.GetOk("multi_az"); ok {
		opts.MultiAZ = aws.Boolean(attr.(bool))
	}

	if attr, ok := d.GetOk("availability_zone"); ok {
		opts.AvailabilityZone = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("maintenance_window"); ok {
		opts.PreferredMaintenanceWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("backup_window"); ok {
		opts.PreferredBackupWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("publicly_accessible"); ok {
		opts.PubliclyAccessible = aws.Boolean(attr.(bool))
	}

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		opts.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("parameter_group_name"); ok {
		opts.DBParameterGroupName = aws.String(attr.(string))
	}

	if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
		var s []string
		for _, v := range attr.List() {
			s = append(s, v.(string))
		}
		opts.VPCSecurityGroupIDs = s
	}

	if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
		var s []string
		for _, v := range attr.List() {
			s = append(s, v.(string))
		}
		opts.DBSecurityGroups = s
	}

	log.Printf("[DEBUG] DB Instance create configuration: %#v", opts)
	_, err := conn.CreateDBInstance(&opts)
	if err != nil {
		return fmt.Errorf("Error creating DB Instance: %s", err)
	}

	d.SetId(d.Get("identifier").(string))

	log.Printf("[INFO] DB Instance ID: %s", d.Id())

	log.Println(
		"[INFO] Waiting for DB Instance to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     "available",
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDbInstanceRead(d, meta)
}

func resourceAwsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsBbInstanceRetrieve(d, meta)

	if err != nil {
		return err
	}
	if v == nil {
		d.SetId("")
		return nil
	}

	if v.DBName != nil {
		d.Set("name", *v.DBName)
	} else {
		d.Set("name", "")
	}
	d.Set("username", *v.MasterUsername)
	d.Set("engine", *v.Engine)
	d.Set("engine_version", *v.EngineVersion)
	d.Set("allocated_storage", *v.AllocatedStorage)
	d.Set("storage_type", *v.StorageType)
	d.Set("instance_class", *v.DBInstanceClass)
	d.Set("availability_zone", *v.AvailabilityZone)
	d.Set("backup_retention_period", *v.BackupRetentionPeriod)
	d.Set("backup_window", *v.PreferredBackupWindow)
	d.Set("maintenance_window", *v.PreferredMaintenanceWindow)
	d.Set("multi_az", *v.MultiAZ)
	d.Set("port", *v.Endpoint.Port)
	d.Set("db_subnet_group_name", *v.DBSubnetGroup.DBSubnetGroupName)

	if len(v.DBParameterGroups) > 0 {
		d.Set("parameter_group_name", *v.DBParameterGroups[0].DBParameterGroupName)
	}

	d.Set("address", *v.Endpoint.Address)
	d.Set("endpoint", fmt.Sprintf("%s:%d", *v.Endpoint.Address, *v.Endpoint.Port))
	d.Set("status", *v.DBInstanceStatus)
	d.Set("storage_encrypted", *v.StorageEncrypted)

	// Create an empty schema.Set to hold all vpc security group ids
	ids := &schema.Set{
		F: func(v interface{}) int {
			return hashcode.String(v.(string))
		},
	}
	for _, v := range v.VPCSecurityGroups {
		ids.Add(*v.VPCSecurityGroupID)
	}
	d.Set("vpc_security_group_ids", ids)

	// Create an empty schema.Set to hold all security group names
	sgn := &schema.Set{
		F: func(v interface{}) int {
			return hashcode.String(v.(string))
		},
	}
	for _, v := range v.DBSecurityGroups {
		sgn.Add(*v.DBSecurityGroupName)
	}
	d.Set("security_group_names", sgn)

	return nil
}

func resourceAwsDbInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceMessage{DBInstanceIdentifier: aws.String(d.Id())}

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
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func resourceAwsBbInstanceRetrieve(
	d *schema.ResourceData, meta interface{}) (*rds.DBInstance, error) {
	conn := meta.(*AWSClient).rdsconn

	opts := rds.DescribeDBInstancesMessage{
		DBInstanceIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] DB Instance describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)

	if err != nil {
		dbinstanceerr, ok := err.(aws.APIError)
		if ok && dbinstanceerr.Code == "DBInstanceNotFound" {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DB Instances: %s", err)
	}

	if len(resp.DBInstances) != 1 ||
		*resp.DBInstances[0].DBInstanceIdentifier != d.Id() {
		if err != nil {
			return nil, nil
		}
	}

	v := resp.DBInstances[0]

	return &v, nil
}

func resourceAwsDbInstanceStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsBbInstanceRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving DB Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		return v, *v.DBInstanceStatus, nil
	}
}

package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbInstanceCreate,
		Read:   resourceAwsDbInstanceRead,
		Update: resourceAwsDbInstanceUpdate,
		Delete: resourceAwsDbInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
			},

			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
			},

			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"storage_encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"allocated_storage": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"storage_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"identifier": &schema.Schema{
                                Type:         schema.TypeString,
                                Required:     true,
                                ForceNew:     true,
                                ValidateFunc: validateRdsId,
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

			"backup_retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"backup_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

			"maintenance_window": &schema.Schema{
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

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"final_snapshot_identifier": &schema.Schema{
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

			"db_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

			// apply_immediately is used to determine when the update modifications
			// take place.
			// See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html
			"apply_immediately": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"replicate_source_db": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"replicas": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"snapshot_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Computed: false,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"auto_minor_version_upgrade": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: false,
				Optional: true,
			},

			"allow_major_version_upgrade": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: false,
				Optional: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	if v, ok := d.GetOk("replicate_source_db"); ok {
		opts := rds.CreateDBInstanceReadReplicaInput{
			SourceDBInstanceIdentifier: aws.String(v.(string)),
			DBInstanceClass:            aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier:       aws.String(d.Get("identifier").(string)),
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

		if attr, ok := d.GetOk("publicly_accessible"); ok {
			opts.PubliclyAccessible = aws.Bool(attr.(bool))
		}
		_, err := conn.CreateDBInstanceReadReplica(&opts)
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}
	} else if _, ok := d.GetOk("snapshot_identifier"); ok {
		opts := rds.RestoreDBInstanceFromDBSnapshotInput{
			DBInstanceClass:      aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier: aws.String(d.Get("identifier").(string)),
			DBSnapshotIdentifier: aws.String(d.Get("snapshot_identifier").(string)),
			Tags:                 tags,
		}

		if attr, ok := d.GetOk("auto_minor_version_upgrade"); ok {
			opts.AutoMinorVersionUpgrade = aws.Bool(attr.(bool))
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("db_subnet_group_name"); ok {
			opts.DBSubnetGroupName = aws.String(attr.(string))
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

		if attr, ok := d.GetOk("publicly_accessible"); ok {
			opts.PubliclyAccessible = aws.Bool(attr.(bool))
		}

		if attr, ok := d.GetOk("tde_credential_arn"); ok {
			opts.TdeCredentialArn = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("storage_type"); ok {
			opts.StorageType = aws.String(attr.(string))
		}

		_, err := conn.RestoreDBInstanceFromDBSnapshot(&opts)
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}
	} else {
		opts := rds.CreateDBInstanceInput{
			AllocatedStorage:     aws.Int64(int64(d.Get("allocated_storage").(int))),
			DBName:               aws.String(d.Get("name").(string)),
			DBInstanceClass:      aws.String(d.Get("instance_class").(string)),
			DBInstanceIdentifier: aws.String(d.Get("identifier").(string)),
			MasterUsername:       aws.String(d.Get("username").(string)),
			MasterUserPassword:   aws.String(d.Get("password").(string)),
			Engine:               aws.String(d.Get("engine").(string)),
			EngineVersion:        aws.String(d.Get("engine_version").(string)),
			StorageEncrypted:     aws.Bool(d.Get("storage_encrypted").(bool)),
			Tags:                 tags,
		}

		attr := d.Get("backup_retention_period")
		opts.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
		if attr, ok := d.GetOk("multi_az"); ok {
			opts.MultiAZ = aws.Bool(attr.(bool))
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

		if attr, ok := d.GetOk("iops"); ok {
			opts.Iops = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("port"); ok {
			opts.Port = aws.Int64(int64(attr.(int)))
		}

		if attr, ok := d.GetOk("availability_zone"); ok {
			opts.AvailabilityZone = aws.String(attr.(string))
		}

		if attr, ok := d.GetOk("publicly_accessible"); ok {
			opts.PubliclyAccessible = aws.Bool(attr.(bool))
		}

		log.Printf("[DEBUG] DB Instance create configuration: %#v", opts)
		var err error
		_, err = conn.CreateDBInstance(&opts)
		if err != nil {
			return fmt.Errorf("Error creating DB Instance: %s", err)
		}
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
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDbInstanceRead(d, meta)
}

func resourceAwsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsDbInstanceRetrieve(d, meta)

	if err != nil {
		return err
	}
	if v == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", v.DBName)
	d.Set("username", v.MasterUsername)
	d.Set("engine", v.Engine)
	d.Set("engine_version", v.EngineVersion)
	d.Set("allocated_storage", v.AllocatedStorage)
	d.Set("storage_type", v.StorageType)
	d.Set("instance_class", v.DBInstanceClass)
	d.Set("availability_zone", v.AvailabilityZone)
	d.Set("backup_retention_period", v.BackupRetentionPeriod)
	d.Set("backup_window", v.PreferredBackupWindow)
	d.Set("license_model", v.LicenseModel)
	d.Set("maintenance_window", v.PreferredMaintenanceWindow)
	d.Set("multi_az", v.MultiAZ)
	if v.DBSubnetGroup != nil {
		d.Set("db_subnet_group_name", v.DBSubnetGroup.DBSubnetGroupName)
	}

	if len(v.DBParameterGroups) > 0 {
		d.Set("parameter_group_name", v.DBParameterGroups[0].DBParameterGroupName)
	}

	if v.Endpoint != nil {
		d.Set("port", v.Endpoint.Port)
		d.Set("address", v.Endpoint.Address)

		if v.Endpoint.Address != nil && v.Endpoint.Port != nil {
			d.Set("endpoint",
				fmt.Sprintf("%s:%d", *v.Endpoint.Address, *v.Endpoint.Port))
		}
	}

	d.Set("status", v.DBInstanceStatus)
	d.Set("storage_encrypted", v.StorageEncrypted)

	// list tags for resource
	// set tags
	conn := meta.(*AWSClient).rdsconn
	arn, err := buildRDSARN(d, meta)
	if err != nil {
		name := "<empty>"
		if v.DBName != nil && *v.DBName != "" {
			name = *v.DBName
		}
		log.Printf("[DEBUG] Error building ARN for DB Instance, not setting Tags for DB %s", name)
	} else {
		resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: aws.String(arn),
		})

		if err != nil {
			log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
		}

		var dt []*rds.Tag
		if len(resp.TagList) > 0 {
			dt = resp.TagList
		}
		d.Set("tags", tagsToMapRDS(dt))
	}

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

	return nil
}

func resourceAwsDbInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	finalSnapshot := d.Get("final_snapshot_identifier").(string)
	if finalSnapshot == "" {
		opts.SkipFinalSnapshot = aws.Bool(true)
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

func resourceAwsDbInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	d.Partial(true)

	req := &rds.ModifyDBInstanceInput{
		ApplyImmediately:     aws.Bool(d.Get("apply_immediately").(bool)),
		DBInstanceIdentifier: aws.String(d.Id()),
	}
	d.SetPartial("apply_immediately")

	requestUpdate := false
	if d.HasChange("allocated_storage") {
		d.SetPartial("allocated_storage")
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
		req.Iops = aws.Int64(int64(d.Get("iops").(int)))
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
			req.VpcSecurityGroupIds = s
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

	if arn, err := buildRDSARN(d, meta); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}
	d.Partial(false)

	return resourceAwsDbInstanceRead(d, meta)
}

func resourceAwsDbInstanceRetrieve(
	d *schema.ResourceData, meta interface{}) (*rds.DBInstance, error) {
	conn := meta.(*AWSClient).rdsconn

	opts := rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] DB Instance describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)
	if err != nil {
		dbinstanceerr, ok := err.(awserr.Error)
		if ok && dbinstanceerr.Code() == "DBInstanceNotFound" {
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

	return resp.DBInstances[0], nil
}

func resourceAwsDbInstanceStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsDbInstanceRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving DB Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.DBInstanceStatus != nil {
			log.Printf("[DEBUG] DB Instance status for instance %s: %s", d.Id(), *v.DBInstanceStatus)
		}

		return v, *v.DBInstanceStatus, nil
	}
}

func buildRDSARN(d *schema.ResourceData, meta interface{}) (string, error) {
	iamconn := meta.(*AWSClient).iamconn
	region := meta.(*AWSClient).region
	// An zero value GetUserInput{} defers to the currently logged in user
	resp, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}
	userARN := *resp.User.Arn
	accountID := strings.Split(userARN, ":")[4]
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:db:%s", region, accountID, d.Id())
	return arn, nil
}

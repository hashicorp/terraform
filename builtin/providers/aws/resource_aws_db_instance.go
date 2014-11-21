package aws

/*
import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func resourceAwsDbInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbInstanceCreate,
		Read:   resourceAwsDbInstanceRead,
		Delete: resourceAwsDbInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeInt,
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

			"allocated_storage": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
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
			// tot hier
			"health_check_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"load_balancers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"vpc_zone_identifier": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsDbInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	opts := rds.CreateDBInstance{}

	opts.AllocatedStorage = d.Get("allocated_storage").(int)
	opts.SetAllocatedStorage = true

	if attr, ok := d.GetOk("instance_class"); ok {
		opts.DBInstanceClass = attr.(string)
	}

	if attr, ok := d.GetOk("backup_retention_period"); ok {
		opts.BackupRetentionPeriod = attr.(int)
		opts.SetBackupRetentionPeriod = true
	}

	if attr, ok := d.GetOk("iops"); ok {
		opts.Iops = attr.(int)
		opts.SetIops = true
	}

	if attr, ok := d.GetOk("port"); ok {
		opts.Port = attr.(int)
		opts.SetPort = true
	}

	if attr, ok := d.GetOk("availability_zone"); ok {
		opts.AvailabilityZone = attr.(string)
	}

	if attr, ok := d.GetOk("maintenance_window"); ok {
		opts.PreferredMaintenanceWindow = attr.(string)
	}

	if attr, ok := d.GetOk("backup_window"); ok {
		opts.PreferredBackupWindow = attr.(string)
	}

	if attr, ok := d.GetOk("multi_az"); ok {
		opts.MultiAZ = attr.(bool)
	}

	if attr, ok := d.GetOk("publicly_accessible"); ok {
		opts.PubliclyAccessible = attr.(bool)
	}

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		opts.DBSubnetGroupName = attr.(string)
	}

	if attr, ok := d.GetOk("parameter_group_name"); ok {
		opts.DBParameterGroupName = attr.(string)
	}

	if d.Get("vpc_security_group_ids.#").(int) > 0 {
		opts.VpcSecurityGroupIds = d.Get("vpc_security_group_ids").([]string)
	}

	if d.Get("security_group_names.#").(int) > 0 {
		opts.DBSecurityGroupNames = d.Get("security_group_names").([]string)
	}

	opts.DBInstanceIdentifier = d.Get("identifier").(string)
	opts.DBName = d.Get("name").(string)
	opts.MasterUsername = d.Get("username").(string)
	opts.MasterUserPassword = d.Get("password").(string)
	opts.EngineVersion = d.Get("engine_version").(string)
	opts.Engine = d.Get("engine").(string)

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
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d.Id(), conn),
		Timeout:    20 * time.Minute,
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
	conn := meta.(*AWSClient).rdsconn

	s.Attributes["address"] = v.Address
	s.Attributes["allocated_storage"] = strconv.Itoa(v.AllocatedStorage)
	s.Attributes["availability_zone"] = v.AvailabilityZone
	s.Attributes["backup_retention_period"] = strconv.Itoa(v.BackupRetentionPeriod)
	s.Attributes["backup_window"] = v.PreferredBackupWindow
	s.Attributes["endpoint"] = fmt.Sprintf("%s:%s", s.Attributes["address"], strconv.Itoa(v.Port))
	s.Attributes["engine"] = v.Engine
	s.Attributes["engine_version"] = v.EngineVersion
	s.Attributes["instance_class"] = v.DBInstanceClass
	s.Attributes["maintenance_window"] = v.PreferredMaintenanceWindow
	s.Attributes["multi_az"] = strconv.FormatBool(v.MultiAZ)
	s.Attributes["name"] = v.DBName
	s.Attributes["port"] = strconv.Itoa(v.Port)
	s.Attributes["status"] = v.DBInstanceStatus
	s.Attributes["username"] = v.MasterUsername
	s.Attributes["db_subnet_group_name"] = v.DBSubnetGroup.Name
	s.Attributes["parameter_group_name"] = v.DBParameterGroupName

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	if len(v.DBSecurityGroupNames) > 0 && v.DBSecurityGroupNames[0] != "" {
		toFlatten["security_group_names"] = v.DBSecurityGroupNames
	}
	if len(v.VpcSecurityGroupIds) > 0 && v.VpcSecurityGroupIds[0] != "" {
		toFlatten["vpc_security_group_ids"] = v.VpcSecurityGroupIds
	}
	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

func resourceAwsDbInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstance{DBInstanceIdentifier: d.Id()}

	if d.Get("skip_final_snapshot").(string) == "true" {
		opts.SkipFinalSnapshot = true
	} else {
		opts.FinalDBSnapshotIdentifier = s.Attributes["final_snapshot_identifier"]
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
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(s.ID, conn),
		Timeout:    20 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func resource_aws_db_instance_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	v, err := resource_aws_db_instance_retrieve(s.ID, conn)

	if err != nil {
		return s, err
	}
	if v == nil {
		s.ID = ""
		return s, nil
	}

	return resource_aws_db_instance_update_state(s, v)
}

func resource_aws_db_instance_retrieve(id string, conn *rds.Rds) (*rds.DBInstance, error) {
	opts := rds.DescribeDBInstances{
		DBInstanceIdentifier: id,
	}

	log.Printf("[DEBUG] DB Instance describe configuration: %#v", opts)

	resp, err := conn.DescribeDBInstances(&opts)

	if err != nil {
		if strings.Contains(err.Error(), "DBInstanceNotFound") {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DB Instances: %s", err)
	}

	if len(resp.DBInstances) != 1 ||
		resp.DBInstances[0].DBInstanceIdentifier != id {
		if err != nil {
			return nil, nil
		}
	}

	v := resp.DBInstances[0]

	return &v, nil
}

func resourceAwsDbInstanceStateRefreshFunc(id string, conn *rds.Rds) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resource_aws_db_instance_retrieve(id, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Instance when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		return v, v.DBInstanceStatus, nil
	}
}
*/

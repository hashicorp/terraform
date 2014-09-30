package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func resource_aws_db_instance_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error
	var attr string

	opts := rds.CreateDBInstance{}

	if attr = rs.Attributes["allocated_storage"]; attr != "" {
		opts.AllocatedStorage, err = strconv.Atoi(attr)
		opts.SetAllocatedStorage = true
	}

	if attr = rs.Attributes["backup_retention_period"]; attr != "" {
		opts.BackupRetentionPeriod, err = strconv.Atoi(attr)
		opts.SetBackupRetentionPeriod = true
	}

	if attr = rs.Attributes["iops"]; attr != "" {
		opts.Iops, err = strconv.Atoi(attr)
		opts.SetIops = true
	}

	if attr = rs.Attributes["port"]; attr != "" {
		opts.Port, err = strconv.Atoi(attr)
		opts.SetPort = true
	}

	if attr = rs.Attributes["availability_zone"]; attr != "" {
		opts.AvailabilityZone = attr
	}

	if attr = rs.Attributes["instance_class"]; attr != "" {
		opts.DBInstanceClass = attr
	}

	if attr = rs.Attributes["maintenance_window"]; attr != "" {
		opts.PreferredMaintenanceWindow = attr
	}

	if attr = rs.Attributes["backup_window"]; attr != "" {
		opts.PreferredBackupWindow = attr
	}

	if attr = rs.Attributes["multi_az"]; attr == "true" {
		opts.MultiAZ = true
	}

	if attr = rs.Attributes["publicly_accessible"]; attr == "true" {
		opts.PubliclyAccessible = true
	}

	if attr = rs.Attributes["subnet_group_name"]; attr != "" {
		opts.DBSubnetGroupName = attr
	}

	if err != nil {
		return nil, fmt.Errorf("Error parsing configuration: %s", err)
	}

	if _, ok := rs.Attributes["vpc_security_group_ids.#"]; ok {
		opts.VpcSecurityGroupIds = expandStringList(flatmap.Expand(
			rs.Attributes, "vpc_security_group_ids").([]interface{}))
	}

	if _, ok := rs.Attributes["security_group_names.#"]; ok {
		opts.DBSecurityGroupNames = expandStringList(flatmap.Expand(
			rs.Attributes, "security_group_names").([]interface{}))
	}

	opts.DBInstanceIdentifier = rs.Attributes["identifier"]
	opts.DBName = rs.Attributes["name"]
	opts.MasterUsername = rs.Attributes["username"]
	opts.MasterUserPassword = rs.Attributes["password"]
	opts.EngineVersion = rs.Attributes["engine_version"]
	opts.Engine = rs.Attributes["engine"]

	// Don't keep the password around in the state
	delete(rs.Attributes, "password")

	log.Printf("[DEBUG] DB Instance create configuration: %#v", opts)
	_, err = conn.CreateDBInstance(&opts)
	if err != nil {
		return nil, fmt.Errorf("Error creating DB Instance: %s", err)
	}

	rs.ID = rs.Attributes["identifier"]

	log.Printf("[INFO] DB Instance ID: %s", rs.ID)

	log.Println(
		"[INFO] Waiting for DB Instance to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     "available",
		Refresh:    DBInstanceStateRefreshFunc(rs.ID, conn),
		Timeout:    10 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return rs, err
	}

	v, err := resource_aws_db_instance_retrieve(rs.ID, conn)
	if err != nil {
		return rs, err
	}

	return resource_aws_db_instance_update_state(rs, v)
}

func resource_aws_db_instance_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	panic("Cannot update DB")
}

func resource_aws_db_instance_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	log.Printf("[DEBUG] DB Instance destroy: %v", s.ID)

	opts := rds.DeleteDBInstance{DBInstanceIdentifier: s.ID}

	if s.Attributes["skip_final_snapshot"] == "true" {
		opts.SkipFinalSnapshot = true
	} else {
		opts.FinalDBSnapshotIdentifier = s.Attributes["final_snapshot_identifier"]
	}

	log.Printf("[DEBUG] DB Instance destroy configuration: %v", opts)
	_, err := conn.DeleteDBInstance(&opts)

	log.Println(
		"[INFO] Waiting for DB Instance to be destroyed")

	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating", "backing-up",
			"modifying", "deleting", "available"},
		Target:     "",
		Refresh:    DBInstanceStateRefreshFunc(s.ID, conn),
		Timeout:    10 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
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

func resource_aws_db_instance_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"allocated_storage":         diff.AttrTypeCreate,
			"availability_zone":         diff.AttrTypeCreate,
			"backup_retention_period":   diff.AttrTypeCreate,
			"backup_window":             diff.AttrTypeCreate,
			"engine":                    diff.AttrTypeCreate,
			"engine_version":            diff.AttrTypeCreate,
			"identifier":                diff.AttrTypeCreate,
			"instance_class":            diff.AttrTypeCreate,
			"iops":                      diff.AttrTypeCreate,
			"maintenance_window":        diff.AttrTypeCreate,
			"multi_az":                  diff.AttrTypeCreate,
			"name":                      diff.AttrTypeCreate,
			"password":                  diff.AttrTypeUpdate,
			"port":                      diff.AttrTypeCreate,
			"publicly_accessible":       diff.AttrTypeCreate,
			"username":                  diff.AttrTypeCreate,
			"vpc_security_group_ids":    diff.AttrTypeCreate,
			"security_group_names":      diff.AttrTypeCreate,
			"subnet_group_name":         diff.AttrTypeCreate,
			"skip_final_snapshot":       diff.AttrTypeUpdate,
			"final_snapshot_identifier": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"address",
			"availability_zone",
			"backup_retention_period",
			"backup_window",
			"engine_version",
			"maintenance_window",
			"endpoint",
			"status",
			"multi_az",
			"port",
			"address",
			"password",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_db_instance_update_state(
	s *terraform.InstanceState,
	v *rds.DBInstance) (*terraform.InstanceState, error) {

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
	s.Attributes["subnet_group_name"] = v.DBSubnetGroup.Name

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

func resource_aws_db_instance_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"allocated_storage",
			"engine",
			"engine_version",
			"identifier",
			"instance_class",
			"name",
			"password",
			"username",
		},
		Optional: []string{
			"availability_zone",
			"backup_retention_period",
			"backup_window",
			"iops",
			"maintenance_window",
			"multi_az",
			"port",
			"publicly_accessible",
			"vpc_security_group_ids.*",
			"skip_final_snapshot",
			"security_group_names.*",
			"subnet_group_name",
		},
	}
}

func DBInstanceStateRefreshFunc(id string, conn *rds.Rds) resource.StateRefreshFunc {
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

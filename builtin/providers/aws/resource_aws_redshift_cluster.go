package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRedshiftCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftClusterCreate,
		Read:   resourceAwsRedshiftClusterRead,
		Update: resourceAwsRedshiftClusterUpdate,
		Delete: resourceAwsRedshiftClusterDelete,

		Schema: map[string]*schema.Schema{
			"database_name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateRedshiftClusterDbName,
			},

			"cluster_identifier": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateRedshiftClusterIdentifier,
			},
			"cluster_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"node_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"master_username": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRedshiftClusterMasterUsername,
			},

			"master_password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"cluster_security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"cluster_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"preferred_maintenance_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
			},

			"cluster_parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"automated_snapshot_retention_period": &schema.Schema{
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

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5439,
			},

			"cluster_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1.0",
			},

			"allow_version_upgrade": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"number_of_nodes": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"publicly_accessible": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"elastic_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"final_snapshot_identifier": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRedshiftClusterFinalSnapshotIdentifier,
			},

			"skip_final_snapshot": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cluster_public_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cluster_revision_number": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsRedshiftClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	log.Printf("[INFO] Building Redshift Cluster Options")
	createOpts := &redshift.CreateClusterInput{
		ClusterIdentifier:                aws.String(d.Get("cluster_identifier").(string)),
		Port:                             aws.Int64(int64(d.Get("port").(int))),
		MasterUserPassword:               aws.String(d.Get("master_password").(string)),
		MasterUsername:                   aws.String(d.Get("master_username").(string)),
		ClusterVersion:                   aws.String(d.Get("cluster_version").(string)),
		NodeType:                         aws.String(d.Get("node_type").(string)),
		DBName:                           aws.String(d.Get("database_name").(string)),
		AllowVersionUpgrade:              aws.Bool(d.Get("allow_version_upgrade").(bool)),
		PubliclyAccessible:               aws.Bool(d.Get("publicly_accessible").(bool)),
		AutomatedSnapshotRetentionPeriod: aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int))),
	}

	if v := d.Get("number_of_nodes").(int); v > 1 {
		createOpts.ClusterType = aws.String("multi-node")
		createOpts.NumberOfNodes = aws.Int64(int64(d.Get("number_of_nodes").(int)))
	} else {
		createOpts.ClusterType = aws.String("single-node")
	}

	if v := d.Get("cluster_security_groups").(*schema.Set); v.Len() > 0 {
		createOpts.ClusterSecurityGroups = expandStringList(v.List())
	}

	if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
		createOpts.VpcSecurityGroupIds = expandStringList(v.List())
	}

	if v, ok := d.GetOk("cluster_subnet_group_name"); ok {
		createOpts.ClusterSubnetGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		createOpts.AvailabilityZone = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_maintenance_window"); ok {
		createOpts.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cluster_parameter_group_name"); ok {
		createOpts.ClusterParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("encrypted"); ok {
		createOpts.Encrypted = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("elastic_ip"); ok {
		createOpts.ElasticIp = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Redshift Cluster create options: %s", createOpts)
	resp, err := conn.CreateCluster(createOpts)
	if err != nil {
		log.Printf("[ERROR] Error creating Redshift Cluster: %s", err)
		return err
	}

	log.Printf("[DEBUG]: Cluster create response: %s", resp)
	d.SetId(*resp.Cluster.ClusterIdentifier)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     []string{"available"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for Redshift Cluster state to be \"available\": %s", err)
	}

	return resourceAwsRedshiftClusterRead(d, meta)
}

func resourceAwsRedshiftClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	log.Printf("[INFO] Reading Redshift Cluster Information: %s", d.Id())
	resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
		ClusterIdentifier: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if "ClusterNotFound" == awsErr.Code() {
				d.SetId("")
				log.Printf("[DEBUG] Redshift Cluster (%s) not found", d.Id())
				return nil
			}
		}
		log.Printf("[DEBUG] Error describing Redshift Cluster (%s)", d.Id())
		return err
	}

	var rsc *redshift.Cluster
	for _, c := range resp.Clusters {
		if *c.ClusterIdentifier == d.Id() {
			rsc = c
		}
	}

	if rsc == nil {
		log.Printf("[WARN] Redshift Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("database_name", rsc.DBName)
	d.Set("cluster_subnet_group_name", rsc.ClusterSubnetGroupName)
	d.Set("availability_zone", rsc.AvailabilityZone)
	d.Set("encrypted", rsc.Encrypted)
	d.Set("automated_snapshot_retention_period", rsc.AutomatedSnapshotRetentionPeriod)
	d.Set("preferred_maintenance_window", rsc.PreferredMaintenanceWindow)
	if rsc.Endpoint != nil && rsc.Endpoint.Address != nil {
		endpoint := *rsc.Endpoint.Address
		if rsc.Endpoint.Port != nil {
			endpoint = fmt.Sprintf("%s:%d", endpoint, *rsc.Endpoint.Port)
		}
		d.Set("endpoint", endpoint)
	}
	d.Set("cluster_parameter_group_name", rsc.ClusterParameterGroups[0].ParameterGroupName)
	if len(rsc.ClusterNodes) > 1 {
		d.Set("cluster_type", "multi-node")
	} else {
		d.Set("cluster_type", "single-node")
	}

	var vpcg []string
	for _, g := range rsc.VpcSecurityGroups {
		vpcg = append(vpcg, *g.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("[DEBUG] Error saving VPC Security Group IDs to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	var csg []string
	for _, g := range rsc.ClusterSecurityGroups {
		csg = append(csg, *g.ClusterSecurityGroupName)
	}
	if err := d.Set("cluster_security_groups", csg); err != nil {
		return fmt.Errorf("[DEBUG] Error saving Cluster Security Group Names to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	d.Set("cluster_public_key", rsc.ClusterPublicKey)
	d.Set("cluster_revision_number", rsc.ClusterRevisionNumber)

	return nil
}

func resourceAwsRedshiftClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	log.Printf("[INFO] Building Redshift Modify Cluster Options")
	req := &redshift.ModifyClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("cluster_type") {
		req.ClusterType = aws.String(d.Get("cluster_type").(string))
	}

	if d.HasChange("node_type") {
		req.NodeType = aws.String(d.Get("node_type").(string))
	}

	if d.HasChange("number_of_nodes") {
		if v := d.Get("number_of_nodes").(int); v > 1 {
			req.ClusterType = aws.String("multi-node")
			req.NumberOfNodes = aws.Int64(int64(d.Get("number_of_nodes").(int)))
		} else {
			req.ClusterType = aws.String("single-node")
		}
		req.NodeType = aws.String(d.Get("node_type").(string))
	}

	if d.HasChange("cluster_security_groups") {
		req.ClusterSecurityGroups = expandStringList(d.Get("cluster_security_groups").(*schema.Set).List())
	}

	if d.HasChange("vpc_security_group_ips") {
		req.VpcSecurityGroupIds = expandStringList(d.Get("vpc_security_group_ips").(*schema.Set).List())
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
	}

	if d.HasChange("cluster_parameter_group_name") {
		req.ClusterParameterGroupName = aws.String(d.Get("cluster_parameter_group_name").(string))
	}

	if d.HasChange("automated_snapshot_retention_period") {
		req.AutomatedSnapshotRetentionPeriod = aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int)))
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
	}

	if d.HasChange("cluster_version") {
		req.ClusterVersion = aws.String(d.Get("cluster_version").(string))
	}

	if d.HasChange("allow_version_upgrade") {
		req.AllowVersionUpgrade = aws.Bool(d.Get("allow_version_upgrade").(bool))
	}

	if d.HasChange("publicly_accessible") {
		req.PubliclyAccessible = aws.Bool(d.Get("publicly_accessible").(bool))
	}

	log.Printf("[INFO] Modifying Redshift Cluster: %s", d.Id())
	log.Printf("[DEBUG] Redshift Cluster Modify options: %s", req)
	_, err := conn.ModifyCluster(req)
	if err != nil {
		return fmt.Errorf("[WARN] Error modifying Redshift Cluster (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "deleting", "rebooting", "resizing", "renaming", "modifying"},
		Target:     []string{"available"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error Modifying Redshift Cluster (%s): %s", d.Id(), err)
	}

	return resourceAwsRedshiftClusterRead(d, meta)
}

func resourceAwsRedshiftClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	log.Printf("[DEBUG] Destroying Redshift Cluster (%s)", d.Id())

	deleteOpts := redshift.DeleteClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalClusterSnapshot = aws.Bool(skipFinalSnapshot)

	if !skipFinalSnapshot {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalClusterSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("Redshift Cluster Instance FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] Redshift Cluster delete options: %s", deleteOpts)
	_, err := conn.DeleteCluster(&deleteOpts)
	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting Redshift Cluster (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "creating", "deleting", "rebooting", "resizing", "renaming"},
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 5 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting Redshift Cluster (%s): %s", d.Id(), err)
	}

	log.Printf("[INFO] Redshift Cluster %s successfully deleted", d.Id())

	return nil
}

func resourceAwsRedshiftClusterStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).redshiftconn

		log.Printf("[INFO] Reading Redshift Cluster Information: %s", d.Id())
		resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
			ClusterIdentifier: aws.String(d.Id()),
		})

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if "ClusterNotFound" == awsErr.Code() {
					return 42, "destroyed", nil
				}
			}
			log.Printf("[WARN] Error on retrieving Redshift Cluster (%s) when waiting: %s", d.Id(), err)
			return nil, "", err
		}

		var rsc *redshift.Cluster

		for _, c := range resp.Clusters {
			if *c.ClusterIdentifier == d.Id() {
				rsc = c
			}
		}

		if rsc == nil {
			return 42, "destroyed", nil
		}

		if rsc.ClusterStatus != nil {
			log.Printf("[DEBUG] Redshift Cluster status (%s): %s", d.Id(), *rsc.ClusterStatus)
		}

		return rsc, *rsc.ClusterStatus, nil
	}
}

func validateRedshiftClusterIdentifier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	return
}

func validateRedshiftClusterDbName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase letters characters allowed in %q", k))
	}
	if len(value) > 64 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 64 characters: %q", k, value))
	}
	if value == "" {
		errors = append(errors, fmt.Errorf(
			"%q cannot be an empty string", k))
	}

	return
}

func validateRedshiftClusterFinalSnapshotIdentifier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q cannot end in a hyphen", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf("%q cannot be more than 255 characters", k))
	}
	return
}

func validateRedshiftClusterMasterUsername(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^\w+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters in %q", k))
	}
	if !regexp.MustCompile(`^[A-Za-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be more than 128 characters", k))
	}
	return
}

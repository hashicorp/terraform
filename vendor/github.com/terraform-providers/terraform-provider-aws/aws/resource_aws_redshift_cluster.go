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
		Importer: &schema.ResourceImporter{
			State: resourceAwsRedshiftClusterImport,
		},

		Schema: map[string]*schema.Schema{
			"database_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateRedshiftClusterDbName,
			},

			"cluster_identifier": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateRedshiftClusterIdentifier,
			},
			"cluster_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"node_type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"master_username": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRedshiftClusterMasterUsername,
			},

			"master_password": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validateRedshiftClusterMasterPassword,
			},

			"cluster_security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"cluster_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

			"cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"automated_snapshot_retention_period": {
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

			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5439,
			},

			"cluster_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1.0",
			},

			"allow_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"number_of_nodes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"enhanced_vpc_routing": {
				Type:     schema.TypeBool,
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

			"elastic_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"final_snapshot_identifier": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRedshiftClusterFinalSnapshotIdentifier,
			},

			"skip_final_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cluster_public_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cluster_revision_number": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"iam_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"enable_logging": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"bucket_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"s3_key_prefix": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"snapshot_cluster_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"owner_account": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRedshiftClusterImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither skip_final_snapshot nor final_snapshot_identifier can be fetched
	// from any API call, so we need to default skip_final_snapshot to true so
	// that final_snapshot_identifier is not required
	d.Set("skip_final_snapshot", true)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsRedshiftClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	tags := tagsFromMapRedshift(d.Get("tags").(map[string]interface{}))

	if v, ok := d.GetOk("snapshot_identifier"); ok {
		restoreOpts := &redshift.RestoreFromClusterSnapshotInput{
			ClusterIdentifier:                aws.String(d.Get("cluster_identifier").(string)),
			SnapshotIdentifier:               aws.String(v.(string)),
			Port:                             aws.Int64(int64(d.Get("port").(int))),
			AllowVersionUpgrade:              aws.Bool(d.Get("allow_version_upgrade").(bool)),
			NodeType:                         aws.String(d.Get("node_type").(string)),
			PubliclyAccessible:               aws.Bool(d.Get("publicly_accessible").(bool)),
			AutomatedSnapshotRetentionPeriod: aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int))),
		}

		if v, ok := d.GetOk("owner_account"); ok {
			restoreOpts.OwnerAccount = aws.String(v.(string))
		}

		if v, ok := d.GetOk("snapshot_cluster_identifier"); ok {
			restoreOpts.SnapshotClusterIdentifier = aws.String(v.(string))
		}

		if v, ok := d.GetOk("availability_zone"); ok {
			restoreOpts.AvailabilityZone = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cluster_subnet_group_name"); ok {
			restoreOpts.ClusterSubnetGroupName = aws.String(v.(string))
		}

		if v, ok := d.GetOk("cluster_parameter_group_name"); ok {
			restoreOpts.ClusterParameterGroupName = aws.String(v.(string))
		}

		if v := d.Get("cluster_security_groups").(*schema.Set); v.Len() > 0 {
			restoreOpts.ClusterSecurityGroups = expandStringList(v.List())
		}

		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			restoreOpts.VpcSecurityGroupIds = expandStringList(v.List())
		}

		if v, ok := d.GetOk("preferred_maintenance_window"); ok {
			restoreOpts.PreferredMaintenanceWindow = aws.String(v.(string))
		}

		if v, ok := d.GetOk("kms_key_id"); ok {
			restoreOpts.KmsKeyId = aws.String(v.(string))
		}

		if v, ok := d.GetOk("elastic_ip"); ok {
			restoreOpts.ElasticIp = aws.String(v.(string))
		}

		if v, ok := d.GetOk("enhanced_vpc_routing"); ok {
			restoreOpts.EnhancedVpcRouting = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("iam_roles"); ok {
			restoreOpts.IamRoles = expandStringList(v.(*schema.Set).List())
		}

		log.Printf("[DEBUG] Redshift Cluster restore cluster options: %s", restoreOpts)

		resp, err := conn.RestoreFromClusterSnapshot(restoreOpts)
		if err != nil {
			log.Printf("[ERROR] Error Restoring Redshift Cluster from Snapshot: %s", err)
			return err
		}

		d.SetId(*resp.Cluster.ClusterIdentifier)

	} else {
		if _, ok := d.GetOk("master_password"); !ok {
			return fmt.Errorf(`provider.aws: aws_redshift_cluster: %s: "master_password": required field is not set`, d.Get("cluster_identifier").(string))
		}

		if _, ok := d.GetOk("master_username"); !ok {
			return fmt.Errorf(`provider.aws: aws_redshift_cluster: %s: "master_username": required field is not set`, d.Get("cluster_identifier").(string))
		}

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
			Tags: tags,
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

		if v, ok := d.GetOk("enhanced_vpc_routing"); ok {
			createOpts.EnhancedVpcRouting = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("kms_key_id"); ok {
			createOpts.KmsKeyId = aws.String(v.(string))
		}

		if v, ok := d.GetOk("elastic_ip"); ok {
			createOpts.ElasticIp = aws.String(v.(string))
		}

		if v, ok := d.GetOk("iam_roles"); ok {
			createOpts.IamRoles = expandStringList(v.(*schema.Set).List())
		}

		log.Printf("[DEBUG] Redshift Cluster create options: %s", createOpts)
		resp, err := conn.CreateCluster(createOpts)
		if err != nil {
			log.Printf("[ERROR] Error creating Redshift Cluster: %s", err)
			return err
		}

		log.Printf("[DEBUG]: Cluster create response: %s", resp)
		d.SetId(*resp.Cluster.ClusterIdentifier)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying", "restoring"},
		Target:     []string{"available"},
		Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d, meta),
		Timeout:    75 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for Redshift Cluster state to be \"available\": %s", err)
	}

	if _, ok := d.GetOk("enable_logging"); ok {

		loggingErr := enableRedshiftClusterLogging(d, conn)
		if loggingErr != nil {
			log.Printf("[ERROR] Error Enabling Logging on Redshift Cluster: %s", err)
			return loggingErr
		}

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

	log.Printf("[INFO] Reading Redshift Cluster Logging Status: %s", d.Id())
	loggingStatus, loggingErr := conn.DescribeLoggingStatus(&redshift.DescribeLoggingStatusInput{
		ClusterIdentifier: aws.String(d.Id()),
	})

	if loggingErr != nil {
		return loggingErr
	}

	d.Set("master_username", rsc.MasterUsername)
	d.Set("node_type", rsc.NodeType)
	d.Set("allow_version_upgrade", rsc.AllowVersionUpgrade)
	d.Set("database_name", rsc.DBName)
	d.Set("cluster_identifier", rsc.ClusterIdentifier)
	d.Set("cluster_version", rsc.ClusterVersion)

	d.Set("cluster_subnet_group_name", rsc.ClusterSubnetGroupName)
	d.Set("availability_zone", rsc.AvailabilityZone)
	d.Set("encrypted", rsc.Encrypted)
	d.Set("enhanced_vpc_routing", rsc.EnhancedVpcRouting)
	d.Set("kms_key_id", rsc.KmsKeyId)
	d.Set("automated_snapshot_retention_period", rsc.AutomatedSnapshotRetentionPeriod)
	d.Set("preferred_maintenance_window", rsc.PreferredMaintenanceWindow)
	if rsc.Endpoint != nil && rsc.Endpoint.Address != nil {
		endpoint := *rsc.Endpoint.Address
		if rsc.Endpoint.Port != nil {
			endpoint = fmt.Sprintf("%s:%d", endpoint, *rsc.Endpoint.Port)
		}
		d.Set("port", rsc.Endpoint.Port)
		d.Set("endpoint", endpoint)
	}
	d.Set("cluster_parameter_group_name", rsc.ClusterParameterGroups[0].ParameterGroupName)
	if len(rsc.ClusterNodes) > 1 {
		d.Set("cluster_type", "multi-node")
	} else {
		d.Set("cluster_type", "single-node")
	}
	d.Set("number_of_nodes", rsc.NumberOfNodes)
	d.Set("publicly_accessible", rsc.PubliclyAccessible)

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

	var iamRoles []string
	for _, i := range rsc.IamRoles {
		iamRoles = append(iamRoles, *i.IamRoleArn)
	}
	if err := d.Set("iam_roles", iamRoles); err != nil {
		return fmt.Errorf("[DEBUG] Error saving IAM Roles to state for Redshift Cluster (%s): %s", d.Id(), err)
	}

	d.Set("cluster_public_key", rsc.ClusterPublicKey)
	d.Set("cluster_revision_number", rsc.ClusterRevisionNumber)
	d.Set("tags", tagsToMapRedshift(rsc.Tags))

	d.Set("bucket_name", loggingStatus.BucketName)
	d.Set("enable_logging", loggingStatus.LoggingEnabled)
	d.Set("s3_key_prefix", loggingStatus.S3KeyPrefix)

	return nil
}

func resourceAwsRedshiftClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	d.Partial(true)

	arn, tagErr := buildRedshiftARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if tagErr != nil {
		return fmt.Errorf("Error building ARN for Redshift Cluster, not updating Tags for cluster %s", d.Id())
	} else {
		if tagErr := setTagsRedshift(conn, d, arn); tagErr != nil {
			return tagErr
		} else {
			d.SetPartial("tags")
		}
	}

	requestUpdate := false
	log.Printf("[INFO] Building Redshift Modify Cluster Options")
	req := &redshift.ModifyClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("cluster_type") {
		req.ClusterType = aws.String(d.Get("cluster_type").(string))
		requestUpdate = true
	}

	if d.HasChange("node_type") {
		req.NodeType = aws.String(d.Get("node_type").(string))
		requestUpdate = true
	}

	if d.HasChange("number_of_nodes") {
		if v := d.Get("number_of_nodes").(int); v > 1 {
			req.ClusterType = aws.String("multi-node")
			req.NumberOfNodes = aws.Int64(int64(d.Get("number_of_nodes").(int)))
		} else {
			req.ClusterType = aws.String("single-node")
		}

		req.NodeType = aws.String(d.Get("node_type").(string))
		requestUpdate = true
	}

	if d.HasChange("cluster_security_groups") {
		req.ClusterSecurityGroups = expandStringList(d.Get("cluster_security_groups").(*schema.Set).List())
		requestUpdate = true
	}

	if d.HasChange("vpc_security_group_ids") {
		req.VpcSecurityGroupIds = expandStringList(d.Get("vpc_security_group_ids").(*schema.Set).List())
		requestUpdate = true
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
		requestUpdate = true
	}

	if d.HasChange("cluster_parameter_group_name") {
		req.ClusterParameterGroupName = aws.String(d.Get("cluster_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("automated_snapshot_retention_period") {
		req.AutomatedSnapshotRetentionPeriod = aws.Int64(int64(d.Get("automated_snapshot_retention_period").(int)))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("cluster_version") {
		req.ClusterVersion = aws.String(d.Get("cluster_version").(string))
		requestUpdate = true
	}

	if d.HasChange("allow_version_upgrade") {
		req.AllowVersionUpgrade = aws.Bool(d.Get("allow_version_upgrade").(bool))
		requestUpdate = true
	}

	if d.HasChange("publicly_accessible") {
		req.PubliclyAccessible = aws.Bool(d.Get("publicly_accessible").(bool))
		requestUpdate = true
	}

	if d.HasChange("enhanced_vpc_routing") {
		req.EnhancedVpcRouting = aws.Bool(d.Get("enhanced_vpc_routing").(bool))
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[INFO] Modifying Redshift Cluster: %s", d.Id())
		log.Printf("[DEBUG] Redshift Cluster Modify options: %s", req)
		_, err := conn.ModifyCluster(req)
		if err != nil {
			return fmt.Errorf("[WARN] Error modifying Redshift Cluster (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("iam_roles") {
		o, n := d.GetChange("iam_roles")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removeIams := os.Difference(ns).List()
		addIams := ns.Difference(os).List()

		log.Printf("[INFO] Building Redshift Modify Cluster IAM Role Options")
		req := &redshift.ModifyClusterIamRolesInput{
			ClusterIdentifier: aws.String(d.Id()),
			AddIamRoles:       expandStringList(addIams),
			RemoveIamRoles:    expandStringList(removeIams),
		}

		log.Printf("[INFO] Modifying Redshift Cluster IAM Roles: %s", d.Id())
		log.Printf("[DEBUG] Redshift Cluster Modify IAM Role options: %s", req)
		_, err := conn.ModifyClusterIamRoles(req)
		if err != nil {
			return fmt.Errorf("[WARN] Error modifying Redshift Cluster IAM Roles (%s): %s", d.Id(), err)
		}

		d.SetPartial("iam_roles")
	}

	if requestUpdate || d.HasChange("iam_roles") {

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"creating", "deleting", "rebooting", "resizing", "renaming", "modifying"},
			Target:     []string{"available"},
			Refresh:    resourceAwsRedshiftClusterStateRefreshFunc(d, meta),
			Timeout:    40 * time.Minute,
			MinTimeout: 10 * time.Second,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("[WARN] Error Modifying Redshift Cluster (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("enable_logging") || d.HasChange("bucket_name") || d.HasChange("s3_key_prefix") {
		var loggingErr error
		if _, ok := d.GetOk("enable_logging"); ok {

			log.Printf("[INFO] Enabling Logging for Redshift Cluster %q", d.Id())
			loggingErr = enableRedshiftClusterLogging(d, conn)
			if loggingErr != nil {
				return loggingErr
			}
		} else {

			log.Printf("[INFO] Disabling Logging for Redshift Cluster %q", d.Id())
			_, loggingErr = conn.DisableLogging(&redshift.DisableLoggingInput{
				ClusterIdentifier: aws.String(d.Id()),
			})
			if loggingErr != nil {
				return loggingErr
			}
		}

		d.SetPartial("enable_logging")
	}

	d.Partial(false)

	return resourceAwsRedshiftClusterRead(d, meta)
}

func enableRedshiftClusterLogging(d *schema.ResourceData, conn *redshift.Redshift) error {
	if _, ok := d.GetOk("bucket_name"); !ok {
		return fmt.Errorf("bucket_name must be set when enabling logging for Redshift Clusters")
	}

	params := &redshift.EnableLoggingInput{
		ClusterIdentifier: aws.String(d.Id()),
		BucketName:        aws.String(d.Get("bucket_name").(string)),
	}

	if v, ok := d.GetOk("s3_key_prefix"); ok {
		params.S3KeyPrefix = aws.String(v.(string))
	}

	_, loggingErr := conn.EnableLogging(params)
	if loggingErr != nil {
		log.Printf("[ERROR] Error Enabling Logging on Redshift Cluster: %s", loggingErr)
		return loggingErr
	}
	return nil
}

func resourceAwsRedshiftClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	log.Printf("[DEBUG] Destroying Redshift Cluster (%s)", d.Id())

	deleteOpts := redshift.DeleteClusterInput{
		ClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalClusterSnapshot = aws.Bool(skipFinalSnapshot)

	if skipFinalSnapshot == false {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalClusterSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("Redshift Cluster Instance FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] Redshift Cluster delete options: %s", deleteOpts)
	err := resource.Retry(15*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteCluster(&deleteOpts)
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "InvalidClusterState" {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting Redshift Cluster (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "creating", "deleting", "rebooting", "resizing", "renaming", "final-snapshot"},
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
	if !regexp.MustCompile(`^[0-9a-z_$]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters, underscores, and dollar signs are allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-zA-Z_]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter or underscore", k))
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

func validateRedshiftClusterMasterPassword(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^.*[a-z].*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must contain at least one lowercase letter", k))
	}
	if !regexp.MustCompile(`^.*[A-Z].*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must contain at least one uppercase letter", k))
	}
	if !regexp.MustCompile(`^.*[0-9].*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must contain at least one number", k))
	}
	if !regexp.MustCompile(`^[^\@\/'" ]*$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain [/@\"' ]", k))
	}
	if len(value) < 8 {
		errors = append(errors, fmt.Errorf("%q must be at least 8 characters", k))
	}
	return
}

func buildRedshiftARN(identifier, partition, accountid, region string) (string, error) {
	if partition == "" {
		return "", fmt.Errorf("Unable to construct cluster ARN because of missing AWS partition")
	}
	if accountid == "" {
		return "", fmt.Errorf("Unable to construct cluster ARN because of missing AWS Account ID")
	}
	arn := fmt.Sprintf("arn:%s:redshift:%s:%s:cluster:%s", partition, region, accountid, identifier)
	return arn, nil

}

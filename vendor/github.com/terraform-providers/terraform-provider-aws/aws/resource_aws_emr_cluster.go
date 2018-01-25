package aws

import (
	"log"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEMRCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEMRClusterCreate,
		Read:   resourceAwsEMRClusterRead,
		Update: resourceAwsEMRClusterUpdate,
		Delete: resourceAwsEMRClusterDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"release_label": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"master_instance_type": {
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				ForceNew: true,
			},
			"core_instance_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"core_instance_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cluster_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"log_uri": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// EMR uses a proprietary filesystem called EMRFS
					// and both s3n & s3 protocols are mapped to that FS
					// so they're equvivalent in this context (confirmed by AWS support)
					old = strings.Replace(old, "s3n://", "s3://", -1)
					return old == new
				},
			},
			"master_public_dns": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"applications": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"termination_protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"keep_job_flow_alive_when_no_steps": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},
			"ec2_attributes": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"additional_master_security_groups": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"additional_slave_security_groups": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"emr_managed_master_security_group": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"emr_managed_slave_security_group": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"instance_profile": {
							Type:     schema.TypeString,
							Required: true,
						},
						"service_access_security_group": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"instance_group": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bid_price": {
							Type:     schema.TypeString,
							Optional: true,
							Required: false,
						},
						"ebs_config": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"iops": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"size": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateAwsEmrEbsVolumeType,
									},
									"volumes_per_instance": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  1,
									},
								},
							},
						},
						"instance_count": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"instance_role": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsEmrInstanceGroupRole,
						},
						"instance_type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"bootstrap_action": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"path": {
							Type:     schema.TypeString,
							Required: true,
						},
						"args": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"tags": tagsSchema(),
			"configurations": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"service_role": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"security_configuration": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"autoscaling_role": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"visible_to_all_users": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"ebs_root_volume_size": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsEMRClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	log.Printf("[DEBUG] Creating EMR cluster")
	applications := d.Get("applications").(*schema.Set).List()

	keepJobFlowAliveWhenNoSteps := true
	if v, ok := d.GetOk("keep_job_flow_alive_when_no_steps"); ok {
		keepJobFlowAliveWhenNoSteps = v.(bool)
	}

	terminationProtection := false
	if v, ok := d.GetOk("termination_protection"); ok {
		terminationProtection = v.(bool)
	}
	instanceConfig := &emr.JobFlowInstancesConfig{
		KeepJobFlowAliveWhenNoSteps: aws.Bool(keepJobFlowAliveWhenNoSteps),
		TerminationProtected:        aws.Bool(terminationProtection),
	}

	if v, ok := d.GetOk("master_instance_type"); ok {
		instanceConfig.MasterInstanceType = aws.String(v.(string))
		instanceConfig.SlaveInstanceType = aws.String(v.(string))
	}
	if v, ok := d.GetOk("core_instance_type"); ok {
		instanceConfig.SlaveInstanceType = aws.String(v.(string))
	}
	if v, ok := d.GetOk("core_instance_count"); ok {
		instanceConfig.InstanceCount = aws.Int64(int64(v.(int)))
	}

	var instanceProfile string
	if a, ok := d.GetOk("ec2_attributes"); ok {
		ec2Attributes := a.([]interface{})
		attributes := ec2Attributes[0].(map[string]interface{})

		if v, ok := attributes["key_name"]; ok {
			instanceConfig.Ec2KeyName = aws.String(v.(string))
		}
		if v, ok := attributes["subnet_id"]; ok {
			instanceConfig.Ec2SubnetId = aws.String(v.(string))
		}
		if v, ok := attributes["subnet_id"]; ok {
			instanceConfig.Ec2SubnetId = aws.String(v.(string))
		}

		if v, ok := attributes["additional_master_security_groups"]; ok {
			strSlice := strings.Split(v.(string), ",")
			for i, s := range strSlice {
				strSlice[i] = strings.TrimSpace(s)
			}
			instanceConfig.AdditionalMasterSecurityGroups = aws.StringSlice(strSlice)
		}

		if v, ok := attributes["additional_slave_security_groups"]; ok {
			strSlice := strings.Split(v.(string), ",")
			for i, s := range strSlice {
				strSlice[i] = strings.TrimSpace(s)
			}
			instanceConfig.AdditionalSlaveSecurityGroups = aws.StringSlice(strSlice)
		}

		if v, ok := attributes["emr_managed_master_security_group"]; ok {
			instanceConfig.EmrManagedMasterSecurityGroup = aws.String(v.(string))
		}
		if v, ok := attributes["emr_managed_slave_security_group"]; ok {
			instanceConfig.EmrManagedSlaveSecurityGroup = aws.String(v.(string))
		}

		if len(strings.TrimSpace(attributes["instance_profile"].(string))) != 0 {
			instanceProfile = strings.TrimSpace(attributes["instance_profile"].(string))
		}

		if v, ok := attributes["service_access_security_group"]; ok {
			instanceConfig.ServiceAccessSecurityGroup = aws.String(v.(string))
		}
	}
	if v, ok := d.GetOk("instance_group"); ok {
		instanceGroupConfigs := v.(*schema.Set).List()
		instanceConfig.InstanceGroups = expandInstanceGroupConfigs(instanceGroupConfigs)
	}

	emrApps := expandApplications(applications)

	params := &emr.RunJobFlowInput{
		Instances:    instanceConfig,
		Name:         aws.String(d.Get("name").(string)),
		Applications: emrApps,

		ReleaseLabel:      aws.String(d.Get("release_label").(string)),
		ServiceRole:       aws.String(d.Get("service_role").(string)),
		VisibleToAllUsers: aws.Bool(d.Get("visible_to_all_users").(bool)),
	}

	if v, ok := d.GetOk("log_uri"); ok {
		params.LogUri = aws.String(v.(string))
	}
	if v, ok := d.GetOk("autoscaling_role"); ok {
		params.AutoScalingRole = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_configuration"); ok {
		params.SecurityConfiguration = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ebs_root_volume_size"); ok {
		params.EbsRootVolumeSize = aws.Int64(int64(v.(int)))
	}

	if instanceProfile != "" {
		params.JobFlowRole = aws.String(instanceProfile)
	}

	if v, ok := d.GetOk("bootstrap_action"); ok {
		bootstrapActions := v.(*schema.Set).List()
		params.BootstrapActions = expandBootstrapActions(bootstrapActions)
	}
	if v, ok := d.GetOk("tags"); ok {
		tagsIn := v.(map[string]interface{})
		params.Tags = expandTags(tagsIn)
	}
	if v, ok := d.GetOk("configurations"); ok {
		confUrl := v.(string)
		params.Configurations = expandConfigures(confUrl)
	}

	log.Printf("[DEBUG] EMR Cluster create options: %s", params)

	var resp *emr.RunJobFlowOutput
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error
		resp, err = conn.RunJobFlow(params)
		if err != nil {
			if isAWSErr(err, "ValidationException", "Invalid InstanceProfile:") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "AccessDeniedException", "Failed to authorize instance profile") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(*resp.JobFlowId)

	log.Println("[INFO] Waiting for EMR Cluster to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"STARTING", "BOOTSTRAPPING"},
		Target:     []string{"WAITING", "RUNNING"},
		Refresh:    resourceAwsEMRClusterStateRefreshFunc(d, meta),
		Timeout:    75 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for EMR Cluster state to be \"WAITING\" or \"RUNNING\": %s", err)
	}

	return resourceAwsEMRClusterRead(d, meta)
}

func resourceAwsEMRClusterRead(d *schema.ResourceData, meta interface{}) error {
	emrconn := meta.(*AWSClient).emrconn

	req := &emr.DescribeClusterInput{
		ClusterId: aws.String(d.Id()),
	}

	resp, err := emrconn.DescribeCluster(req)
	if err != nil {
		return fmt.Errorf("Error reading EMR cluster: %s", err)
	}

	if resp.Cluster == nil {
		log.Printf("[DEBUG] EMR Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	cluster := resp.Cluster

	if cluster.Status != nil {
		if *cluster.Status.State == "TERMINATED" {
			log.Printf("[DEBUG] EMR Cluster (%s) was TERMINATED already", d.Id())
			d.SetId("")
			return nil
		}

		if *cluster.Status.State == "TERMINATED_WITH_ERRORS" {
			log.Printf("[DEBUG] EMR Cluster (%s) was TERMINATED_WITH_ERRORS already", d.Id())
			d.SetId("")
			return nil
		}

		d.Set("cluster_state", cluster.Status.State)
	}

	instanceGroups, err := fetchAllEMRInstanceGroups(emrconn, d.Id())
	if err == nil {
		coreGroup := findGroup(instanceGroups, "CORE")
		if coreGroup != nil {
			d.Set("core_instance_type", coreGroup.InstanceType)
		}
		if err := d.Set("instance_group", flattenInstanceGroups(instanceGroups)); err != nil {
			log.Printf("[ERR] Error setting EMR instance groups: %s", err)
		}
	}

	d.Set("name", cluster.Name)
	d.Set("service_role", cluster.ServiceRole)
	d.Set("security_configuration", cluster.SecurityConfiguration)
	d.Set("autoscaling_role", cluster.AutoScalingRole)
	d.Set("release_label", cluster.ReleaseLabel)
	d.Set("log_uri", cluster.LogUri)
	d.Set("master_public_dns", cluster.MasterPublicDnsName)
	d.Set("visible_to_all_users", cluster.VisibleToAllUsers)
	d.Set("tags", tagsToMapEMR(cluster.Tags))
	d.Set("ebs_root_volume_size", cluster.EbsRootVolumeSize)

	if err := d.Set("applications", flattenApplications(cluster.Applications)); err != nil {
		log.Printf("[ERR] Error setting EMR Applications for cluster (%s): %s", d.Id(), err)
	}

	// Configurations is a JSON document. It's built with an expand method but a
	// simple string should be returned as JSON
	if err := d.Set("configurations", cluster.Configurations); err != nil {
		log.Printf("[ERR] Error setting EMR configurations for cluster (%s): %s", d.Id(), err)
	}

	if err := d.Set("ec2_attributes", flattenEc2Attributes(cluster.Ec2InstanceAttributes)); err != nil {
		log.Printf("[ERR] Error setting EMR Ec2 Attributes: %s", err)
	}

	respBootstraps, err := emrconn.ListBootstrapActions(&emr.ListBootstrapActionsInput{
		ClusterId: cluster.Id,
	})
	if err != nil {
		log.Printf("[WARN] Error listing bootstrap actions: %s", err)
	}

	if err := d.Set("bootstrap_action", flattenBootstrapArguments(respBootstraps.BootstrapActions)); err != nil {
		log.Printf("[WARN] Error setting Bootstrap Actions: %s", err)
	}

	return nil
}

func resourceAwsEMRClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	d.Partial(true)

	if d.HasChange("core_instance_count") {
		d.SetPartial("core_instance_count")
		log.Printf("[DEBUG] Modify EMR cluster")
		groups, err := fetchAllEMRInstanceGroups(conn, d.Id())
		if err != nil {
			log.Printf("[DEBUG] Error finding all instance groups: %s", err)
			return err
		}

		coreInstanceCount := d.Get("core_instance_count").(int)
		coreGroup := findGroup(groups, "CORE")
		if coreGroup == nil {
			return fmt.Errorf("[ERR] Error finding core group")
		}

		params := &emr.ModifyInstanceGroupsInput{
			InstanceGroups: []*emr.InstanceGroupModifyConfig{
				{
					InstanceGroupId: coreGroup.Id,
					InstanceCount:   aws.Int64(int64(coreInstanceCount) - 1),
				},
			},
		}
		_, errModify := conn.ModifyInstanceGroups(params)
		if errModify != nil {
			log.Printf("[ERROR] %s", errModify)
			return errModify
		}

		log.Printf("[DEBUG] Modify EMR Cluster done...")

		log.Println("[INFO] Waiting for EMR Cluster to be available")

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"STARTING", "BOOTSTRAPPING"},
			Target:     []string{"WAITING", "RUNNING"},
			Refresh:    resourceAwsEMRClusterStateRefreshFunc(d, meta),
			Timeout:    40 * time.Minute,
			MinTimeout: 10 * time.Second,
			Delay:      5 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("[WARN] Error waiting for EMR Cluster state to be \"WAITING\" or \"RUNNING\" after modification: %s", err)
		}
	}

	if d.HasChange("visible_to_all_users") {
		d.SetPartial("visible_to_all_users")
		_, errModify := conn.SetVisibleToAllUsers(&emr.SetVisibleToAllUsersInput{
			JobFlowIds:        []*string{aws.String(d.Id())},
			VisibleToAllUsers: aws.Bool(d.Get("visible_to_all_users").(bool)),
		})
		if errModify != nil {
			log.Printf("[ERROR] %s", errModify)
			return errModify
		}
	}

	if d.HasChange("termination_protection") {
		d.SetPartial("termination_protection")
		_, errModify := conn.SetTerminationProtection(&emr.SetTerminationProtectionInput{
			JobFlowIds:           []*string{aws.String(d.Id())},
			TerminationProtected: aws.Bool(d.Get("termination_protection").(bool)),
		})
		if errModify != nil {
			log.Printf("[ERROR] %s", errModify)
			return errModify
		}
	}

	if err := setTagsEMR(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsEMRClusterRead(d, meta)
}

func resourceAwsEMRClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	req := &emr.TerminateJobFlowsInput{
		JobFlowIds: []*string{
			aws.String(d.Id()),
		},
	}

	_, err := conn.TerminateJobFlows(req)
	if err != nil {
		log.Printf("[ERROR], %s", err)
		return err
	}

	err = resource.Retry(10*time.Minute, func() *resource.RetryError {
		resp, err := conn.ListInstances(&emr.ListInstancesInput{
			ClusterId: aws.String(d.Id()),
		})

		if err != nil {
			return resource.NonRetryableError(err)
		}

		instanceCount := len(resp.Instances)

		if resp == nil || instanceCount == 0 {
			log.Printf("[DEBUG] No instances found for EMR Cluster (%s)", d.Id())
			return nil
		}

		// Collect instance status states, wait for all instances to be terminated
		// before moving on
		var terminated []string
		for j, i := range resp.Instances {
			if i.Status != nil {
				if *i.Status.State == "TERMINATED" {
					terminated = append(terminated, *i.Ec2InstanceId)
				}
			} else {
				log.Printf("[DEBUG] Cluster instance (%d : %s) has no status", j, *i.Ec2InstanceId)
			}
		}
		if len(terminated) == instanceCount {
			log.Printf("[DEBUG] All (%d) EMR Cluster (%s) Instances terminated", instanceCount, d.Id())
			return nil
		}
		return resource.RetryableError(fmt.Errorf("[DEBUG] EMR Cluster (%s) has (%d) Instances remaining, retrying", d.Id(), len(resp.Instances)))
	})

	if err != nil {
		log.Printf("[ERR] Error waiting for EMR Cluster (%s) Instances to drain", d.Id())
	}

	d.SetId("")
	return nil
}

func expandApplications(apps []interface{}) []*emr.Application {
	appOut := make([]*emr.Application, 0, len(apps))

	for _, appName := range expandStringList(apps) {
		app := &emr.Application{
			Name: appName,
		}
		appOut = append(appOut, app)
	}
	return appOut
}

func flattenApplications(apps []*emr.Application) []interface{} {
	appOut := make([]interface{}, 0, len(apps))

	for _, app := range apps {
		appOut = append(appOut, *app.Name)
	}
	return appOut
}

func flattenEc2Attributes(ia *emr.Ec2InstanceAttributes) []map[string]interface{} {
	attrs := map[string]interface{}{}
	result := make([]map[string]interface{}, 0)

	if ia.Ec2KeyName != nil {
		attrs["key_name"] = *ia.Ec2KeyName
	}
	if ia.Ec2SubnetId != nil {
		attrs["subnet_id"] = *ia.Ec2SubnetId
	}
	if ia.IamInstanceProfile != nil {
		attrs["instance_profile"] = *ia.IamInstanceProfile
	}
	if ia.EmrManagedMasterSecurityGroup != nil {
		attrs["emr_managed_master_security_group"] = *ia.EmrManagedMasterSecurityGroup
	}
	if ia.EmrManagedSlaveSecurityGroup != nil {
		attrs["emr_managed_slave_security_group"] = *ia.EmrManagedSlaveSecurityGroup
	}

	if len(ia.AdditionalMasterSecurityGroups) > 0 {
		strs := aws.StringValueSlice(ia.AdditionalMasterSecurityGroups)
		attrs["additional_master_security_groups"] = strings.Join(strs, ",")
	}
	if len(ia.AdditionalSlaveSecurityGroups) > 0 {
		strs := aws.StringValueSlice(ia.AdditionalSlaveSecurityGroups)
		attrs["additional_slave_security_groups"] = strings.Join(strs, ",")
	}

	if ia.ServiceAccessSecurityGroup != nil {
		attrs["service_access_security_group"] = *ia.ServiceAccessSecurityGroup
	}

	result = append(result, attrs)

	return result
}

func flattenInstanceGroups(igs []*emr.InstanceGroup) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	for _, ig := range igs {
		attrs := make(map[string]interface{})
		if ig.BidPrice != nil {
			attrs["bid_price"] = *ig.BidPrice
		} else {
			attrs["bid_price"] = ""
		}
		ebsConfig := make([]map[string]interface{}, 0)
		for _, ebs := range ig.EbsBlockDevices {
			ebsAttrs := make(map[string]interface{})
			if ebs.VolumeSpecification.Iops != nil {
				ebsAttrs["iops"] = *ebs.VolumeSpecification.Iops
			} else {
				ebsAttrs["iops"] = ""
			}
			ebsAttrs["size"] = *ebs.VolumeSpecification.SizeInGB
			ebsAttrs["type"] = *ebs.VolumeSpecification.VolumeType
			ebsAttrs["volumes_per_instance"] = 1

			ebsConfig = append(ebsConfig, ebsAttrs)
		}
		attrs["ebs_config"] = ebsConfig
		attrs["instance_count"] = *ig.RequestedInstanceCount
		attrs["instance_role"] = *ig.InstanceGroupType
		attrs["instance_type"] = *ig.InstanceType
		attrs["name"] = *ig.Name
		result = append(result, attrs)
	}

	return result
}

func flattenBootstrapArguments(actions []*emr.Command) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	for _, b := range actions {
		attrs := make(map[string]interface{})
		attrs["name"] = *b.Name
		attrs["path"] = *b.ScriptPath
		attrs["args"] = flattenStringList(b.Args)
		result = append(result, attrs)
	}

	return result
}

func loadGroups(d *schema.ResourceData, meta interface{}) ([]*emr.InstanceGroup, error) {
	emrconn := meta.(*AWSClient).emrconn
	reqGrps := &emr.ListInstanceGroupsInput{
		ClusterId: aws.String(d.Id()),
	}

	respGrps, errGrps := emrconn.ListInstanceGroups(reqGrps)
	if errGrps != nil {
		return nil, fmt.Errorf("Error reading EMR cluster: %s", errGrps)
	}
	return respGrps.InstanceGroups, nil
}

func findGroup(grps []*emr.InstanceGroup, typ string) *emr.InstanceGroup {
	for _, grp := range grps {
		if grp.InstanceGroupType != nil {
			if *grp.InstanceGroupType == typ {
				return grp
			}
		}
	}
	return nil
}

func expandTags(m map[string]interface{}) []*emr.Tag {
	var result []*emr.Tag
	for k, v := range m {
		result = append(result, &emr.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapEMR(ts []*emr.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}

func diffTagsEMR(oldTags, newTags []*emr.Tag) ([]*emr.Tag, []*emr.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*emr.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return expandTags(create), remove
}

func setTagsEMR(conn *emr.EMR, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsEMR(expandTags(o), expandTags(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %s", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.RemoveTags(&emr.RemoveTagsInput{
				ResourceId: aws.String(d.Id()),
				TagKeys:    k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s", create)
			_, err := conn.AddTags(&emr.AddTagsInput{
				ResourceId: aws.String(d.Id()),
				Tags:       create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func expandBootstrapActions(bootstrapActions []interface{}) []*emr.BootstrapActionConfig {
	actionsOut := []*emr.BootstrapActionConfig{}

	for _, raw := range bootstrapActions {
		actionAttributes := raw.(map[string]interface{})
		actionName := actionAttributes["name"].(string)
		actionPath := actionAttributes["path"].(string)
		actionArgs := actionAttributes["args"].([]interface{})

		action := &emr.BootstrapActionConfig{
			Name: aws.String(actionName),
			ScriptBootstrapAction: &emr.ScriptBootstrapActionConfig{
				Path: aws.String(actionPath),
				Args: expandStringList(actionArgs),
			},
		}
		actionsOut = append(actionsOut, action)
	}

	return actionsOut
}

func expandInstanceGroupConfigs(instanceGroupConfigs []interface{}) []*emr.InstanceGroupConfig {
	configsOut := []*emr.InstanceGroupConfig{}

	for _, raw := range instanceGroupConfigs {
		configAttributes := raw.(map[string]interface{})
		configInstanceRole := configAttributes["instance_role"].(string)
		configInstanceCount := configAttributes["instance_count"].(int)
		configInstanceType := configAttributes["instance_type"].(string)
		configName := configAttributes["name"].(string)
		config := &emr.InstanceGroupConfig{
			Name:          aws.String(configName),
			InstanceRole:  aws.String(configInstanceRole),
			InstanceType:  aws.String(configInstanceType),
			InstanceCount: aws.Int64(int64(configInstanceCount)),
		}

		if bidPrice, ok := configAttributes["bid_price"]; ok {
			if bidPrice != "" {
				config.BidPrice = aws.String(bidPrice.(string))
				config.Market = aws.String("SPOT")
			} else {
				config.Market = aws.String("ON_DEMAND")
			}
		}

		if rawEbsConfigs, ok := configAttributes["ebs_config"]; ok {
			ebsConfig := &emr.EbsConfiguration{}

			ebsBlockDeviceConfigs := make([]*emr.EbsBlockDeviceConfig, 0)
			for _, rawEbsConfig := range rawEbsConfigs.(*schema.Set).List() {
				rawEbsConfig := rawEbsConfig.(map[string]interface{})
				ebsBlockDeviceConfig := &emr.EbsBlockDeviceConfig{
					VolumesPerInstance: aws.Int64(int64(rawEbsConfig["volumes_per_instance"].(int))),
					VolumeSpecification: &emr.VolumeSpecification{
						SizeInGB:   aws.Int64(int64(rawEbsConfig["size"].(int))),
						VolumeType: aws.String(rawEbsConfig["type"].(string)),
					},
				}
				if v, ok := rawEbsConfig["iops"].(int); ok && v != 0 {
					ebsBlockDeviceConfig.VolumeSpecification.Iops = aws.Int64(int64(v))
				}
				ebsBlockDeviceConfigs = append(ebsBlockDeviceConfigs, ebsBlockDeviceConfig)
			}
			ebsConfig.EbsBlockDeviceConfigs = ebsBlockDeviceConfigs

			config.EbsConfiguration = ebsConfig
		}

		configsOut = append(configsOut, config)
	}

	return configsOut
}

func expandConfigures(input string) []*emr.Configuration {
	configsOut := []*emr.Configuration{}
	if strings.HasPrefix(input, "http") {
		if err := readHttpJson(input, &configsOut); err != nil {
			log.Printf("[ERR] Error reading HTTP JSON: %s", err)
		}
	} else if strings.HasSuffix(input, ".json") {
		if err := readLocalJson(input, &configsOut); err != nil {
			log.Printf("[ERR] Error reading local JSON: %s", err)
		}
	} else {
		if err := readBodyJson(input, &configsOut); err != nil {
			log.Printf("[ERR] Error reading body JSON: %s", err)
		}
	}
	log.Printf("[DEBUG] Expanded EMR Configurations %s", configsOut)

	return configsOut
}

func readHttpJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func readLocalJson(localFile string, target interface{}) error {
	file, e := ioutil.ReadFile(localFile)
	if e != nil {
		log.Printf("[ERROR] %s", e)
		return e
	}

	return json.Unmarshal(file, target)
}

func readBodyJson(body string, target interface{}) error {
	log.Printf("[DEBUG] Raw Body %s\n", body)
	err := json.Unmarshal([]byte(body), target)
	if err != nil {
		log.Printf("[ERROR] parsing JSON %s", err)
		return err
	}
	return nil
}

func resourceAwsEMRClusterStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).emrconn

		log.Printf("[INFO] Reading EMR Cluster Information: %s", d.Id())
		params := &emr.DescribeClusterInput{
			ClusterId: aws.String(d.Id()),
		}

		resp, err := conn.DescribeCluster(params)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if "ClusterNotFound" == awsErr.Code() {
					return 42, "destroyed", nil
				}
			}
			log.Printf("[WARN] Error on retrieving EMR Cluster (%s) when waiting: %s", d.Id(), err)
			return nil, "", err
		}

		emrc := resp.Cluster

		if emrc == nil {
			return 42, "destroyed", nil
		}

		if resp.Cluster.Status != nil {
			log.Printf("[DEBUG] EMR Cluster status (%s): %s", d.Id(), *resp.Cluster.Status)
		}

		status := emrc.Status
		if *status.State == "TERMINATING" || *status.State == "TERMINATED_WITH_ERRORS" {
			reason := *status.StateChangeReason
			return emrc, *status.State, fmt.Errorf("%s: %s",
				*reason.Code, *reason.Message)
		}

		return emrc, *status.State, nil
	}
}

package aws

import (
	"log"

	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/schema"
	"io/ioutil"
	"net/http"
	"strings"
)

func resourceAwsEMR() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEMRCreate,
		Read:   resourceAwsEMRRead,
		Update: resourceAwsEMRUpdate,
		Delete: resourceAwsEMRDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"release_label": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"master_instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"core_instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"core_instance_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"log_uri": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"applications": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"ec2_attributes": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"additional_master_security_groups": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"additional_slave_security_groups": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"emr_managed_master_security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"emr_managed_slave_security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"instance_profile": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"bootstrap_action": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"args": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
			"tags": tagsSchema(),
			"configurations": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"service_role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsEMRCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	log.Printf("[DEBUG] Creating EMR cluster")
	masterInstanceType := d.Get("master_instance_type").(string)
	coreInstanceType := masterInstanceType
	if v, ok := d.GetOk("core_instance_type"); ok {
		coreInstanceType = v.(string)
	}
	coreInstanceCount := d.Get("core_instance_count").(int)

	applications := d.Get("applications").(*schema.Set).List()
	var userKey, subnet, extraMasterSecGrp, extraSlaveSecGrp, emrMasterSecGrp, emrSlaveSecGrp, instanceProfile, serviceRole string
	instanceProfile = "EMR_EC2_DefaultRole"
	ec2Attributes := d.Get("ec2_attributes").([]interface{})
	if len(ec2Attributes) == 1 {
		attributes := ec2Attributes[0].(map[string]interface{})
		userKey = attributes["key_name"].(string)
		subnet = attributes["subnet_id"].(string)
		extraMasterSecGrp = attributes["additional_master_security_groups"].(string)
		extraSlaveSecGrp = attributes["additional_slave_security_groups"].(string)
		emrMasterSecGrp = attributes["emr_managed_master_security_group"].(string)
		emrSlaveSecGrp = attributes["emr_managed_slave_security_group"].(string)

		if len(strings.TrimSpace(attributes["instance_profile"].(string))) != 0 {
			instanceProfile = strings.TrimSpace(attributes["instance_profile"].(string))
		}
	}

	if v, ok := d.GetOk("service_role"); ok {
		serviceRole = v.(string)
	} else {
		serviceRole = "EMR_DefaultRole"
	}

	emrApps := expandApplications(applications)

	params := &emr.RunJobFlowInput{
		Instances: &emr.JobFlowInstancesConfig{
			Ec2KeyName:                  aws.String(userKey),
			Ec2SubnetId:                 aws.String(subnet),
			InstanceCount:               aws.Int64(int64(coreInstanceCount + 1)),
			KeepJobFlowAliveWhenNoSteps: aws.Bool(true),
			MasterInstanceType:          aws.String(masterInstanceType),
			SlaveInstanceType:           aws.String(coreInstanceType),
			TerminationProtected:        aws.Bool(false),
			AdditionalMasterSecurityGroups: []*string{
				aws.String(extraMasterSecGrp),
			},
			AdditionalSlaveSecurityGroups: []*string{
				aws.String(extraSlaveSecGrp),
			},
			EmrManagedMasterSecurityGroup: aws.String(emrMasterSecGrp),
			EmrManagedSlaveSecurityGroup:  aws.String(emrSlaveSecGrp),
		},
		Name:         aws.String(d.Get("name").(string)),
		Applications: emrApps,

		JobFlowRole:       aws.String(instanceProfile),
		ReleaseLabel:      aws.String(d.Get("release_label").(string)),
		ServiceRole:       aws.String(serviceRole),
		VisibleToAllUsers: aws.Bool(true),
	}

	if v, ok := d.GetOk("log_uri"); ok {
		logUrl := v.(string)
		params.LogUri = aws.String(logUrl)
	}
	if v, ok := d.GetOk("bootstrap_action"); ok {
		bootstrapActions := v.(*schema.Set).List()
		log.Printf("[DEBUG] %v\n", bootstrapActions)
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
	resp, err := conn.RunJobFlow(params)

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	log.Printf("[DEBUG] Created EMR Cluster done...")
	d.SetId(*resp.JobFlowId)

	return resourceAwsEMRRead(d, meta)
}

func resourceAwsEMRRead(d *schema.ResourceData, meta interface{}) error {
	emrconn := meta.(*AWSClient).emrconn

	req := &emr.DescribeClusterInput{
		ClusterId: aws.String(d.Id()),
	}

	resp, err := emrconn.DescribeCluster(req)
	if err != nil {
		return fmt.Errorf("Error reading EMR cluster: %s", err)
	}

	if resp.Cluster == nil {
		d.SetId("")
		return nil
	}

	instance := resp.Cluster

	if instance.Status != nil {
		if *resp.Cluster.Status.State == "TERMINATED" {
			d.SetId("")
			return nil
		}

		if *resp.Cluster.Status.State == "TERMINATED_WITH_ERRORS" {
			d.SetId("")
			return nil
		}
	}

	instanceGroups, errGrps := loadGroups(d, meta)
	if errGrps == nil {
		coreGroup := findGroup(instanceGroups, "CORE")
		d.Set("core_instance_type", coreGroup.InstanceType)
	}

	return nil
}

func resourceAwsEMRUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	log.Printf("[DEBUG] Modify EMR cluster")
	req := &emr.ListInstanceGroupsInput{
		ClusterId: aws.String(d.Id()),
	}

	respGrps, errGrps := conn.ListInstanceGroups(req)
	if errGrps != nil {
		return fmt.Errorf("Error reading EMR cluster: %s", errGrps)
	}
	instanceGroups := respGrps.InstanceGroups

	coreInstanceCount := d.Get("core_instance_count").(int)
	coreGroup := findGroup(instanceGroups, "CORE")

	params := &emr.ModifyInstanceGroupsInput{
		InstanceGroups: []*emr.InstanceGroupModifyConfig{
			{
				InstanceGroupId: aws.String(*coreGroup.Id),
				InstanceCount:   aws.Int64(int64(coreInstanceCount)),
			},
		},
	}
	respModify, errModify := conn.ModifyInstanceGroups(params)
	if errModify != nil {
		log.Printf("[ERROR] %s", errModify)
		return errModify
	}

	log.Printf("[DEBUG] Modify EMR Cluster done...")

	return nil
}

func resourceAwsEMRDelete(d *schema.ResourceData, meta interface{}) error {
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
		if *grp.InstanceGroupType == typ {
			return grp
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

func expandBootstrapActions(bootstrapActions []interface{}) []*emr.BootstrapActionConfig {
	actionsOut := []*emr.BootstrapActionConfig{}

	for _, raw := range bootstrapActions {
		actionAttributes := raw.(map[string]interface{})
		actionName := actionAttributes["name"].(string)
		actionPath := actionAttributes["path"].(string)
		actionArgs := actionAttributes["args"].(*schema.Set).List()

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

func expandConfigures(input string) []*emr.Configuration {
	configsOut := []*emr.Configuration{}
	if strings.HasPrefix(input, "http") {
		readHttpJson(input, &configsOut)
	} else if strings.HasSuffix(input, ".json") {
		readLocalJson(input, &configsOut)
	} else {
		readBodyJson(input, &configsOut)
	}
	log.Printf("[DEBUG] Configures %v\n", configsOut)

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
		fmt.Printf("File error: %v\n", e)
		log.Printf("[ERROR] %s", e)
		return e
	}
	log.Printf("[DEBUG] %s\n", string(file))

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

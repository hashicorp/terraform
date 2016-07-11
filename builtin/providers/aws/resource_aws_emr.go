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
			"use_default_roles": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"master_instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"core_instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"core_instance_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"log_uri": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"configurations": &schema.Schema{
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
	ec2Attributes := d.Get("ec2_attributes").([]interface{})
	attributes := ec2Attributes[0].(map[string]interface{})
	userKey := attributes["key_name"].(string)
	subnet := attributes["subnet_id"].(string)
	extraMasterSecGrp := attributes["additional_master_security_groups"].(string)
	extraSlaveSecGrp := attributes["additional_slave_security_groups"].(string)
	emrMasterSecGrp := attributes["emr_managed_master_security_group"].(string)
	emrSlaveSecGrp := attributes["emr_managed_slave_security_group"].(string)

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

		JobFlowRole:       aws.String("EMR_EC2_DefaultRole"),
		LogUri:            aws.String(d.Get("log_uri").(string)),
		ReleaseLabel:      aws.String(d.Get("release_label").(string)),
		ServiceRole:       aws.String("EMR_DefaultRole"),
		VisibleToAllUsers: aws.Bool(true),
	}

	if v, ok := d.GetOk("bootstrap_action"); ok {
		bootstrapActions := v.(*schema.Set).List()
		log.Printf("[DEBUG] %v\n", bootstrapActions)
		params.BootstrapActions = expandBootstrapActions(bootstrapActions)
	}
	if v, ok := d.GetOk("tags"); ok {
		tagsIn := v.(*schema.Set).List()
		params.Tags = expandTags(tagsIn)
	}
	if v, ok := d.GetOk("configurations"); ok {
		confUrl := v.(string)
		params.Configurations = expandConfigures(confUrl)
	}

	resp, err := conn.RunJobFlow(params)

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	log.Printf("[DEBUG] Created EMR Cluster done...")
	fmt.Println(resp)
	d.SetId(*resp.JobFlowId)

	return nil
}

func resourceAwsEMRRead(d *schema.ResourceData, meta interface{}) error {

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
	fmt.Println(respGrps)
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

	fmt.Println(respModify)
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

func findGroup(grps []*emr.InstanceGroup, typ string) *emr.InstanceGroup {
	for _, grp := range grps {
		if *grp.InstanceGroupType == typ {
			return grp
		}
	}
	return nil
}

func expandTags(tagsIn []interface{}) []*emr.Tag {
	tagsOut := []*emr.Tag{}

	for _, tagStr := range expandStringList(tagsIn) {
		s := strings.FieldsFunc(*tagStr, func(r rune) bool {
			return r == ':' || r == '='
		})
		if len(s) > 1 {
			key := s[0]
			value := s[1]
			tag := &emr.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			}
			tagsOut = append(tagsOut, tag)
		}
	}
	return tagsOut
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

func expandConfigures(url string) []*emr.Configuration {
	configsOut := []*emr.Configuration{}
	if strings.HasPrefix(url, "http") {
		readHttpJson(url, &configsOut)
	} else {
		readLocalJson(url, &configsOut)
	}
	log.Printf("[DEBUG] %v\n", configsOut)

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

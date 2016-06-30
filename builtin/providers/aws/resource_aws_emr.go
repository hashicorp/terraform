package aws

import (
	"log"

	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/schema"
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
				Required: true,
			},
			"core_instance_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
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
							Optional: true,
						},
						"additional_master_security_groups": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsEMRCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	log.Printf("[DEBUG] Creating EMR cluster")
	masterInstanceType := d.Get("master_instance_type").(string)
	coreInstanceType := d.Get("core_instance_type").(string)
	coreInstanceCount := d.Get("core_instance_count").(int)

	applications := d.Get("applications").(*schema.Set).List()
	ec2Attributes := d.Get("ec2_attributes").([]interface{})
	attributes := ec2Attributes[0].(map[string]interface{})
	userKey := attributes["key_name"].(string)
	subnet := attributes["subnet_id"].(string)
	secGrp := attributes["additional_master_security_groups"].(string)

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
				aws.String(secGrp),
			},
		},
		Name:         aws.String(d.Get("name").(string)),
		Applications: emrApps,

		JobFlowRole:       aws.String("EMR_EC2_DefaultRole"),
		LogUri:            aws.String(d.Get("log_uri").(string)),
		ReleaseLabel:      aws.String(d.Get("release_label").(string)),
		ServiceRole:       aws.String("EMR_DefaultRole"),
		VisibleToAllUsers: aws.Bool(true),
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

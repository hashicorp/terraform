package aws

import (
	"log"

	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
	"strings"
)

func resourceAwsEMRTaskGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEMRTaskGroupCreate,
		Read:   resourceAwsEMRTaskGroupRead,
		Update: resourceAwsEMRTaskGroupUpdate,
		Delete: resourceAwsEMRTaskGroupDelete,
		Schema: map[string]*schema.Schema{
			"cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
			},
		},
	}
}

func resourceAwsEMRTaskGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	clusterId := d.Get("cluster_id").(string)
	instanceType := d.Get("instance_type").(string)
	instanceCount := d.Get("instance_count").(int)

	log.Printf("[DEBUG] Creating EMR cluster task group")
	params := &emr.AddInstanceGroupsInput{
		InstanceGroups: []*emr.InstanceGroupConfig{
			{
				InstanceRole:  aws.String("TASK"),
				InstanceCount: aws.Int64(int64(instanceCount)),
				InstanceType:  aws.String(instanceType),

				//Name:   aws.String("XmlStringMaxLen256"),
			},
		},
		JobFlowId: aws.String(clusterId),
	}
	resp, err := conn.AddInstanceGroups(params)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	fmt.Println(resp)

	log.Printf("[DEBUG] Created EMR Cluster done...")
	d.SetId(*resp.InstanceGroupIds[0])

	return nil
}

func resourceAwsEMRTaskGroupRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceAwsEMRTaskGroupUpdate(d *schema.ResourceData, meta interface{}) error {
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

	grpsTF := d.Get("resize_instance_groups").(*schema.Set).List()
	mdConf, newConf := expandTaskGrps(grpsTF, instanceGroups, d.Get("instance_type").(string))

	if len(mdConf) > 0 {
		params := &emr.ModifyInstanceGroupsInput{
			InstanceGroups: mdConf,
		}
		respModify, errModify := conn.ModifyInstanceGroups(params)
		if errModify != nil {
			log.Printf("[ERROR] %s", errModify)
			return errModify
		}

		fmt.Println(respModify)
	}

	if len(newConf) > 0 {
		newParams := &emr.AddInstanceGroupsInput{
			InstanceGroups: newConf,
			JobFlowId:      aws.String(d.Id()),
		}
		respNew, errNew := conn.AddInstanceGroups(newParams)
		if errNew != nil {
			log.Printf("[ERROR] %s", errNew)
			return errNew
		}

		fmt.Println(respNew)
	}

	log.Printf("[DEBUG] Modify EMR Cluster done...")

	return nil
}

func resourceAwsEMRTaskGroupDelete(d *schema.ResourceData, meta interface{}) error {
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

func expandApps(apps []interface{}) []*emr.Application {
	appOut := make([]*emr.Application, 0, len(apps))

	for _, appName := range expandStringList(apps) {
		app := &emr.Application{
			Name: appName,
		}
		appOut = append(appOut, app)
	}
	return appOut
}

func findTaskGroup(grps []*emr.InstanceGroup, name string) *emr.InstanceGroup {
	for _, grp := range grps {
		if *grp.Name == name {
			return grp
		}
	}
	return nil
}

func expandTaskGrps(grpsTF []interface{},
	grpsEmr []*emr.InstanceGroup, instanceType string) ([]*emr.InstanceGroupModifyConfig,
	[]*emr.InstanceGroupConfig) {
	modiConfOut := []*emr.InstanceGroupModifyConfig{}
	newConfOut := []*emr.InstanceGroupConfig{}

	for _, grp := range expandStringList(grpsTF) {
		s := strings.Split(*grp, ":")
		name := s[0]
		count, _ := strconv.Atoi(s[1])

		oneGrp := findTaskGroup(grpsEmr, name)

		fmt.Println(oneGrp)

		if oneGrp == nil {
			//New TASK group
			confNew := &emr.InstanceGroupConfig{
				InstanceRole:  aws.String("TASK"),
				InstanceCount: aws.Int64(int64(count)),
				InstanceType:  aws.String(instanceType),
				Name:          aws.String(name),
			}
			newConfOut = append(newConfOut, confNew)

		} else if oneGrp != nil {
			//Existed group
			confModi := &emr.InstanceGroupModifyConfig{
				InstanceGroupId: aws.String(*oneGrp.Id),
				InstanceCount:   aws.Int64(int64(count)),
			}
			modiConfOut = append(modiConfOut, confModi)

		}
	}
	return modiConfOut, newConfOut
}

package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsFlowLog() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLogFlowCreate,
		Read:   resourceAwsLogFlowRead,
		Delete: resourceAwsLogFlowDelete,

		Schema: map[string]*schema.Schema{
			"iam_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"log_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vpc_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"subnet_id", "eni_id"},
			},

			"subnet_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"eni_id", "vpc_id"},
			},

			"eni_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"subnet_id", "vpc_id"},
			},

			"traffic_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsLogFlowCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	types := []struct {
		ID   string
		Type string
	}{
		{ID: d.Get("vpc_id").(string), Type: "VPC"},
		{ID: d.Get("subnet_id").(string), Type: "Subnet"},
		{ID: d.Get("eni_id").(string), Type: "NetworkInterface"},
	}

	var resourceId string
	var resourceType string
	for _, t := range types {
		if t.ID != "" {
			resourceId = t.ID
			resourceType = t.Type
			break
		}
	}

	if resourceId == "" || resourceType == "" {
		return fmt.Errorf("Error: Flow Logs require either a VPC, Subnet, or ENI ID")
	}

	opts := &ec2.CreateFlowLogsInput{
		DeliverLogsPermissionArn: aws.String(d.Get("iam_role_arn").(string)),
		LogGroupName:             aws.String(d.Get("log_group_name").(string)),
		ResourceIds:              []*string{aws.String(resourceId)},
		ResourceType:             aws.String(resourceType),
		TrafficType:              aws.String(d.Get("traffic_type").(string)),
	}

	log.Printf(
		"[DEBUG] Flow Log Create configuration: %s", opts)
	resp, err := conn.CreateFlowLogs(opts)
	if err != nil {
		return fmt.Errorf("Error creating Flow Log for (%s), error: %s", resourceId, err)
	}

	if len(resp.FlowLogIds) > 1 {
		return fmt.Errorf("Error: multiple Flow Logs created for (%s)", resourceId)
	}

	d.SetId(*resp.FlowLogIds[0])

	return resourceAwsLogFlowRead(d, meta)
}

func resourceAwsLogFlowRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	opts := &ec2.DescribeFlowLogsInput{
		FlowLogIds: []*string{aws.String(d.Id())},
	}

	resp, err := conn.DescribeFlowLogs(opts)
	if err != nil {
		log.Printf("[WARN] Error describing Flow Logs for id (%s)", d.Id())
		d.SetId("")
		return nil
	}

	if len(resp.FlowLogs) == 0 {
		log.Printf("[WARN] No Flow Logs found for id (%s)", d.Id())
		d.SetId("")
		return nil
	}

	fl := resp.FlowLogs[0]
	d.Set("traffic_type", fl.TrafficType)
	d.Set("log_group_name", fl.LogGroupName)
	d.Set("iam_role_arn", fl.DeliverLogsPermissionArn)

	var resourceKey string
	if strings.HasPrefix(*fl.ResourceId, "vpc-") {
		resourceKey = "vpc_id"
	} else if strings.HasPrefix(*fl.ResourceId, "subnet-") {
		resourceKey = "subnet_id"
	} else if strings.HasPrefix(*fl.ResourceId, "eni-") {
		resourceKey = "eni_id"
	}
	if resourceKey != "" {
		d.Set(resourceKey, fl.ResourceId)
	}

	return nil
}

func resourceAwsLogFlowDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf(
		"[DEBUG] Flow Log Destroy: %s", d.Id())
	_, err := conn.DeleteFlowLogs(&ec2.DeleteFlowLogsInput{
		FlowLogIds: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("[WARN] Error deleting Flow Log with ID (%s), error: %s", d.Id(), err)
	}

	return nil
}

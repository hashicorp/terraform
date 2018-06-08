package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcEndpointConnectionNotification() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcEndpointConnectionNotificationCreate,
		Read:   resourceAwsVpcEndpointConnectionNotificationRead,
		Update: resourceAwsVpcEndpointConnectionNotificationUpdate,
		Delete: resourceAwsVpcEndpointConnectionNotificationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc_endpoint_service_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vpc_endpoint_id"},
			},
			"vpc_endpoint_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vpc_endpoint_service_id"},
			},
			"connection_notification_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"connection_events": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"notification_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsVpcEndpointConnectionNotificationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.CreateVpcEndpointConnectionNotificationInput{
		ConnectionNotificationArn: aws.String(d.Get("connection_notification_arn").(string)),
		ConnectionEvents:          expandStringSet(d.Get("connection_events").(*schema.Set)),
	}
	if v, ok := d.GetOk("vpc_endpoint_service_id"); ok {
		req.ServiceId = aws.String(v.(string))
	} else if v, ok := d.GetOk("vpc_endpoint_id"); ok {
		req.VpcEndpointId = aws.String(v.(string))
	} else {
		return fmt.Errorf(
			"One of ['vpc_endpoint_service_id', 'vpc_endpoint_id'] must be set to create a VPC Endpoint connection notification")
	}

	log.Printf("[DEBUG] Creating VPC Endpoint connection notification: %#v", req)
	resp, err := conn.CreateVpcEndpointConnectionNotification(req)
	if err != nil {
		return fmt.Errorf("Error creating VPC Endpoint connection notification: %s", err.Error())
	}

	d.SetId(aws.StringValue(resp.ConnectionNotification.ConnectionNotificationId))

	return resourceAwsVpcEndpointConnectionNotificationRead(d, meta)
}

func resourceAwsVpcEndpointConnectionNotificationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeVpcEndpointConnectionNotifications(&ec2.DescribeVpcEndpointConnectionNotificationsInput{
		ConnectionNotificationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "InvalidConnectionNotification", "") {
			log.Printf("[WARN] VPC Endpoint connection notification (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading VPC Endpoint connection notification: %s", err.Error())
	}

	return vpcEndpointConnectionNotificationAttributes(d, resp.ConnectionNotificationSet[0])
}

func resourceAwsVpcEndpointConnectionNotificationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.ModifyVpcEndpointConnectionNotificationInput{
		ConnectionNotificationId: aws.String(d.Id()),
	}

	if d.HasChange("connection_notification_arn") {
		req.ConnectionNotificationArn = aws.String(d.Get("connection_notification_arn").(string))
	}

	if d.HasChange("connection_events") {
		req.ConnectionEvents = expandStringSet(d.Get("connection_events").(*schema.Set))
	}

	log.Printf("[DEBUG] Updating VPC Endpoint connection notification: %#v", req)
	if _, err := conn.ModifyVpcEndpointConnectionNotification(req); err != nil {
		return fmt.Errorf("Error updating VPC Endpoint connection notification: %s", err.Error())
	}

	return resourceAwsVpcEndpointConnectionNotificationRead(d, meta)
}

func resourceAwsVpcEndpointConnectionNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Deleting VPC Endpoint connection notification: %s", d.Id())
	_, err := conn.DeleteVpcEndpointConnectionNotifications(&ec2.DeleteVpcEndpointConnectionNotificationsInput{
		ConnectionNotificationIds: aws.StringSlice([]string{d.Id()}),
	})
	if err != nil {
		if isAWSErr(err, "InvalidConnectionNotification", "") {
			log.Printf("[DEBUG] VPC Endpoint connection notification %s is already gone", d.Id())
		} else {
			return fmt.Errorf("Error deleting VPC Endpoint connection notification: %s", err.Error())
		}
	}

	return nil
}

func vpcEndpointConnectionNotificationAttributes(d *schema.ResourceData, cn *ec2.ConnectionNotification) error {
	d.Set("vpc_endpoint_service_id", cn.ServiceId)
	d.Set("vpc_endpoint_id", cn.VpcEndpointId)
	d.Set("connection_notification_arn", cn.ConnectionNotificationArn)
	d.Set("connection_events", flattenStringList(cn.ConnectionEvents))
	d.Set("state", cn.ConnectionNotificationState)
	d.Set("notification_type", cn.ConnectionNotificationType)

	return nil
}

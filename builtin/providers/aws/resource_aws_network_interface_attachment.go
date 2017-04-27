package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkInterfaceAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkInterfaceAttachmentCreate,
		Read:   resourceAwsNetworkInterfaceAttachmentRead,
		Delete: resourceAwsNetworkInterfaceAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"device_index": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_interface_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"attachment_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsNetworkInterfaceAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	device_index := d.Get("device_index").(int)
	instance_id := d.Get("instance_id").(string)
	network_interface_id := d.Get("network_interface_id").(string)

	opts := &ec2.AttachNetworkInterfaceInput{
		DeviceIndex:        aws.Int64(int64(device_index)),
		InstanceId:         aws.String(instance_id),
		NetworkInterfaceId: aws.String(network_interface_id),
	}

	log.Printf("[DEBUG] Attaching network interface (%s) to instance (%s)", network_interface_id, instance_id)
	resp, err := conn.AttachNetworkInterface(opts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("Error attaching network interface (%s) to instance (%s), message: \"%s\", code: \"%s\"",
				network_interface_id, instance_id, awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"false"},
		Target:     []string{"true"},
		Refresh:    networkInterfaceAttachmentRefreshFunc(conn, network_interface_id),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Volume (%s) to attach to Instance: %s, error: %s", network_interface_id, instance_id, err)
	}

	d.SetId(*resp.AttachmentId)
	return resourceAwsNetworkInterfaceAttachmentRead(d, meta)
}

func resourceAwsNetworkInterfaceAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	interfaceId := d.Get("network_interface_id").(string)

	req := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{aws.String(interfaceId)},
	}

	resp, err := conn.DescribeNetworkInterfaces(req)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidNetworkInterfaceID.NotFound" {
			// The ENI is gone now, so just remove the attachment from the state
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ENI: %s", err)
	}
	if len(resp.NetworkInterfaces) != 1 {
		return fmt.Errorf("Unable to find ENI (%s): %#v", interfaceId, resp.NetworkInterfaces)
	}

	eni := resp.NetworkInterfaces[0]

	if eni.Attachment == nil {
		// Interface is no longer attached, remove from state
		d.SetId("")
		return nil
	}

	d.Set("attachment_id", eni.Attachment.AttachmentId)
	d.Set("device_index", eni.Attachment.DeviceIndex)
	d.Set("instance_id", eni.Attachment.InstanceId)
	d.Set("network_interface_id", eni.NetworkInterfaceId)
	d.Set("status", eni.Attachment.Status)

	return nil
}

func resourceAwsNetworkInterfaceAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	interfaceId := d.Get("network_interface_id").(string)

	detach_request := &ec2.DetachNetworkInterfaceInput{
		AttachmentId: aws.String(d.Id()),
		Force:        aws.Bool(true),
	}

	_, detach_err := conn.DetachNetworkInterface(detach_request)
	if detach_err != nil {
		if awsErr, _ := detach_err.(awserr.Error); awsErr.Code() != "InvalidAttachmentID.NotFound" {
			return fmt.Errorf("Error detaching ENI: %s", detach_err)
		}
	}

	log.Printf("[DEBUG] Waiting for ENI (%s) to become dettached", interfaceId)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"true"},
		Target:  []string{"false"},
		Refresh: networkInterfaceAttachmentRefreshFunc(conn, interfaceId),
		Timeout: 10 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for ENI (%s) to become dettached: %s", interfaceId, err)
	}

	return nil
}

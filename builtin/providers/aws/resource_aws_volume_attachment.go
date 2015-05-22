package aws

import (
	"bytes"
	"fmt"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVolumeAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVolumeAttachmentCreate,
		Read:   resourceAwsVolumeAttachmentRead,
		Delete: resourceAwsVolumeAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"device_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"volume_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"force_detach": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsVolumeAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	name := d.Get("device_name").(string)
	iID := d.Get("instance_id").(string)
	vID := d.Get("volume_id").(string)

	opts := &ec2.AttachVolumeInput{
		Device:     aws.String(name),
		InstanceID: aws.String(iID),
		VolumeID:   aws.String(vID),
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "attaching",
		Refresh:    attachVolumeFunc(conn, opts),
		Timeout:    1 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error attaching volume %s to instance %s: %s", vID, iID, err)
	}

	d.SetId(volumeAttachmentID(name, vID, iID))
	return resourceAwsVolumeAttachmentRead(d, meta)
}

func attachVolumeFunc(conn *ec2.EC2, opts *ec2.AttachVolumeInput) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		va, err := conn.AttachVolume(opts)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "VolumeInUse" && *va.InstanceID == *opts.InstanceID {
				return nil, "attaching", nil
			}
			return nil, "error", err
		}
		return va, *va.State, nil
	}
}

func resourceAwsVolumeAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DescribeVolumesInput{
		VolumeIDs: []*string{aws.String(d.Get("volume_id").(string))},
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("attachment.instance-id"),
				Values: []*string{aws.String(d.Get("instance_id").(string))},
			},
		},
	}

	_, err := conn.DescribeVolumes(request)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVolume.NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading EC2 volume %s for instance: %s: %#v", d.Get("volume_id").(string), d.Get("instance_id").(string), err)
	}
	return nil
}

func resourceAwsVolumeAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	volume := d.Get("volume_id").(string)
	instance := d.Get("instance_id").(string)

	opts := &ec2.DetachVolumeInput{
		Device:     aws.String(d.Get("device_name").(string)),
		InstanceID: aws.String(instance),
		VolumeID:   aws.String(volume),
		Force:      aws.Boolean(d.Get("force_detach").(bool)),
	}

	return resource.Retry(1*time.Minute, func() error {
		resp, err := conn.DetachVolume(opts)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && (awsErr.Code() == "IncorrectState" || awsErr.Code() == "InvalidVolume.NotFound") {
				// volume attachment is not in a valid "attachment state"
				return nil
			}

			return err
		}

		if resp.State != nil && *resp.State == "detaching" {
			return fmt.Errorf("waiting for volume %s to detach from instance %s", volume, instance)
		} else if *resp.State == "detached" {
			return nil
		}
		return fmt.Errorf("Error detaching volume %s from instance %s", volume, instance)
	})
}

func volumeAttachmentID(name, volumeID, instanceID string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", name))
	buf.WriteString(fmt.Sprintf("%s-", instanceID))
	buf.WriteString(fmt.Sprintf("%s-", volumeID))

	return fmt.Sprintf("vai-%d", hashcode.String(buf.String()))
}

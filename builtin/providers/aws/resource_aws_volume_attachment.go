package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVolumeAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVolumeAttachmentCreate,
		Update: resourceAwsVolumeAttachmentUpdate,
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
			},

			"volume_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

	log.Printf("[DEBUG] Attaching Volume (%s) to Instance (%s)", vID, iID)
	err := attach(conn, name, iID, vID)
	if err != nil {
		return err
	}

	err = waitForAttach(conn, vID, iID)

	d.SetId(volumeAttachmentID(name, vID, iID))
	return resourceAwsVolumeAttachmentRead(d, meta)
}

func attach(conn *ec2.EC2, name, iID, vID string) error {

	opts := &ec2.AttachVolumeInput{
		Device:     aws.String(name),
		InstanceId: aws.String(iID),
		VolumeId:   aws.String(vID),
	}

	_, err := conn.AttachVolume(opts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error attaching volume (%s) to instance (%s), message: \"%s\", code: \"%s\"",
				vID, iID, awsErr.Message(), awsErr.Code())
		}
		return err
	}
	return nil
}

func volumeAttachmentStateRefreshFunc(conn *ec2.EC2, volumeID, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		request := &ec2.DescribeVolumesInput{
			VolumeIds: []*string{aws.String(volumeID)},
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("attachment.instance-id"),
					Values: []*string{aws.String(instanceID)},
				},
			},
		}

		resp, err := conn.DescribeVolumes(request)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				return nil, "failed", fmt.Errorf("code: %s, message: %s", awsErr.Code(), awsErr.Message())
			}
			return nil, "failed", err
		}

		if len(resp.Volumes) > 0 {
			v := resp.Volumes[0]
			for _, a := range v.Attachments {
				if a.InstanceId != nil && *a.InstanceId == instanceID {
					return a, *a.State, nil
				}
			}
		}
		// assume detached if volume count is 0
		return 42, "detached", nil
	}
}

func resourceAwsVolumeAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{aws.String(d.Get("volume_id").(string))},
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
	name := d.Get("device_name").(string)
	vID := d.Get("volume_id").(string)
	iID := d.Get("instance_id").(string)
	force := d.Get("force_detach").(bool)

	err := detach(conn, name, iID, vID, force)
	if err != nil {
		return err
	}

	err = waitForDetach(conn, vID, iID)

	d.SetId("")
	return err
}

func detach(conn *ec2.EC2, name, iID, vID string, force bool) error {
	opts := &ec2.DetachVolumeInput{
		Device:     aws.String(name),
		InstanceId: aws.String(iID),
		VolumeId:   aws.String(vID),
		Force:      aws.Bool(force),
	}

	log.Printf("[DEBUG] Detaching Volume (%s) from Instance (%s)", vID, iID)
	_, err := conn.DetachVolume(opts)
	return err
}

func waitForDetach(conn *ec2.EC2, vID, iID string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"detaching"},
		Target:     []string{"detached"},
		Refresh:    volumeAttachmentStateRefreshFunc(conn, vID, iID),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Volume (%s) to detach from Instance: %s",
			vID, iID)
	}

	return err
}

func waitForAttach(conn *ec2.EC2, vID, iID string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"attaching"},
		Target:     []string{"attached"},
		Refresh:    volumeAttachmentStateRefreshFunc(conn, vID, iID),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Volume (%s) to attach to Instance: %s, error: %s",
			vID, iID, err)
	}

	return err
}

func resourceAwsVolumeAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	d.Partial(true)
	conn := meta.(*AWSClient).ec2conn
	name_old, name := d.GetChange("device_name")
	iID_old, iID := d.GetChange("instance_id")
	vID_old, vID := d.GetChange("volume_id")
	force := d.Get("force_detach").(bool)

	fmt.Printf("Moving volume from vol %s as dev %s on %s to %s as dev %s on %s",
		vID_old, name_old, iID_old, vID, name, iID)
	err := detach(conn, name_old.(string), iID_old.(string), vID_old.(string), force)
	if err != nil {
		return err
	}

	err = waitForDetach(conn, vID_old.(string), iID_old.(string))
	if err != nil {
		return err
	}

	//we set the ID early since, if the attach fails, it would leave things
	//looking like the old resource was still existing/attached
	d.SetId(volumeAttachmentID(name.(string), vID.(string), iID.(string)))
	fmt.Printf("attaching %s onto %s from volume %s", name, iID, vID)
	err = attach(conn, name.(string), iID.(string), vID.(string))
	if err != nil {
		return err
	}

	err = waitForAttach(conn, vID.(string), iID.(string))
	if err != nil {
		return err
	}

	d.Partial(false)
	return nil
}

func volumeAttachmentID(name, volumeID, instanceID string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", name))
	buf.WriteString(fmt.Sprintf("%s-", instanceID))
	buf.WriteString(fmt.Sprintf("%s-", volumeID))

	return fmt.Sprintf("vai-%d", hashcode.String(buf.String()))
}

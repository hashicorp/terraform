package aws

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEbsVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEbsVolumeCreate,
		Read:   resourceAwsEbsVolumeRead,
		Delete: resourceAwsEbsVolumeDelete,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"iops": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"kms_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEbsVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
	}
	if value, ok := d.GetOk("encrypted"); ok {
		request.Encrypted = aws.Boolean(value.(bool))
	}
	if value, ok := d.GetOk("iops"); ok {
		request.IOPS = aws.Long(int64(value.(int)))
	}
	if value, ok := d.GetOk("kms_key_id"); ok {
		request.KMSKeyID = aws.String(value.(string))
	}
	if value, ok := d.GetOk("size"); ok {
		request.Size = aws.Long(int64(value.(int)))
	}
	if value, ok := d.GetOk("snapshot_id"); ok {
		request.SnapshotID = aws.String(value.(string))
	}
	if value, ok := d.GetOk("type"); ok {
		request.VolumeType = aws.String(value.(string))
	}

	result, err := conn.CreateVolume(request)
	if err != nil {
		return fmt.Errorf("Error creating EC2 volume: %s", err)
	}
	return readVolume(d, result)
}

func resourceAwsEbsVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DescribeVolumesInput{
		VolumeIDs: []*string{aws.String(d.Id())},
	}

	response, err := conn.DescribeVolumes(request)
	if err != nil {
		if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidVolume.NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading EC2 volume %s: %#v", d.Id(), err)
	}

	return readVolume(d, response.Volumes[0])
}

func resourceAwsEbsVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DeleteVolumeInput{
		VolumeID: aws.String(d.Id()),
	}

	_, err := conn.DeleteVolume(request)
	if err != nil {
		return fmt.Errorf("Error deleting EC2 volume %s: %s", d.Id(), err)
	}
	return nil
}

func readVolume(d *schema.ResourceData, volume *ec2.Volume) error {
	d.SetId(*volume.VolumeID)

	d.Set("availability_zone", *volume.AvailabilityZone)
	if volume.Encrypted != nil {
		d.Set("encrypted", *volume.Encrypted)
	}
	if volume.IOPS != nil {
		d.Set("iops", *volume.IOPS)
	}
	if volume.KMSKeyID != nil {
		d.Set("kms_key_id", *volume.KMSKeyID)
	}
	if volume.Size != nil {
		d.Set("size", *volume.Size)
	}
	if volume.SnapshotID != nil {
		d.Set("snapshot_id", *volume.SnapshotID)
	}
	if volume.VolumeType != nil {
		d.Set("type", *volume.VolumeType)
	}

	return nil
}

package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEbsVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEbsVolumeCreate,
		Read:   resourceAwsEbsVolumeRead,
		Update: resourceAWSEbsVolumeUpdate,
		Delete: resourceAwsEbsVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"iops": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"size": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsEbsVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
	}
	if value, ok := d.GetOk("encrypted"); ok {
		request.Encrypted = aws.Bool(value.(bool))
	}
	if value, ok := d.GetOk("kms_key_id"); ok {
		request.KmsKeyId = aws.String(value.(string))
	}
	if value, ok := d.GetOk("size"); ok {
		request.Size = aws.Int64(int64(value.(int)))
	}
	if value, ok := d.GetOk("snapshot_id"); ok {
		request.SnapshotId = aws.String(value.(string))
	}

	// IOPs are only valid, and required for, storage type io1. The current minimu
	// is 100. Instead of a hard validation we we only apply the IOPs to the
	// request if the type is io1, and log a warning otherwise. This allows users
	// to "disable" iops. See https://github.com/hashicorp/terraform/pull/4146
	var t string
	if value, ok := d.GetOk("type"); ok {
		t = value.(string)
		request.VolumeType = aws.String(t)
	}

	iops := d.Get("iops").(int)
	if t != "io1" && iops > 0 {
		log.Printf("[WARN] IOPs is only valid for storate type io1 for EBS Volumes")
	} else if t == "io1" {
		// We add the iops value without validating it's size, to allow AWS to
		// enforce a size requirement (currently 100)
		request.Iops = aws.Int64(int64(iops))
	}

	log.Printf(
		"[DEBUG] EBS Volume create opts: %s", request)
	result, err := conn.CreateVolume(request)
	if err != nil {
		return fmt.Errorf("Error creating EC2 volume: %s", err)
	}

	log.Println("[DEBUG] Waiting for Volume to become available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    volumeStateRefreshFunc(conn, *result.VolumeId),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Volume (%s) to become available: %s",
			*result.VolumeId, err)
	}

	d.SetId(*result.VolumeId)

	if _, ok := d.GetOk("tags"); ok {
		if err := setTags(conn, d); err != nil {
			return fmt.Errorf("Error setting tags for EBS Volume: %s", err)
		}
	}

	return resourceAwsEbsVolumeRead(d, meta)
}

func resourceAWSEbsVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	if _, ok := d.GetOk("tags"); ok {
		if err := setTags(conn, d); err != nil {
			return fmt.Errorf("Error updating tags for EBS Volume: %s", err)
		}
	}

	requestUpdate := false
	params := &ec2.ModifyVolumeInput{
		VolumeId: aws.String(d.Id()),
	}

	if d.HasChange("size") {
		requestUpdate = true
		params.Size = aws.Int64(int64(d.Get("size").(int)))
	}

	if d.HasChange("type") {
		requestUpdate = true
		params.VolumeType = aws.String(d.Get("type").(string))
	}

	if d.HasChange("iops") {
		requestUpdate = true
		params.Iops = aws.Int64(int64(d.Get("iops").(int)))
	}

	if requestUpdate {
		result, err := conn.ModifyVolume(params)
		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"creating", "modifying"},
			Target:     []string{"available", "in-use"},
			Refresh:    volumeStateRefreshFunc(conn, *result.VolumeModification.VolumeId),
			Timeout:    5 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for Volume (%s) to become available: %s",
				*result.VolumeModification.VolumeId, err)
		}
	}

	return resourceAwsEbsVolumeRead(d, meta)
}

// volumeStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a the state of a Volume. Returns successfully when volume is available
func volumeStateRefreshFunc(conn *ec2.EC2, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVolumes(&ec2.DescribeVolumesInput{
			VolumeIds: []*string{aws.String(volumeID)},
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok {
				// Set this to nil as if we didn't find anything.
				log.Printf("Error on Volume State Refresh: message: \"%s\", code:\"%s\"", ec2err.Message(), ec2err.Code())
				resp = nil
				return nil, "", err
			} else {
				log.Printf("Error on Volume State Refresh: %s", err)
				return nil, "", err
			}
		}

		v := resp.Volumes[0]
		return v, *v.State, nil
	}
}

func resourceAwsEbsVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{aws.String(d.Id())},
	}

	response, err := conn.DescribeVolumes(request)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVolume.NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading EC2 volume %s: %s", d.Id(), err)
	}

	return readVolume(d, meta.(*AWSClient), response.Volumes[0])
}

func resourceAwsEbsVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		request := &ec2.DeleteVolumeInput{
			VolumeId: aws.String(d.Id()),
		}
		_, err := conn.DeleteVolume(request)
		if err == nil {
			return nil
		}

		ebsErr, ok := err.(awserr.Error)
		if ebsErr.Code() == "VolumeInUse" {
			return resource.RetryableError(fmt.Errorf("EBS VolumeInUse - trying again while it detaches"))
		}

		if !ok {
			return resource.NonRetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

}

func readVolume(d *schema.ResourceData, client *AWSClient, volume *ec2.Volume) error {
	d.SetId(*volume.VolumeId)

	arn := arn.ARN{
		Partition: client.partition,
		Region:    client.region,
		Service:   "ec2",
		AccountID: client.accountid,
		Resource:  fmt.Sprintf("volume/%s", d.Id()),
	}
	d.Set("arn", arn.String())

	d.Set("availability_zone", *volume.AvailabilityZone)
	if volume.Encrypted != nil {
		d.Set("encrypted", *volume.Encrypted)
	}
	if volume.KmsKeyId != nil {
		d.Set("kms_key_id", *volume.KmsKeyId)
	}
	if volume.Size != nil {
		d.Set("size", *volume.Size)
	}
	if volume.SnapshotId != nil {
		d.Set("snapshot_id", *volume.SnapshotId)
	}
	if volume.VolumeType != nil {
		d.Set("type", *volume.VolumeType)
	}

	if volume.VolumeType != nil && *volume.VolumeType == "io1" {
		// Only set the iops attribute if the volume type is io1. Setting otherwise
		// can trigger a refresh/plan loop based on the computed value that is given
		// from AWS, and prevent us from specifying 0 as a valid iops.
		//   See https://github.com/hashicorp/terraform/pull/4146
		if volume.Iops != nil {
			d.Set("iops", *volume.Iops)
		}
	}

	d.Set("tags", tagsToMap(volume.Tags))

	return nil
}

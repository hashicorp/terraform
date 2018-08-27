package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/storagegateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsStorageGatewayCachedIscsiVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewayCachedIscsiVolumeCreate,
		Read:   resourceAwsStorageGatewayCachedIscsiVolumeRead,
		Delete: resourceAwsStorageGatewayCachedIscsiVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"chap_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"gateway_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"lun_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			// Poor API naming: this accepts the IP address of the network interface
			"network_interface_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network_interface_port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"source_volume_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"target_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"volume_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_size_in_bytes": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsStorageGatewayCachedIscsiVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.CreateCachediSCSIVolumeInput{
		ClientToken:        aws.String(resource.UniqueId()),
		GatewayARN:         aws.String(d.Get("gateway_arn").(string)),
		NetworkInterfaceId: aws.String(d.Get("network_interface_id").(string)),
		TargetName:         aws.String(d.Get("target_name").(string)),
		VolumeSizeInBytes:  aws.Int64(int64(d.Get("volume_size_in_bytes").(int))),
	}

	if v, ok := d.GetOk("snapshot_id"); ok {
		input.SnapshotId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("source_volume_arn"); ok {
		input.SourceVolumeARN = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Storage Gateway cached iSCSI volume: %s", input)
	output, err := conn.CreateCachediSCSIVolume(input)
	if err != nil {
		return fmt.Errorf("error creating Storage Gateway cached iSCSI volume: %s", err)
	}

	d.SetId(aws.StringValue(output.VolumeARN))

	return resourceAwsStorageGatewayCachedIscsiVolumeRead(d, meta)
}

func resourceAwsStorageGatewayCachedIscsiVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DescribeCachediSCSIVolumesInput{
		VolumeARNs: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Reading Storage Gateway cached iSCSI volume: %s", input)
	output, err := conn.DescribeCachediSCSIVolumes(input)

	if err != nil {
		if isAWSErr(err, storagegateway.ErrorCodeVolumeNotFound, "") {
			log.Printf("[WARN] Storage Gateway cached iSCSI volume %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway cached iSCSI volume %q: %s", d.Id(), err)
	}

	if output == nil || len(output.CachediSCSIVolumes) == 0 || output.CachediSCSIVolumes[0] == nil || aws.StringValue(output.CachediSCSIVolumes[0].VolumeARN) != d.Id() {
		log.Printf("[WARN] Storage Gateway cached iSCSI volume %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	volume := output.CachediSCSIVolumes[0]

	d.Set("arn", aws.StringValue(volume.VolumeARN))
	d.Set("snapshot_id", aws.StringValue(volume.SourceSnapshotId))
	d.Set("volume_arn", aws.StringValue(volume.VolumeARN))
	d.Set("volume_id", aws.StringValue(volume.VolumeId))
	d.Set("volume_size_in_bytes", int(aws.Int64Value(volume.VolumeSizeInBytes)))

	if volume.VolumeiSCSIAttributes != nil {
		d.Set("chap_enabled", aws.BoolValue(volume.VolumeiSCSIAttributes.ChapEnabled))
		d.Set("lun_number", int(aws.Int64Value(volume.VolumeiSCSIAttributes.LunNumber)))
		d.Set("network_interface_id", aws.StringValue(volume.VolumeiSCSIAttributes.NetworkInterfaceId))
		d.Set("network_interface_port", int(aws.Int64Value(volume.VolumeiSCSIAttributes.NetworkInterfacePort)))

		targetARN := aws.StringValue(volume.VolumeiSCSIAttributes.TargetARN)
		d.Set("target_arn", targetARN)

		gatewayARN, targetName, err := parseStorageGatewayVolumeGatewayARNAndTargetNameFromARN(targetARN)
		if err != nil {
			return fmt.Errorf("error parsing Storage Gateway volume gateway ARN and target name from target ARN %q: %s", targetARN, err)
		}
		d.Set("gateway_arn", gatewayARN)
		d.Set("target_name", targetName)
	}

	return nil
}

func resourceAwsStorageGatewayCachedIscsiVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DeleteVolumeInput{
		VolumeARN: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Storage Gateway cached iSCSI volume: %s", input)
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteVolume(input)
		if err != nil {
			if isAWSErr(err, storagegateway.ErrorCodeVolumeNotFound, "") {
				return nil
			}
			// InvalidGatewayRequestException: The specified gateway is not connected.
			// Can occur during concurrent DeleteVolume operations
			if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified gateway is not connected") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error deleting Storage Gateway cached iSCSI volume %q: %s", d.Id(), err)
	}

	return nil
}

func parseStorageGatewayVolumeGatewayARNAndTargetNameFromARN(inputARN string) (string, string, error) {
	// inputARN = arn:aws:storagegateway:us-east-2:111122223333:gateway/sgw-12A3456B/target/iqn.1997-05.com.amazon:TargetName
	targetARN, err := arn.Parse(inputARN)
	if err != nil {
		return "", "", err
	}
	// We need to get:
	//  * The Gateway ARN portion of the target ARN resource (gateway/sgw-12A3456B)
	//  * The target name portion of the target ARN resource (TargetName)
	// First, let's split up the resource of the target ARN
	// targetARN.Resource = gateway/sgw-12A3456B/target/iqn.1997-05.com.amazon:TargetName
	expectedFormatErr := fmt.Errorf("expected resource format gateway/sgw-12A3456B/target/iqn.1997-05.com.amazon:TargetName, received: %s", targetARN.Resource)
	resourceParts := strings.SplitN(targetARN.Resource, "/", 4)
	if len(resourceParts) != 4 {
		return "", "", expectedFormatErr
	}
	gatewayARN := arn.ARN{
		AccountID: targetARN.AccountID,
		Partition: targetARN.Partition,
		Region:    targetARN.Region,
		Resource:  fmt.Sprintf("%s/%s", resourceParts[0], resourceParts[1]),
		Service:   targetARN.Service,
	}.String()
	// Second, let's split off the target name from the initiator name
	// resourceParts[3] = iqn.1997-05.com.amazon:TargetName
	initiatorParts := strings.SplitN(resourceParts[3], ":", 2)
	if len(initiatorParts) != 2 {
		return "", "", expectedFormatErr
	}
	targetName := initiatorParts[1]
	return gatewayARN, targetName, nil
}

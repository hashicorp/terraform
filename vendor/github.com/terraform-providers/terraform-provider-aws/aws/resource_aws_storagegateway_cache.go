package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/storagegateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsStorageGatewayCache() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewayCacheCreate,
		Read:   resourceAwsStorageGatewayCacheRead,
		Delete: schema.Noop,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"disk_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"gateway_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
		},
	}
}

func resourceAwsStorageGatewayCacheCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	diskID := d.Get("disk_id").(string)
	gatewayARN := d.Get("gateway_arn").(string)

	input := &storagegateway.AddCacheInput{
		DiskIds:    []*string{aws.String(diskID)},
		GatewayARN: aws.String(gatewayARN),
	}

	log.Printf("[DEBUG] Adding Storage Gateway cache: %s", input)
	_, err := conn.AddCache(input)
	if err != nil {
		return fmt.Errorf("error adding Storage Gateway cache: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", gatewayARN, diskID))

	// Depending on the Storage Gateway software, it will sometimes relabel a local DiskId
	// with a UUID if previously unlabeled, e.g.
	//   Before conn.AddCache(): "DiskId": "/dev/xvdb",
	//   After conn.AddCache(): "DiskId": "112764d7-7e83-42ce-9af3-d482985a31cc",
	// This prevents us from successfully reading the disk after creation.
	// Here we try to refresh the local disks to see if we can find a new DiskId.

	listLocalDisksInput := &storagegateway.ListLocalDisksInput{
		GatewayARN: aws.String(gatewayARN),
	}

	log.Printf("[DEBUG] Reading Storage Gateway Local Disk: %s", listLocalDisksInput)
	output, err := conn.ListLocalDisks(listLocalDisksInput)
	if err != nil {
		return fmt.Errorf("error reading Storage Gateway Local Disk: %s", err)
	}

	if output != nil {
		for _, disk := range output.Disks {
			if aws.StringValue(disk.DiskId) == diskID || aws.StringValue(disk.DiskNode) == diskID || aws.StringValue(disk.DiskPath) == diskID {
				diskID = aws.StringValue(disk.DiskId)
				break
			}
		}
	}

	d.SetId(fmt.Sprintf("%s:%s", gatewayARN, diskID))

	return resourceAwsStorageGatewayCacheRead(d, meta)
}

func resourceAwsStorageGatewayCacheRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	gatewayARN, diskID, err := decodeStorageGatewayCacheID(d.Id())
	if err != nil {
		return err
	}

	input := &storagegateway.DescribeCacheInput{
		GatewayARN: aws.String(gatewayARN),
	}

	log.Printf("[DEBUG] Reading Storage Gateway cache: %s", input)
	output, err := conn.DescribeCache(input)
	if err != nil {
		if isAWSErrStorageGatewayGatewayNotFound(err) {
			log.Printf("[WARN] Storage Gateway cache %q not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway cache: %s", err)
	}

	if output == nil || len(output.DiskIds) == 0 {
		log.Printf("[WARN] Storage Gateway cache %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	found := false
	for _, existingDiskID := range output.DiskIds {
		if aws.StringValue(existingDiskID) == diskID {
			found = true
			break
		}
	}

	if !found {
		log.Printf("[WARN] Storage Gateway cache %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("disk_id", diskID)
	d.Set("gateway_arn", gatewayARN)

	return nil
}

func decodeStorageGatewayCacheID(id string) (string, string, error) {
	// id = arn:aws:storagegateway:us-east-1:123456789012:gateway/sgw-12345678:pci-0000:03:00.0-scsi-0:0:0:0
	idFormatErr := fmt.Errorf("expected ID in form of GatewayARN:DiskId, received: %s", id)
	gatewayARNAndDisk, err := arn.Parse(id)
	if err != nil {
		return "", "", idFormatErr
	}
	// gatewayARNAndDisk.Resource = gateway/sgw-12345678:pci-0000:03:00.0-scsi-0:0:0:0
	resourceParts := strings.SplitN(gatewayARNAndDisk.Resource, ":", 2)
	if len(resourceParts) != 2 {
		return "", "", idFormatErr
	}
	// resourceParts = ["gateway/sgw-12345678", "pci-0000:03:00.0-scsi-0:0:0:0"]
	gatewayARN := &arn.ARN{
		AccountID: gatewayARNAndDisk.AccountID,
		Partition: gatewayARNAndDisk.Partition,
		Region:    gatewayARNAndDisk.Region,
		Service:   gatewayARNAndDisk.Service,
		Resource:  resourceParts[0],
	}
	return gatewayARN.String(), resourceParts[1], nil
}

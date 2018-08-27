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

func resourceAwsStorageGatewayUploadBuffer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewayUploadBufferCreate,
		Read:   resourceAwsStorageGatewayUploadBufferRead,
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

func resourceAwsStorageGatewayUploadBufferCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	diskID := d.Get("disk_id").(string)
	gatewayARN := d.Get("gateway_arn").(string)

	input := &storagegateway.AddUploadBufferInput{
		DiskIds:    []*string{aws.String(diskID)},
		GatewayARN: aws.String(gatewayARN),
	}

	log.Printf("[DEBUG] Adding Storage Gateway upload buffer: %s", input)
	_, err := conn.AddUploadBuffer(input)
	if err != nil {
		return fmt.Errorf("error adding Storage Gateway upload buffer: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", gatewayARN, diskID))

	return resourceAwsStorageGatewayUploadBufferRead(d, meta)
}

func resourceAwsStorageGatewayUploadBufferRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	gatewayARN, diskID, err := decodeStorageGatewayUploadBufferID(d.Id())
	if err != nil {
		return err
	}

	input := &storagegateway.DescribeUploadBufferInput{
		GatewayARN: aws.String(gatewayARN),
	}

	log.Printf("[DEBUG] Reading Storage Gateway upload buffer: %s", input)
	output, err := conn.DescribeUploadBuffer(input)
	if err != nil {
		if isAWSErrStorageGatewayGatewayNotFound(err) {
			log.Printf("[WARN] Storage Gateway upload buffer %q not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway upload buffer: %s", err)
	}

	if output == nil || len(output.DiskIds) == 0 {
		log.Printf("[WARN] Storage Gateway upload buffer %q not found - removing from state", d.Id())
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
		log.Printf("[WARN] Storage Gateway upload buffer %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("disk_id", diskID)
	d.Set("gateway_arn", gatewayARN)

	return nil
}

func decodeStorageGatewayUploadBufferID(id string) (string, string, error) {
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

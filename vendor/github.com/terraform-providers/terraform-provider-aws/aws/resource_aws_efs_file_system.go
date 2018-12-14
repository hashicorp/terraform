package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEfsFileSystem() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEfsFileSystemCreate,
		Read:   resourceAwsEfsFileSystemRead,
		Update: resourceAwsEfsFileSystemUpdate,
		Delete: resourceAwsEfsFileSystemDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_token": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},

			"reference_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Deprecated:   "Please use attribute `creation_token' instead. This attribute might be removed in future releases.",
				ValidateFunc: validateReferenceName,
			},

			"performance_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					efs.PerformanceModeGeneralPurpose,
					efs.PerformanceModeMaxIo,
				}, false),
			},

			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"provisioned_throughput_in_mibps": {
				Type:     schema.TypeFloat,
				Optional: true,
			},

			"tags": tagsSchema(),

			"throughput_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  efs.ThroughputModeBursting,
				ValidateFunc: validation.StringInSlice([]string{
					efs.ThroughputModeBursting,
					efs.ThroughputModeProvisioned,
				}, false),
			},
		},
	}
}

func resourceAwsEfsFileSystemCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	creationToken := ""
	if v, ok := d.GetOk("creation_token"); ok {
		creationToken = v.(string)
	} else {
		if v, ok := d.GetOk("reference_name"); ok {
			creationToken = resource.PrefixedUniqueId(fmt.Sprintf("%s-", v.(string)))
			log.Printf("[WARN] Using deprecated `reference_name' attribute.")
		} else {
			creationToken = resource.UniqueId()
		}
	}
	throughputMode := d.Get("throughput_mode").(string)

	createOpts := &efs.CreateFileSystemInput{
		CreationToken:  aws.String(creationToken),
		ThroughputMode: aws.String(throughputMode),
	}

	if v, ok := d.GetOk("performance_mode"); ok {
		createOpts.PerformanceMode = aws.String(v.(string))
	}

	if throughputMode == efs.ThroughputModeProvisioned {
		createOpts.ProvisionedThroughputInMibps = aws.Float64(d.Get("provisioned_throughput_in_mibps").(float64))
	}

	encrypted, hasEncrypted := d.GetOk("encrypted")
	kmsKeyId, hasKmsKeyId := d.GetOk("kms_key_id")

	if hasEncrypted {
		createOpts.Encrypted = aws.Bool(encrypted.(bool))
	}

	if hasKmsKeyId {
		createOpts.KmsKeyId = aws.String(kmsKeyId.(string))
	}

	if encrypted == false && hasKmsKeyId {
		return errors.New("encrypted must be set to true when kms_key_id is specified")
	}

	log.Printf("[DEBUG] EFS file system create options: %#v", *createOpts)
	fs, err := conn.CreateFileSystem(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating EFS file system: %s", err)
	}

	d.SetId(*fs.FileSystemId)
	log.Printf("[INFO] EFS file system ID: %s", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{efs.LifeCycleStateCreating},
		Target:     []string{efs.LifeCycleStateAvailable},
		Refresh:    resourceEfsFileSystemCreateUpdateRefreshFunc(d.Id(), conn),
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS file system (%q) to create: %s", d.Id(), err)
	}
	log.Printf("[DEBUG] EFS file system %q created.", d.Id())

	err = setTagsEFS(conn, d)
	if err != nil {
		return fmt.Errorf("error setting tags for EFS file system (%q): %s", d.Id(), err)
	}

	return resourceAwsEfsFileSystemRead(d, meta)
}

func resourceAwsEfsFileSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	if d.HasChange("provisioned_throughput_in_mibps") || d.HasChange("throughput_mode") {
		throughputMode := d.Get("throughput_mode").(string)

		input := &efs.UpdateFileSystemInput{
			FileSystemId:   aws.String(d.Id()),
			ThroughputMode: aws.String(throughputMode),
		}

		if throughputMode == efs.ThroughputModeProvisioned {
			input.ProvisionedThroughputInMibps = aws.Float64(d.Get("provisioned_throughput_in_mibps").(float64))
		}

		_, err := conn.UpdateFileSystem(input)
		if err != nil {
			return fmt.Errorf("error updating EFS File System %q: %s", d.Id(), err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{efs.LifeCycleStateUpdating},
			Target:     []string{efs.LifeCycleStateAvailable},
			Refresh:    resourceEfsFileSystemCreateUpdateRefreshFunc(d.Id(), conn),
			Timeout:    10 * time.Minute,
			Delay:      2 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("error waiting for EFS file system (%q) to update: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags") {
		err := setTagsEFS(conn, d)
		if err != nil {
			return fmt.Errorf("Error setting EC2 tags for EFS file system (%q): %s",
				d.Id(), err.Error())
		}
	}

	return resourceAwsEfsFileSystemRead(d, meta)
}

func resourceAwsEfsFileSystemRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "FileSystemNotFound" {
			log.Printf("[WARN] EFS file system (%s) could not be found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if hasEmptyFileSystems(resp) {
		return fmt.Errorf("EFS file system %q could not be found.", d.Id())
	}

	tags := make([]*efs.Tag, 0)
	var marker string
	for {
		params := &efs.DescribeTagsInput{
			FileSystemId: aws.String(d.Id()),
		}
		if marker != "" {
			params.Marker = aws.String(marker)
		}

		tagsResp, err := conn.DescribeTags(params)
		if err != nil {
			return fmt.Errorf("Error retrieving EC2 tags for EFS file system (%q): %s",
				d.Id(), err.Error())
		}

		for _, tag := range tagsResp.Tags {
			tags = append(tags, tag)
		}

		if tagsResp.NextMarker != nil {
			marker = *tagsResp.NextMarker
		} else {
			break
		}
	}

	err = d.Set("tags", tagsToMapEFS(tags))
	if err != nil {
		return err
	}

	var fs *efs.FileSystemDescription
	for _, f := range resp.FileSystems {
		if d.Id() == *f.FileSystemId {
			fs = f
			break
		}
	}
	if fs == nil {
		log.Printf("[WARN] EFS (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	fsARN := arn.ARN{
		AccountID: meta.(*AWSClient).accountid,
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Resource:  fmt.Sprintf("file-system/%s", aws.StringValue(fs.FileSystemId)),
		Service:   "elasticfilesystem",
	}.String()

	d.Set("arn", fsARN)
	d.Set("creation_token", fs.CreationToken)
	d.Set("encrypted", fs.Encrypted)
	d.Set("kms_key_id", fs.KmsKeyId)
	d.Set("performance_mode", fs.PerformanceMode)
	d.Set("provisioned_throughput_in_mibps", fs.ProvisionedThroughputInMibps)
	d.Set("throughput_mode", fs.ThroughputMode)

	region := meta.(*AWSClient).region
	err = d.Set("dns_name", resourceAwsEfsDnsName(*fs.FileSystemId, region))
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsEfsFileSystemDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	log.Printf("[DEBUG] Deleting EFS file system: %s", d.Id())
	_, err := conn.DeleteFileSystem(&efs.DeleteFileSystemInput{
		FileSystemId: aws.String(d.Id()),
	})
	stateConf := &resource.StateChangeConf{
		Pending: []string{"available", "deleting"},
		Target:  []string{},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
				FileSystemId: aws.String(d.Id()),
			})
			if err != nil {
				efsErr, ok := err.(awserr.Error)
				if ok && efsErr.Code() == "FileSystemNotFound" {
					return nil, "", nil
				}
				return nil, "error", err
			}

			if hasEmptyFileSystems(resp) {
				return nil, "", nil
			}

			fs := resp.FileSystems[0]
			log.Printf("[DEBUG] current status of %q: %q", *fs.FileSystemId, *fs.LifeCycleState)
			return fs, *fs.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS file system (%q) to delete: %s",
			d.Id(), err.Error())
	}

	log.Printf("[DEBUG] EFS file system %q deleted.", d.Id())

	return nil
}

func validateReferenceName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	creationToken := resource.PrefixedUniqueId(fmt.Sprintf("%s-", value))
	if len(creationToken) > 64 {
		errors = append(errors, fmt.Errorf(
			"%q cannot take the Creation Token over the limit of 64 characters: %q", k, value))
	}
	return
}

func hasEmptyFileSystems(fs *efs.DescribeFileSystemsOutput) bool {
	if fs != nil && len(fs.FileSystems) > 0 {
		return false
	}
	return true
}

func resourceAwsEfsDnsName(fileSystemId, region string) string {
	return fmt.Sprintf("%s.efs.%s.amazonaws.com", fileSystemId, region)
}

func resourceEfsFileSystemCreateUpdateRefreshFunc(id string, conn *efs.EFS) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(id),
		})
		if err != nil {
			return nil, "error", err
		}

		if hasEmptyFileSystems(resp) {
			return nil, "not-found", fmt.Errorf("EFS file system %q could not be found.", id)
		}

		fs := resp.FileSystems[0]
		state := aws.StringValue(fs.LifeCycleState)
		log.Printf("[DEBUG] current status of %q: %q", id, state)
		return fs, state, nil
	}
}

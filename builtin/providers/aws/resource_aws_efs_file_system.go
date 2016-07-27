package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
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
			"reference_name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateReferenceName,
			},

			"performance_mode": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validatePerformanceModeType,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsEfsFileSystemCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	creationToken := ""
	if v, ok := d.GetOk("reference_name"); ok {
		creationToken = resource.PrefixedUniqueId(fmt.Sprintf("%s-", v.(string)))
	} else {
		creationToken = resource.UniqueId()
	}

	createOpts := &efs.CreateFileSystemInput{
		CreationToken: aws.String(creationToken),
	}

	if v, ok := d.GetOk("performance_mode"); ok {
		createOpts.PerformanceMode = aws.String(v.(string))
	}

	log.Printf("[DEBUG] EFS file system create options: %#v", *createOpts)
	fs, err := conn.CreateFileSystem(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating EFS file system: %s", err)
	}

	d.SetId(*fs.FileSystemId)
	log.Printf("[INFO] EFS file system ID: %s", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating"},
		Target:  []string{"available"},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
				FileSystemId: aws.String(d.Id()),
			})
			if err != nil {
				return nil, "error", err
			}

			if len(resp.FileSystems) < 1 {
				return nil, "not-found", fmt.Errorf("EFS file system %q not found", d.Id())
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
		return fmt.Errorf("Error waiting for EFS file system (%q) to create: %q",
			d.Id(), err.Error())
	}
	log.Printf("[DEBUG] EFS file system %q created.", d.Id())

	return resourceAwsEfsFileSystemUpdate(d, meta)
}

func resourceAwsEfsFileSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn
	err := setTagsEFS(conn, d)
	if err != nil {
		return fmt.Errorf("Error setting EC2 tags for EFS file system (%q): %q",
			d.Id(), err.Error())
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
			log.Printf("[WARN] EFS File System (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	if len(resp.FileSystems) < 1 {
		return fmt.Errorf("EFS file system %q not found", d.Id())
	}

	tagsResp, err := conn.DescribeTags(&efs.DescribeTagsInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error retrieving EC2 tags for EFS file system (%q): %q",
			d.Id(), err.Error())
	}

	d.Set("tags", tagsToMapEFS(tagsResp.Tags))

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

			if len(resp.FileSystems) < 1 {
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
		return fmt.Errorf("Error waiting for EFS file system (%q) to delete: %q",
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

func validatePerformanceModeType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "generalPurpose" && value != "maxIO" {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Performance Mode %q. Valid modes are either %q or %q",
			k, value, "generalPurpose", "maxIO"))
	}
	return
}

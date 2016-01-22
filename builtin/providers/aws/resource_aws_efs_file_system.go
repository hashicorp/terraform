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

		Schema: map[string]*schema.Schema{
			"reference_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsEfsFileSystemCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	referenceName := ""
	if v, ok := d.GetOk("reference_name"); ok {
		referenceName = v.(string) + "-"
	}
	token := referenceName + resource.UniqueId()
	fs, err := conn.CreateFileSystem(&efs.CreateFileSystemInput{
		CreationToken: aws.String(token),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating EFS file system: %s", *fs)
	d.SetId(*fs.FileSystemId)

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
	log.Printf("[DEBUG] EFS file system created: %q", *fs.FileSystemId)

	return resourceAwsEfsFileSystemUpdate(d, meta)
}

func resourceAwsEfsFileSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn
	err := setTagsEFS(conn, d)
	if err != nil {
		return err
	}

	return resourceAwsEfsFileSystemRead(d, meta)
}

func resourceAwsEfsFileSystemRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	if len(resp.FileSystems) < 1 {
		return fmt.Errorf("EFS file system %q not found", d.Id())
	}

	tagsResp, err := conn.DescribeTags(&efs.DescribeTagsInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	d.Set("tags", tagsToMapEFS(tagsResp.Tags))

	return nil
}

func resourceAwsEfsFileSystemDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	log.Printf("[DEBUG] Deleting EFS file system %s", d.Id())
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
			log.Printf("[DEBUG] current status of %q: %q",
				*fs.FileSystemId, *fs.LifeCycleState)
			return fs, *fs.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] EFS file system %q deleted.", d.Id())

	return nil
}

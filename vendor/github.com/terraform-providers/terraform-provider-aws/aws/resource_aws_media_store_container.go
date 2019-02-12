package aws

import (
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mediastore"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsMediaStoreContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsMediaStoreContainerCreate,
		Read:   resourceAwsMediaStoreContainerRead,
		Delete: resourceAwsMediaStoreContainerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\w+$`), "must contain alphanumeric characters or underscores"),
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsMediaStoreContainerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mediastoreconn

	input := &mediastore.CreateContainerInput{
		ContainerName: aws.String(d.Get("name").(string)),
	}

	_, err := conn.CreateContainer(input)
	if err != nil {
		return err
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{mediastore.ContainerStatusCreating},
		Target:     []string{mediastore.ContainerStatusActive},
		Refresh:    mediaStoreContainerRefreshStatusFunc(conn, d.Get("name").(string)),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsMediaStoreContainerRead(d, meta)
}

func resourceAwsMediaStoreContainerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mediastoreconn

	input := &mediastore.DescribeContainerInput{
		ContainerName: aws.String(d.Id()),
	}
	resp, err := conn.DescribeContainer(input)
	if err != nil {
		return err
	}
	d.Set("arn", resp.Container.ARN)
	d.Set("name", resp.Container.Name)
	d.Set("endpoint", resp.Container.Endpoint)
	return nil
}

func resourceAwsMediaStoreContainerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mediastoreconn

	input := &mediastore.DeleteContainerInput{
		ContainerName: aws.String(d.Id()),
	}
	_, err := conn.DeleteContainer(input)
	if err != nil {
		if isAWSErr(err, mediastore.ErrCodeContainerNotFoundException, "") {
			return nil
		}
		return err
	}

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		dcinput := &mediastore.DescribeContainerInput{
			ContainerName: aws.String(d.Id()),
		}
		_, err := conn.DescribeContainer(dcinput)
		if err != nil {
			if isAWSErr(err, mediastore.ErrCodeContainerNotFoundException, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("Media Store Container (%s) still exists", d.Id()))
	})
	if err != nil {
		return fmt.Errorf("error waiting for Media Store Container (%s) deletion: %s", d.Id(), err)
	}

	return nil
}

func mediaStoreContainerRefreshStatusFunc(conn *mediastore.MediaStore, cn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &mediastore.DescribeContainerInput{
			ContainerName: aws.String(cn),
		}
		resp, err := conn.DescribeContainer(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Container.Status, nil
	}
}

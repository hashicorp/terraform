package aws

import (
	"log"
	"time"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrRepositoryCreate,
		Read:   resourceAwsEcrRepositoryRead,
		Update: resourceAwsEcrRepositoryUpdate,
		Delete: resourceAwsEcrRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"registry_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"repository_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := ecr.CreateRepositoryInput{
		RepositoryName: aws.String(d.Get("name").(string)),
	}

	log.Printf("[DEBUG] Creating ECR repository: %#v", input)
	out, err := conn.CreateRepository(&input)
	if err != nil {
		return fmt.Errorf("error creating ECR repository: %s", err)
	}

	repository := *out.Repository

	log.Printf("[DEBUG] ECR repository created: %q", *repository.RepositoryArn)

	d.SetId(aws.StringValue(repository.RepositoryName))
	// ARN required for setting any tags.
	d.Set("arn", repository.RepositoryArn)

	if err := setTagsECR(conn, d); err != nil {
		return fmt.Errorf("error setting ECR repository tags: %s", err)
	}

	return resourceAwsEcrRepositoryRead(d, meta)
}

func resourceAwsEcrRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	log.Printf("[DEBUG] Reading ECR repository %s", d.Id())
	var out *ecr.DescribeRepositoriesOutput
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: aws.StringSlice([]string{d.Id()}),
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.DescribeRepositories(input)
		if d.IsNewResource() && isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
		log.Printf("[WARN] ECR Repository (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading ECR repository: %s", err)
	}

	repository := out.Repositories[0]

	d.Set("arn", repository.RepositoryArn)
	d.Set("name", repository.RepositoryName)
	d.Set("registry_id", repository.RegistryId)
	d.Set("repository_url", repository.RepositoryUri)

	if err := getTagsECR(conn, d); err != nil {
		return fmt.Errorf("error getting ECR repository tags: %s", err)
	}

	return nil
}

func resourceAwsEcrRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	if err := setTagsECR(conn, d); err != nil {
		return fmt.Errorf("error setting ECR repository tags: %s", err)
	}

	return resourceAwsEcrRepositoryRead(d, meta)
}

func resourceAwsEcrRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	_, err := conn.DeleteRepository(&ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(d.Id()),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		Force:          aws.Bool(true),
	})
	if err != nil {
		if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting ECR repository: %s", err)
	}

	log.Printf("[DEBUG] Waiting for ECR Repository %q to be deleted", d.Id())
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.DescribeRepositories(&ecr.DescribeRepositoriesInput{
			RepositoryNames: aws.StringSlice([]string{d.Id()}),
		})
		if err != nil {
			if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for the ECR Repository to be deleted", d.Id()))
	})
	if err != nil {
		return fmt.Errorf("error deleting ECR repository: %s", err)
	}

	log.Printf("[DEBUG] repository %q deleted.", d.Get("name").(string))

	return nil
}

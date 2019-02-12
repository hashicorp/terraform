package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEcrRepository() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEcrRepositoryRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
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
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsEcrRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	params := &ecr.DescribeRepositoriesInput{
		RepositoryNames: aws.StringSlice([]string{d.Get("name").(string)}),
	}
	log.Printf("[DEBUG] Reading ECR repository: %s", params)
	out, err := conn.DescribeRepositories(params)
	if err != nil {
		if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
			log.Printf("[WARN] ECR Repository %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading ECR repository: %s", err)
	}

	repository := out.Repositories[0]

	log.Printf("[DEBUG] Received ECR repository %s", out)

	d.SetId(aws.StringValue(repository.RepositoryName))
	d.Set("arn", repository.RepositoryArn)
	d.Set("registry_id", repository.RegistryId)
	d.Set("name", repository.RepositoryName)
	d.Set("repository_url", repository.RepositoryUri)

	if err := getTagsECR(conn, d); err != nil {
		return fmt.Errorf("error getting ECR repository tags: %s", err)
	}

	return nil
}

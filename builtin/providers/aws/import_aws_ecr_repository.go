package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrRepositoryImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ecrconn

	id := d.Id()
	resp, err := conn.DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Repositories) < 1 || resp.Repositories[0] == nil {
		return nil, fmt.Errorf("ECR %s is not found", id)
	}
	ecr := resp.Repositories[0]

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	d.SetId(id)
	d.SetType("aws_ecr_repository")
	d.Set("registry_id", ecr.RegistryId)
	d.Set("repository_url", ecr.RepositoryUri)
	d.Set("arn", ecr.RepositoryArn)
	d.Set("name", d.Id())

	return results, nil

}

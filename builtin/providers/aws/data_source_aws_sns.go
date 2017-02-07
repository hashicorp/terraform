package aws

import (
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSnsTopic() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSnsTopicsRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsSnsTopicsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).snsconn
	params := &sns.ListTopicsInput{
		NextToken: aws.String("nextToken"),
	}

	target := d.Get("name")

	validNamePattern := "^[A-Za-z0-9_-]+$"
	validName, nameMatchErr := regexp.MatchString(validNamePattern, target.(string))
	if !validName || nameMatchErr != nil {
		return fmt.Errorf("Supplied topic name %v is invalid, should match regex '%v'.", target, validNamePattern)
	}

	var arns []string
	err := conn.ListTopicsPages(params, func(page *sns.ListTopicsOutput, lastPage bool) bool {
		for _, topic := range page.Topics {
			topicPattern := fmt.Sprintf(".*:%v$", target)
			matched, regexpErr := regexp.MatchString(topicPattern, *topic.TopicArn)
			if matched && regexpErr == nil {
				arns = append(arns, *topic.TopicArn)
			}
		}

		return true
	})
	if err != nil {
		return errwrap.Wrapf("Error describing topics: {{err}}", err)
	}

	if len(arns) == 0 {
		return fmt.Errorf("No topic with name %q found in this region.", target)
	}
	if len(arns) > 1 {
		return fmt.Errorf("Multiple topics with name %q found in this region.", target)
	}

	d.Set("arn", arns[0])

	return nil
}

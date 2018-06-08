package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

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
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					validNamePattern := "^[A-Za-z0-9_-]+$"
					validName, nameMatchErr := regexp.MatchString(validNamePattern, value)
					if !validName || nameMatchErr != nil {
						errors = append(errors, fmt.Errorf(
							"%q must match regex '%v'", k, validNamePattern))
					}
					return
				},
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
	params := &sns.ListTopicsInput{}

	target := d.Get("name")
	var arns []string
	log.Printf("[DEBUG] Reading SNS Topic: %s", params)
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

	d.SetId(time.Now().UTC().String())
	d.Set("arn", arns[0])

	return nil
}

package aws

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodeCommitTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeCommitRepositoryCreate,
		Read:   resourceAwsCodeCommitRepositoryRead,

		Schema: map[string]*schema.Schema{
			"repository_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 100 characters", k))
					}
					return
				},
			},

			"trigger": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"destination_arn": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"custom_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"branches": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"events": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceAwsCodeCommitTriggerHash,
			},

			// Computed values.
			"configuration_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCodeCommitTriggerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn
	region := meta.(*AWSClient).region

	//	This is a temporary thing - we need to ensure that CodeCommit is only being run against us-east-1
	//	As this is the only place that AWS currently supports it
	if region != "us-east-1" {
		return fmt.Errorf("CodeCommit can only be used with us-east-1. You are trying to use it on %s", region)
	}

	// Expand the "trigger" set to aws-sdk-go compat []*codecommit.RepositoryTrigger
	triggers, err := expandTriggers(d.Get("trigger").(*schema.Set).List())

	input := &codecommit.PutRepositoryTriggersInput{
		RepositoryName: aws.String(d.Get("repository_name").(string)),
		Triggers:       triggers,
	}

	out, err := conn.PutRepositoryTriggers(input)
	if err != nil {
		return fmt.Errorf("Error creating CodeCommit Repository: %s", err)
	}

	d.SetId(d.Get("repository_name").(string))
	d.Set("configuration_id", *out.ConfigurationId)

	return resourceAwsCodeCommitTriggerRead(d, meta)
}

func resourceAwsCodeCommitTriggerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	input := &codecommit.GetRepositoryTriggersInput{
		RepositoryName: aws.String(d.Id()),
	}

	out, err := conn.GetRepositoryTriggers(input)
	if err != nil {
		return fmt.Errorf("Error reading CodeCommit Repository Trigger: %s", err.Error())
	}

	d.Set("configuration_id", *out.ConfigurationId)

	return nil
}

func resourceAwsCodeCommitTriggerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["destination_arn"].(string))))
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["custom_data"].(string))))
	buf.WriteString(fmt.Sprintf("%s-", expandStringList(m["branches"].(*schema.Set).List())))
	buf.WriteString(fmt.Sprintf("%s-", expandStringList(m["events"].(*schema.Set).List())))

	return hashcode.String(buf.String())
}

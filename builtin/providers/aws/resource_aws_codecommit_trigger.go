package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodeCommitTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeCommitTriggerCreate,
		Read:   resourceAwsCodeCommitTriggerRead,
		Delete: resourceAwsCodeCommitTriggerDelete,

		Schema: map[string]*schema.Schema{
			"repository_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"configuration_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"trigger": &schema.Schema{
				Type:     schema.TypeSet,
				ForceNew: true,
				Required: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"destination_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"custom_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"branches": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"events": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceAwsCodeCommitTriggerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	// Expand the "trigger" set to aws-sdk-go compat []*codecommit.RepositoryTrigger
	triggers := expandAwsCodeCommitTriggers(d.Get("trigger").(*schema.Set).List())

	input := &codecommit.PutRepositoryTriggersInput{
		RepositoryName: aws.String(d.Get("repository_name").(string)),
		Triggers:       triggers,
	}

	resp, err := conn.PutRepositoryTriggers(input)
	if err != nil {
		return fmt.Errorf("Error creating CodeCommit Trigger: %s", err)
	}

	log.Printf("[INFO] Code Commit Trigger Created %s input %s", resp, input)

	d.SetId(d.Get("repository_name").(string))
	d.Set("configuration_id", resp.ConfigurationId)

	return resourceAwsCodeCommitTriggerRead(d, meta)
}

func resourceAwsCodeCommitTriggerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	input := &codecommit.GetRepositoryTriggersInput{
		RepositoryName: aws.String(d.Id()),
	}

	resp, err := conn.GetRepositoryTriggers(input)
	if err != nil {
		return fmt.Errorf("Error reading CodeCommit Trigger: %s", err.Error())
	}

	log.Printf("[DEBUG] CodeCommit Trigger: %s", resp)

	return nil
}

func resourceAwsCodeCommitTriggerDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).codecommitconn

	log.Printf("[DEBUG] Deleting Trigger: %q", d.Id())

	input := &codecommit.PutRepositoryTriggersInput{
		RepositoryName: aws.String(d.Get("repository_name").(string)),
		Triggers:       []*codecommit.RepositoryTrigger{},
	}

	_, err := conn.PutRepositoryTriggers(input)

	if err != nil {
		return err
	}

	return nil
}

func expandAwsCodeCommitTriggers(configured []interface{}) []*codecommit.RepositoryTrigger {
	triggers := make([]*codecommit.RepositoryTrigger, 0, len(configured))
	// Loop over our configured triggers and create
	// an array of aws-sdk-go compatabile objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})
		t := &codecommit.RepositoryTrigger{
			CustomData:     aws.String(data["custom_data"].(string)),
			DestinationArn: aws.String(data["destination_arn"].(string)),
			Name:           aws.String(data["name"].(string)),
		}

		branches := make([]*string, len(data["branches"].([]interface{})))
		for i, vv := range data["branches"].([]interface{}) {
			str := vv.(string)
			branches[i] = aws.String(str)
		}
		t.Branches = branches

		events := make([]*string, len(data["events"].([]interface{})))
		for i, vv := range data["events"].([]interface{}) {
			str := vv.(string)
			events[i] = aws.String(str)
		}
		t.Events = events

		triggers = append(triggers, t)
	}
	return triggers
}

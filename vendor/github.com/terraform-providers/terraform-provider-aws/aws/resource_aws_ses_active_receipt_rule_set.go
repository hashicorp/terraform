package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesActiveReceiptRuleSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesActiveReceiptRuleSetUpdate,
		Update: resourceAwsSesActiveReceiptRuleSetUpdate,
		Read:   resourceAwsSesActiveReceiptRuleSetRead,
		Delete: resourceAwsSesActiveReceiptRuleSetDelete,

		Schema: map[string]*schema.Schema{
			"rule_set_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsSesActiveReceiptRuleSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	ruleSetName := d.Get("rule_set_name").(string)

	createOpts := &ses.SetActiveReceiptRuleSetInput{
		RuleSetName: aws.String(ruleSetName),
	}

	_, err := conn.SetActiveReceiptRuleSet(createOpts)
	if err != nil {
		return fmt.Errorf("Error setting active SES rule set: %s", err)
	}

	d.SetId(ruleSetName)

	return resourceAwsSesActiveReceiptRuleSetRead(d, meta)
}

func resourceAwsSesActiveReceiptRuleSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	describeOpts := &ses.DescribeActiveReceiptRuleSetInput{}

	response, err := conn.DescribeActiveReceiptRuleSet(describeOpts)
	if err != nil {
		return err
	}

	if response.Metadata != nil {
		d.Set("rule_set_name", response.Metadata.Name)
	} else {
		log.Print("[WARN] No active Receipt Rule Set found")
		d.SetId("")
	}

	return nil
}

func resourceAwsSesActiveReceiptRuleSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	deleteOpts := &ses.SetActiveReceiptRuleSetInput{
		RuleSetName: nil,
	}

	_, err := conn.SetActiveReceiptRuleSet(deleteOpts)
	if err != nil {
		return fmt.Errorf("Error deleting active SES rule set: %s", err)
	}

	return nil
}

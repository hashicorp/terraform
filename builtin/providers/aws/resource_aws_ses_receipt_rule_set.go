package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesReceiptRuleSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesReceiptRuleSetCreate,
		Read:   resourceAwsSesReceiptRuleSetRead,
		Delete: resourceAwsSesReceiptRuleSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"rule_set_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSesReceiptRuleSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	ruleSetName := d.Get("rule_set_name").(string)

	createOpts := &ses.CreateReceiptRuleSetInput{
		RuleSetName: aws.String(ruleSetName),
	}

	_, err := conn.CreateReceiptRuleSet(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating SES rule set: %s", err)
	}

	d.SetId(ruleSetName)

	return resourceAwsSesReceiptRuleSetRead(d, meta)
}

func resourceAwsSesReceiptRuleSetRead(d *schema.ResourceData, meta interface{}) error {
	ruleSetExists, err := findRuleSet(d.Id(), nil, meta)

	if !ruleSetExists {
		log.Printf("[WARN] SES Receipt Rule Set (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.Set("rule_set_name", d.Id())

	return nil
}

func resourceAwsSesReceiptRuleSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	log.Printf("[DEBUG] SES Delete Receipt Rule Set: %s", d.Id())
	_, err := conn.DeleteReceiptRuleSet(&ses.DeleteReceiptRuleSetInput{
		RuleSetName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	return nil
}

func findRuleSet(name string, token *string, meta interface{}) (bool, error) {
	conn := meta.(*AWSClient).sesConn

	ruleSetExists := false

	listOpts := &ses.ListReceiptRuleSetsInput{
		NextToken: token,
	}

	response, err := conn.ListReceiptRuleSets(listOpts)
	for _, element := range response.RuleSets {
		if *element.Name == name {
			ruleSetExists = true
		}
	}

	if err != nil && !ruleSetExists && response.NextToken != nil {
		ruleSetExists, err = findRuleSet(name, response.NextToken, meta)
	}

	if err != nil {
		return false, err
	}

	return ruleSetExists, nil
}

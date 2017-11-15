package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSsmParameter() *schema.Resource {
	return &schema.Resource{
		Read: dataAwsSsmParameterRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"value": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataAwsSsmParameterRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading SSM Parameter: %q", name)

	paramInput := &ssm.GetParametersInput{
		Names: []*string{
			aws.String(name),
		},
		WithDecryption: aws.Bool(true),
	}

	resp, err := ssmconn.GetParameters(paramInput)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error describing SSM parameter: {{err}}", err)
	}

	if len(resp.InvalidParameters) > 0 {
		return fmt.Errorf("[ERROR] SSM Parameter %s is invalid", name)
	}

	param := resp.Parameters[0]
	d.SetId(*param.Name)
	d.Set("name", param.Name)
	d.Set("type", param.Type)
	d.Set("value", param.Value)

	return nil
}

package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotPolicyCreate,
		Read:   resourceAwsIotPolicyRead,
		Update: resourceAwsIotPolicyUpdate,
		Delete: resourceAwsIotPolicyDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsIotPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

package aws

import "github.com/hashicorp/terraform/helper/schema"

func resourceAwsConfigServiceInventory() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigServiceInventoryCreate,
		Read:   resourceAwsConfigServiceInventoryRead,
		Update: resourceAwsConfigServiceInventoryUpdate,
		Delete: resourceAwsConfigServiceInventoryDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsConfigServiceInventoryCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsConfigServiceInventoryRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsConfigServiceInventoryUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsConfigServiceInventoryDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

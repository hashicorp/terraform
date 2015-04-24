package azure

import "github.com/hashicorp/terraform/helper/schema"

func resourceAzureNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureNetworkCreate,
		Read:   resourceAzureNetworkRead,
		Update: resourceAzureNetworkUpdate,
		Delete: resourceAzureNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"virtual": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
		},
	}
}

func resourceAzureNetworkCreate(d *schema.ResourceData, meta interface{}) (err error) {
	//mc := meta.(*management.Client)

	return resourceAzureNetworkRead(d, meta)
}

func resourceAzureNetworkRead(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return nil
}

func resourceAzureNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return resourceAzureNetworkRead(d, meta)
}

func resourceAzureNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return nil
}

package azure

import "github.com/hashicorp/terraform/helper/schema"

func resourceAzureDisk() *schema.Resource {
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
		},
	}
}

func resourceAzureDiskCreate(d *schema.ResourceData, meta interface{}) (err error) {
	//mc := meta.(*management.Client)

	return resourceAzureDiskRead(d, meta)
}

func resourceAzureDiskRead(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return nil
}

func resourceAzureDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return resourceAzureDiskRead(d, meta)
}

func resourceAzureDiskDelete(d *schema.ResourceData, meta interface{}) error {
	//mc := meta.(*management.Client)

	return nil
}

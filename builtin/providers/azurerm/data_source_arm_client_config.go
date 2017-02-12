package azurerm

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceArmClientConfig() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmClientConfigRead,

		Schema: map[string]*schema.Schema{
			"client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"subscription_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceArmClientConfigRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)

	d.SetId(time.Now().UTC().String())
	d.Set("client_id", client.clientId)
	d.Set("tenant_id", client.tenantId)
	d.Set("subscription_id", client.subscriptionId)

	return nil
}

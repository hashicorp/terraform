package aws

import (
	"log"

	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnectionAccepter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCPeeringAccepterCreate,
		Read:   resourceAwsVPCPeeringRead,
		Update: resourceAwsVPCPeeringUpdate,
		Delete: resourceAwsVPCPeeringAccepterDelete,

		Schema: map[string]*schema.Schema{
			"vpc_peering_connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Computed: false,
			},
			"auto_accept": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"accept_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"peer_vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"peer_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"peer_region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"accepter":  vpcPeeringConnectionOptionsSchema(),
			"requester": vpcPeeringConnectionOptionsSchema(),
			"tags":      tagsSchema(),
		},
	}
}

func resourceAwsVPCPeeringAccepterCreate(d *schema.ResourceData, meta interface{}) error {
	id := d.Get("vpc_peering_connection_id").(string)
	d.SetId(id)

	if err := resourceAwsVPCPeeringRead(d, meta); err != nil {
		return err
	}
	if d.Id() == "" {
		return fmt.Errorf("VPC Peering Connection %q not found", id)
	}

	return resourceAwsVPCPeeringUpdate(d, meta)
}

func resourceAwsVPCPeeringAccepterDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Will not delete VPC peering connection. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}

package postgresql

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePostgresqlRole() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgresqlRoleCreate,
		Read:   resourcePostgresqlRoleRead,
		Update: resourcePostgresqlRoleUpdate,
		Delete: resourcePostgresqlRoleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"login": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
		},
	}
}

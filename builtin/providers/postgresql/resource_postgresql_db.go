package postgresql

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePostgresqlDb() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgresqlDbCreate,
		Read:   resourcePostgresqlDbRead,
		Update: resourcePostgresqlDbUpdate,
		Delete: resourcePostgresqlDbDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

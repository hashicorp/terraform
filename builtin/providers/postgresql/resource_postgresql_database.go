package postgresql

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePostgresqlDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgresqlDatabaseCreate,
		Read:   resourcePostgresqlDatabaseRead,
		Update: resourcePostgresqlDatabaseUpdate,
		Delete: resourcePostgresqlDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

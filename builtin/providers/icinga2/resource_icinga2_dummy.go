package icinga2

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIcinga2Dummy() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2DummyCreate,
		Read:   resourceIcinga2DummyRead,
		Delete: resourceIcinga2DummyDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "name",
				ForceNew:    true,
			},
		},
	}
}

func resourceIcinga2DummyCreate(d *schema.ResourceData, meta interface{}) error {

	// config 		:= meta.(*Config)
	// name 		:= d.Get("name").(string)

	return nil

}

func resourceIcinga2DummyRead(d *schema.ResourceData, meta interface{}) error {

	// config 		:= meta.(*Config)
	// name 		:= d.Get("name").(string)

	return nil
}

func resourceIcinga2DummyDelete(d *schema.ResourceData, meta interface{}) error {

	// config 		:= meta.(*Config)
	// name 		:= d.Get("name").(string)

	return nil
}

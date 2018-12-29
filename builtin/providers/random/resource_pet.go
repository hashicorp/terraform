package random

import (
	"fmt"
	"strings"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePet() *schema.Resource {
	return &schema.Resource{
		Create: CreatePet,
		Read:   ReadPet,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"keepers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"length": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
				ForceNew: true,
			},

			"prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"separator": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "-",
				ForceNew: true,
			},
		},
	}
}

func CreatePet(d *schema.ResourceData, meta interface{}) error {
	length := d.Get("length").(int)
	separator := d.Get("separator").(string)
	prefix := d.Get("prefix").(string)

	pet := strings.ToLower(petname.Generate(length, separator))

	if prefix != "" {
		pet = fmt.Sprintf("%s%s%s", prefix, separator, pet)
	}

	d.SetId(pet)

	return nil
}

func ReadPet(d *schema.ResourceData, meta interface{}) error {
	return nil
}

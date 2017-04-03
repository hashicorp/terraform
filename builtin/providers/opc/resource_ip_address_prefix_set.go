package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCIPAddressPrefixSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPAddressPrefixSetCreate,
		Read:   resourceOPCIPAddressPrefixSetRead,
		Update: resourceOPCIPAddressPrefixSetUpdate,
		Delete: resourceOPCIPAddressPrefixSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"prefixes": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateIPPrefixCIDR,
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"tags": tagsOptionalSchema(),
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCIPAddressPrefixSetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressPrefixSets()

	input := compute.CreateIPAddressPrefixSetInput{
		Name: d.Get("name").(string),
	}

	prefixes := getStringList(d, "prefixes")
	if len(prefixes) != 0 {
		input.IPAddressPrefixes = prefixes
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateIPAddressPrefixSet(&input)
	if err != nil {
		return fmt.Errorf("Error creating IP Address Prefix Set: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCIPAddressPrefixSetRead(d, meta)
}

func resourceOPCIPAddressPrefixSetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressPrefixSets()

	getInput := compute.GetIPAddressPrefixSetInput{
		Name: d.Id(),
	}
	result, err := client.GetIPAddressPrefixSet(&getInput)
	if err != nil {
		// IP Address Prefix Set does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IP Address Prefix Set %s: %s", d.Id(), err)
	}

	d.Set("name", result.Name)
	d.Set("description", result.Description)
	d.Set("uri", result.Uri)
	if err := setStringList(d, "prefixes", result.IPAddressPrefixes); err != nil {
		return err
	}
	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCIPAddressPrefixSetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressPrefixSets()

	input := compute.UpdateIPAddressPrefixSetInput{
		Name: d.Get("name").(string),
	}

	prefixes := getStringList(d, "prefixes")
	if len(prefixes) != 0 {
		input.IPAddressPrefixes = prefixes
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.UpdateIPAddressPrefixSet(&input)
	if err != nil {
		return fmt.Errorf("Error updating IP Address Prefix Set: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCIPAddressPrefixSetRead(d, meta)
}

func resourceOPCIPAddressPrefixSetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressPrefixSets()
	name := d.Id()

	input := compute.DeleteIPAddressPrefixSetInput{
		Name: name,
	}
	if err := client.DeleteIPAddressPrefixSet(&input); err != nil {
		return fmt.Errorf("Error deleting IP Address Prefix Set: %s", err)
	}
	return nil
}

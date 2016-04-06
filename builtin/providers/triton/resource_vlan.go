package triton

import (
	"errors"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/gosdc/cloudapi"
)

func resourceVLAN() *schema.Resource {
	return &schema.Resource{
		Create: resourceVLANCreate,
		Exists: resourceVLANExists,
		Read:   resourceVLANRead,
		Update: resourceVLANUpdate,
		Delete: resourceVLANDelete,

		Schema: map[string]*schema.Schema{
			"vlan_id": {
				Description: "number between 0-4095 indicating VLAN ID",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeInt,
				ValidateFunc: func(val interface{}, field string) (warn []string, err []error) {
					value := val.(int)
					if value < 0 || value > 4095 {
						err = append(err, errors.New("vlan_id must be between 0 and 4095"))
					}
					return
				},
			},
			"name": {
				Description: "Unique name to identify VLAN",
				Required:    true,
				Type:        schema.TypeString,
			},
			"description": {
				Description: "Optional description of the VLAN",
				Optional:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceVLANCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	vlan, err := client.CreateFabricVLAN(cloudapi.FabricVLAN{
		Id:          int16(d.Get("vlan_id").(int)),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(resourceVLANIDString(vlan.Id))
	return resourceVLANRead(d, meta)
}

func resourceVLANExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*cloudapi.Client)

	id, err := resourceVLANIDInt16(d.Id())
	if err != nil {
		return false, err
	}

	vlan, err := client.GetFabricVLAN(id)

	return vlan != nil && err == nil, err
}

func resourceVLANRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	vlan, err := client.GetFabricVLAN(int16(d.Get("vlan_id").(int)))
	if err != nil {
		return err
	}

	d.SetId(resourceVLANIDString(vlan.Id))
	d.Set("vlan_id", vlan.Id)
	d.Set("name", vlan.Name)
	d.Set("description", vlan.Description)

	return nil
}

func resourceVLANUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	vlan, err := client.UpdateFabricVLAN(cloudapi.FabricVLAN{
		Id:          int16(d.Get("vlan_id").(int)),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(resourceVLANIDString(vlan.Id))
	return resourceVLANRead(d, meta)
}

func resourceVLANDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	id, err := resourceVLANIDInt16(d.Id())
	if err != nil {
		return err
	}

	return client.DeleteFabricVLAN(id)
}

// convenience conversion functions

func resourceVLANIDString(id int16) string {
	return strconv.Itoa(int(id))
}

func resourceVLANIDInt16(id string) (int16, error) {
	result, err := strconv.ParseInt(id, 10, 16)
	if err != nil {
		return 0, err
	}

	return int16(result), nil
}

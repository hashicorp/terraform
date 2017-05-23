package triton

import (
	"context"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/triton-go"
)

func resourceVLAN() *schema.Resource {
	return &schema.Resource{
		Create:   resourceVLANCreate,
		Exists:   resourceVLANExists,
		Read:     resourceVLANRead,
		Update:   resourceVLANUpdate,
		Delete:   resourceVLANDelete,
		Timeouts: fastResourceTimeout,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vlan_id": {
				Description: "Number between 0-4095 indicating VLAN ID",
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
				Description: "Description of the VLAN",
				Optional:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceVLANCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	vlan, err := client.Fabrics().CreateFabricVLAN(context.Background(), &triton.CreateFabricVLANInput{
		ID:          d.Get("vlan_id").(int),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(vlan.ID))
	return resourceVLANRead(d, meta)
}

func resourceVLANExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*triton.Client)

	id, err := resourceVLANIDInt(d.Id())
	if err != nil {
		return false, err
	}

	return resourceExists(client.Fabrics().GetFabricVLAN(context.Background(), &triton.GetFabricVLANInput{
		ID: id,
	}))
}

func resourceVLANRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	id, err := resourceVLANIDInt(d.Id())
	if err != nil {
		return err
	}

	vlan, err := client.Fabrics().GetFabricVLAN(context.Background(), &triton.GetFabricVLANInput{
		ID: id,
	})
	if err != nil {
		return err
	}

	d.Set("vlan_id", vlan.ID)
	d.Set("name", vlan.Name)
	d.Set("description", vlan.Description)

	return nil
}

func resourceVLANUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	vlan, err := client.Fabrics().UpdateFabricVLAN(context.Background(), &triton.UpdateFabricVLANInput{
		ID:          d.Get("vlan_id").(int),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(vlan.ID))
	return resourceVLANRead(d, meta)
}

func resourceVLANDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	id, err := resourceVLANIDInt(d.Id())
	if err != nil {
		return err
	}

	return client.Fabrics().DeleteFabricVLAN(context.Background(), &triton.DeleteFabricVLANInput{
		ID: id,
	})
}

func resourceVLANIDInt(id string) (int, error) {
	result, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		return -1, err
	}

	return int(result), nil
}

package clevercloud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/samber/go-clevercloud-api/clever"
)

func resourceCleverCloudAddon(addonType string, availablePlans []string, availableRegions []string) *schema.Resource {
	return &schema.Resource{
		Create: CreateAddon,
		// Does not expect to update an addons
		//Update: UpdateAddon,
		Delete: DeleteAddon,
		Exists: AddonExists,
		Read:   ReadAddon,

		Schema: map[string]*schema.Schema{

			// SET BY TERRAFORM sub-resource
			"addon_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  addonType,
				ForceNew: true,
			},

			// SET BY USER
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					plan := strings.ToLower(v.(string))
					sort.Strings(availablePlans)
					if i := sort.SearchStrings(availablePlans, plan); i >= len(availablePlans) {
						es = append(es, fmt.Errorf(plan+" plan is not valid for addon "+addonType))
					}
					return
				},
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "eu",
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					region := strings.ToLower(v.(string))
					sort.Strings(availableRegions)
					if i := sort.SearchStrings(availableRegions, region); i >= len(availableRegions) {
						es = append(es, fmt.Errorf(region+" region is not available for addon "+addonType))
					}
					return
				},
			},

			// COMPUTED
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"real_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"price": &schema.Schema{
				Type:     schema.TypeFloat,
				Computed: true,
			},

			"environment": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func CreateAddon(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	addonInput, err := resourceDataToAddon(d)
	if err != nil {
		return err
	}

	addonOutput, err := client.CreateAddon(addonInput)
	if err != nil {
		return err
	}

	env, err := client.GetAddonEnvById(addonOutput.Id)
	if err != nil {
		return err
	}

	return addonToResourceData(addonOutput, env, d)
}

func DeleteAddon(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	err := client.DeleteAddon(d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func AddonExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*clever.Client)

	_, err := client.GetAddonById(d.Id())
	if err != nil {
		if _, ok := err.(clever.NotFoundError); ok {
			err = nil
		}
		return false, err
	}

	return true, nil
}

func ReadAddon(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	addonOutput, err := client.GetAddonById(d.Id())
	if err != nil {
		return err
	}

	env, err := client.GetAddonEnvById(d.Id())
	if err != nil {
		return err
	}

	return addonToResourceData(addonOutput, env, d)
}

func resourceDataToAddon(d *schema.ResourceData) (*clever.AddonInput, error) {
	addonInput := &clever.AddonInput{
		Name:       d.Get("name").(string),
		Plan:       d.Get("plan").(string),
		ProviderId: d.Get("addon_type").(string),
		Region:     d.Get("region").(string),
	}
	return addonInput, nil
}

func addonToResourceData(addonOutput *clever.AddonOutput, environment map[string]string, d *schema.ResourceData) error {
	d.SetId(addonOutput.Id)
	d.Set("id", addonOutput.Id)
	d.Set("real_id", addonOutput.RealId)
	d.Set("name", addonOutput.Name)
	d.Set("region", addonOutput.Region)
	d.Set("price", addonOutput.Plan.Price)
	d.Set("environment", environment)
	return nil
}

package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/affinitygroup"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmAffinityGroup returns the *schema.Resource associated to a
// resource affinity group on Azure.
func resourceAsmAffinityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAsmAffinityGroupCreate,
		Read:   resourceAsmAffinityGroupRead,
		Update: resourceAsmAffinityGroupUpdate,
		Exists: resourceAsmAffinityGroupExists,
		Delete: resourceAsmAffinityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"label": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

// resourceAsmAffinityGroupCreate does all the necessary API calls to
// create an affinity group on Azure.
func resourceAsmAffinityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	affinityGroupClient := meta.(*AzureClient).asmClient.affinityGroupClient

	log.Println("[INFO] Begun creating Azure Affinity Group creation request.")
	name := d.Get("name").(string)
	params := affinitygroup.CreateAffinityGroupParams{
		Name:     name,
		Label:    d.Get("label").(string),
		Location: d.Get("location").(string),
	}

	if desc, ok := d.GetOk("description"); ok {
		params.Description = desc.(string)
	}

	log.Println("[INFO] Sending Affinity Group creation request to Azure.")
	err := affinityGroupClient.CreateAffinityGroup(params)
	if err != nil {
		return fmt.Errorf("Error issuing Azure Affinity Group creation: %s", err)
	}

	d.SetId(name)
	return nil
}

// resourceAsmAffinityGroupRead does all the necessary API calls to
// read the state of the affinity group off Azure.
func resourceAsmAffinityGroupRead(d *schema.ResourceData, meta interface{}) error {
	affinityGroupClient := meta.(*AzureClient).asmClient.affinityGroupClient

	log.Println("[INFO] Issuing Azure Affinity Group list request.")
	affinityGroups, err := affinityGroupClient.ListAffinityGroups()
	if err != nil {
		return fmt.Errorf("Error obtaining Affinity Group list off Azure: %s", err)
	}

	var found bool
	name := d.Get("name").(string)
	for _, group := range affinityGroups.AffinityGroups {
		if group.Name == name {
			found = true
			d.Set("location", group.Location)
			d.Set("label", group.Label)
			d.Set("description", group.Description)
			break
		}
	}

	if !found {
		// it means the affinity group has been deleted in the meantime, so we
		// must stop tracking it:
		d.SetId("")
	}

	return nil
}

// resourceAsmAffinityGroupUpdate does all the necessary API calls to
// update the state of the affinity group on Azure.
func resourceAsmAffinityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	affinityGroupClient := meta.(*AzureClient).asmClient.affinityGroupClient

	name := d.Get("name").(string)
	clabel := d.HasChange("label")
	cdesc := d.HasChange("description")
	if clabel || cdesc {
		log.Println("[INFO] Beginning Affinity Group update process.")
		params := affinitygroup.UpdateAffinityGroupParams{}

		if clabel {
			params.Label = d.Get("label").(string)
		}
		if cdesc {
			params.Description = d.Get("description").(string)
		}

		log.Println("[INFO] Sending Affinity Group update request to Azure.")
		err := affinityGroupClient.UpdateAffinityGroup(name, params)
		if err != nil {
			return fmt.Errorf("Error updating Azure Affinity Group parameters: %s", err)
		}
	}

	return nil
}

// resourceAsmAffinityGroupExists does all the necessary API calls to
// check for the existence of the affinity group on Azure.
func resourceAsmAffinityGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	affinityGroupClient := meta.(*AzureClient).asmClient.affinityGroupClient

	log.Println("[INFO] Issuing Azure Affinity Group get request.")
	name := d.Get("name").(string)
	_, err := affinityGroupClient.GetAffinityGroup(name)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			// it means that the affinity group has been deleted in the
			// meantime, so we must untrack it from the schema:
			d.SetId("")
			return false, nil
		} else {
			return false, fmt.Errorf("Error getting Affinity Group off Azure: %s", err)
		}
	}

	return true, nil
}

// resourceAsmAffinityGroupDelete does all the necessary API calls to
// delete the affinity group off Azure.
func resourceAsmAffinityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	affinityGroupClient := meta.(*AzureClient).asmClient.affinityGroupClient

	log.Println("[INFO] Sending Affinity Group deletion request to Azure.")
	name := d.Get("name").(string)
	err := affinityGroupClient.DeleteAffinityGroup(name)
	if err != nil {
		return fmt.Errorf("Error deleting Azure Affinity Group: %s", err)
	}

	return nil
}

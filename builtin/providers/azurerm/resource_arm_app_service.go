package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/web"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmAppService() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmWebAppCreateUpdate,
		Read:   resourceArmWebAppRead,
		Update: resourceArmWebAppCreateUpdate,
		Delete: resourceArmWebAppDelete,

		Schema: map[string]*schema.Schema{
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"skip_dns_registration": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"skip_custom_domain_verification": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"force_dns_registration": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ttl_in_seconds": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"server_farm_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"always_on": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"location": locationSchema(),
			"tags":     tagsSchema(),
		},
	}
}

func resourceArmWebAppCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	appClient := client.appsClient

	log.Printf("[INFO] preparing arguments for Azure ARM Web App creation.")

	resGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	skipDNSRegistration := d.Get("skip_dns_registration").(bool)
	skipCustomDomainVerification := d.Get("skip_custom_domain_verification").(bool)
	forceDNSRegistration := d.Get("force_dns_registration").(bool)
	ttlInSeconds := d.Get("ttl_in_seconds").(string)
	tags := d.Get("tags").(map[string]interface{})

	siteConfig := web.SiteConfig{}
	if v, ok := d.GetOk("always_on"); ok {
		alwaysOn := v.(bool)
		siteConfig.AlwaysOn = &alwaysOn
	}

	siteProps := web.SiteProperties{
		SiteConfig: &siteConfig,
	}
	if v, ok := d.GetOk("server_farm_id"); ok {
		serverFarmID := v.(string)
		siteProps.ServerFarmID = &serverFarmID
	}

	siteEnvelope := web.Site{
		Location:       &location,
		Tags:           expandTags(tags),
		SiteProperties: &siteProps,
	}

	_, err := appClient.CreateOrUpdate(resGroup, name, siteEnvelope, &skipDNSRegistration, &skipCustomDomainVerification, &forceDNSRegistration, ttlInSeconds, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := appClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Site %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmWebAppRead(d, meta)
}

func resourceArmWebAppRead(d *schema.ResourceData, meta interface{}) error {
	appClient := meta.(*ArmClient).appsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading web app details %s", id)

	resGroup := id.ResourceGroup
	name := id.Path["sites"]

	resp, err := appClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Web App %s: %s", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)

	return nil
}

func resourceArmWebAppDelete(d *schema.ResourceData, meta interface{}) error {
	appClient := meta.(*ArmClient).appsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["sites"]

	log.Printf("[DEBUG] Deleting web app %s: %s", resGroup, name)

	deleteMetrics := true
	deleteEmptyServerFarm := true
	skipDNSRegistration := true

	_, err = appClient.Delete(resGroup, name, &deleteMetrics, &deleteEmptyServerFarm, &skipDNSRegistration)

	return err
}

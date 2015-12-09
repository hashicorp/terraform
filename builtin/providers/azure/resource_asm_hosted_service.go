package azure

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/hostedservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmHostedService returns the schema.Resource associated to an
// Azure hosted service.
func resourceAsmHostedService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAsmHostedServiceCreate,
		Read:   resourceAsmHostedServiceRead,
		Update: resourceAsmHostedServiceUpdate,
		Delete: resourceAsmHostedServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["name"],
			},
			"location": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["location"],
			},
			"ephemeral_contents": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: parameterDescriptions["ephemeral_contents"],
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"reverse_dns_fqdn": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: parameterDescriptions["reverse_dns_fqdn"],
			},
			"label": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Default:     "Made by Terraform.",
				Description: parameterDescriptions["label"],
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: parameterDescriptions["description"],
			},
			"default_certificate_thumbprint": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: parameterDescriptions["default_certificate_thumbprint"],
			},
		},
	}
}

// resourceAsmHostedServiceCreate does all the necessary API calls
// to create a hosted service on Azure.
func resourceAsmHostedServiceCreate(d *schema.ResourceData, meta interface{}) error {
	hostedServiceClient := meta.(*AzureClient).asmClient.hostedServiceClient

	serviceName := d.Get("name").(string)
	location := d.Get("location").(string)
	reverseDNS := d.Get("reverse_dns_fqdn").(string)
	description := d.Get("description").(string)
	label := base64.StdEncoding.EncodeToString([]byte(d.Get("label").(string)))

	err := hostedServiceClient.CreateHostedService(
		hostedservice.CreateHostedServiceParameters{
			ServiceName:    serviceName,
			Location:       location,
			Label:          label,
			Description:    description,
			ReverseDNSFqdn: reverseDNS,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed defining new Azure hosted service: %s", err)
	}

	d.SetId(serviceName)
	return nil
}

// resourceAsmHostedServiceRead does all the necessary API calls
// to read the state of a hosted service from Azure.
func resourceAsmHostedServiceRead(d *schema.ResourceData, meta interface{}) error {
	hostedServiceClient := meta.(*AzureClient).asmClient.hostedServiceClient

	log.Println("[INFO] Querying for hosted service info.")
	serviceName := d.Get("name").(string)
	hostedService, err := hostedServiceClient.GetHostedService(serviceName)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			// it means the hosted service was deleted in the meantime,
			// so we must remove it here:
			d.SetId("")
			return nil
		} else {
			return fmt.Errorf("Failed to get hosted service: %s", err)
		}
	}

	log.Println("[DEBUG] Reading hosted service query result data.")
	d.Set("name", hostedService.ServiceName)
	d.Set("url", hostedService.URL)
	d.Set("location", hostedService.Location)
	d.Set("description", hostedService.Description)
	d.Set("label", hostedService.Label)
	d.Set("status", hostedService.Status)
	d.Set("reverse_dns_fqdn", hostedService.ReverseDNSFqdn)
	d.Set("default_certificate_thumbprint", hostedService.DefaultWinRmCertificateThumbprint)

	return nil
}

// resourceAsmHostedServiceUpdate does all the necessary API calls to
// update some settings of a hosted service on Azure.
func resourceAsmHostedServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: although no-op; this is still required in order for updates to
	// ephemeral_contents to be possible.

	// check if the service still exists:
	return resourceAsmHostedServiceRead(d, meta)
}

// resourceAsmHostedServiceDelete does all the necessary API calls to
// delete a hosted service from Azure.
func resourceAsmHostedServiceDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	hostedServiceClient := azureClient.asmClient.hostedServiceClient

	log.Println("[INFO] Issuing hosted service deletion.")
	serviceName := d.Get("name").(string)
	ephemeral := d.Get("ephemeral_contents").(bool)
	reqID, err := hostedServiceClient.DeleteHostedService(serviceName, ephemeral)
	if err != nil {
		return fmt.Errorf("Failed issuing hosted service deletion request: %s", err)
	}

	log.Println("[DEBUG] Awaiting confirmation on hosted service deletion.")
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error on hosted service deletion: %s", err)
	}

	return nil
}

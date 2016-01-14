package google

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeSslCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSslCertificateCreate,
		Read:   resourceComputeSslCertificateRead,
		Delete: resourceComputeSslCertificateDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"certificate": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"private_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeSslCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the certificate parameter
	cert := &compute.SslCertificate{
		Name:        d.Get("name").(string),
		Certificate: d.Get("certificate").(string),
		PrivateKey:  d.Get("private_key").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		cert.Description = v.(string)
	}

	op, err := config.clientCompute.SslCertificates.Insert(
		config.Project, cert).Do()

	if err != nil {
		return fmt.Errorf("Error creating ssl certificate: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Creating SslCertificate")
	if err != nil {
		return err
	}

	d.SetId(cert.Name)

	return resourceComputeSslCertificateRead(d, meta)
}

func resourceComputeSslCertificateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	cert, err := config.clientCompute.SslCertificates.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing SSL Certificate %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading ssl certificate: %s", err)
	}

	d.Set("self_link", cert.SelfLink)
	d.Set("id", strconv.FormatUint(cert.Id, 10))

	return nil
}

func resourceComputeSslCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	op, err := config.clientCompute.SslCertificates.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting ssl certificate: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting SslCertificate")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

package digitalocean

import (
	"context"
	"fmt"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanCertificateCreate,
		Read:   resourceDigitalOceanCertificateRead,
		Delete: resourceDigitalOceanCertificateDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"private_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"leaf_certificate": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"certificate_chain": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"not_after": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"sha1_fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func buildCertificateRequest(d *schema.ResourceData) (*godo.CertificateRequest, error) {
	req := &godo.CertificateRequest{
		Name:             d.Get("name").(string),
		PrivateKey:       d.Get("private_key").(string),
		LeafCertificate:  d.Get("leaf_certificate").(string),
		CertificateChain: d.Get("certificate_chain").(string),
	}

	return req, nil
}

func resourceDigitalOceanCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Create a Certificate Request")

	certReq, err := buildCertificateRequest(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Certificate Create: %#v", certReq)
	cert, _, err := client.Certificates.Create(context.Background(), certReq)
	if err != nil {
		return fmt.Errorf("Error creating Certificate: %s", err)
	}

	d.SetId(cert.ID)

	return resourceDigitalOceanCertificateRead(d, meta)
}

func resourceDigitalOceanCertificateRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Reading the details of the Certificate %s", d.Id())
	cert, _, err := client.Certificates.Get(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving Certificate: %s", err)
	}

	d.Set("name", cert.Name)
	d.Set("not_after", cert.NotAfter)
	d.Set("sha1_fingerprint", cert.SHA1Fingerprint)

	return nil

}

func resourceDigitalOceanCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting Certificate: %s", d.Id())
	_, err := client.Certificates.Delete(context.Background(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Certificate: %s", err)
	}

	return nil

}

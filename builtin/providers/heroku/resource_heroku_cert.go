package heroku

import (
	"fmt"
	"log"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuCert() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuCertCreate,
		Read:   resourceHerokuCertRead,
		Update: resourceHerokuCertUpdate,
		Delete: resourceHerokuCertDelete,

		Schema: map[string]*schema.Schema{
			"app": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"certificate_chain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"private_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"cname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuCertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app := d.Get("app").(string)
	preprocess := true
	opts := heroku.SSLEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		Preprocess:       &preprocess,
		PrivateKey:       d.Get("private_key").(string)}

	log.Printf("[DEBUG] SSL Certificate create configuration: %#v, %#v", app, opts)
	a, err := client.SSLEndpointCreate(app, opts)
	if err != nil {
		panic(err)
	}

	d.SetId(a.ID)
	log.Printf("[INFO] SSL Certificate ID: %s", d.Id())

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	cert, err := resourceHerokuSSLCertRetrieve(
		d.Get("app").(string), d.Id(), client)
	if err != nil {
		return err
	}

	d.Set("certificate_chain", cert.CertificateChain)
	d.Set("name", cert.Name)
	d.Set("cname", cert.CName)

	return nil
}

func resourceHerokuCertUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app := d.Get("app").(string)

	if d.HasChange("certificate_chain") {
		preprocess := true
		rollback := false
		ad, err := client.SSLEndpointUpdate(
			app, d.Id(), heroku.SSLEndpointUpdateOpts{
				CertificateChain: d.Get("certificate_chain").(*string),
				Preprocess:       &preprocess,
				PrivateKey:       d.Get("private_key").(*string),
				Rollback:         &rollback})
		if err != nil {
			return err
		}

		// Store the new ID
		d.SetId(ad.ID)
	}

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	// Destroy the app
	err := client.SSLEndpointDelete(d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSL Cert: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuSSLCertRetrieve(app string, id string, client *heroku.Service) (*heroku.SSLEndpoint, error) {
	addon, err := client.SSLEndpointInfo(app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
	}

	return addon, nil
}

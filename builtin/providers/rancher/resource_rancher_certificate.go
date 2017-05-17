package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	rancher "github.com/rancher/go-rancher/client"
)

func resourceRancherCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherCertificateCreate,
		Read:   resourceRancherCertificateRead,
		Update: resourceRancherCertificateUpdate,
		Delete: resourceRancherCertificateDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRancherCertificateImport,
		},

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cert": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cert_chain": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"algorithm": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cert_fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expires_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"issued_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"issuer": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_size": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"serial_number": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"subject_alternative_names": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRancherCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO][rancher] Creating Certificate: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	cert := d.Get("cert").(string)
	certChain := d.Get("cert_chain").(string)
	key := d.Get("key").(string)

	certificate := rancher.Certificate{
		Name:        name,
		Description: description,
		Cert:        cert,
		CertChain:   certChain,
		Key:         key,
	}
	newCertificate, err := client.Certificate.Create(&certificate)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    CertificateStateRefreshFunc(client, newCertificate.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry credential (%s) to be created: %s", newCertificate.Id, waitErr)
	}

	d.SetId(newCertificate.Id)
	log.Printf("[INFO] Certificate ID: %s", d.Id())

	return resourceRancherCertificateUpdate(d, meta)
}

func resourceRancherCertificateRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Certificate: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	certificate, err := client.Certificate.ById(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Certificate Name: %s", certificate.Name)

	d.Set("description", certificate.Description)
	d.Set("name", certificate.Name)

	// Computed values
	d.Set("cn", certificate.CN)
	d.Set("algorithm", certificate.Algorithm)
	d.Set("cert_fingerprint", certificate.CertFingerprint)
	d.Set("expires_at", certificate.ExpiresAt)
	d.Set("issued_at", certificate.IssuedAt)
	d.Set("issuer", certificate.Issuer)
	d.Set("key_size", certificate.KeySize)
	d.Set("serial_number", certificate.SerialNumber)
	d.Set("subject_alternative_names", certificate.SubjectAlternativeNames)
	d.Set("version", certificate.Version)

	return nil
}

func resourceRancherCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating Certificate: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	certificate, err := client.Certificate.ById(d.Id())
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	cert := d.Get("cert").(string)
	certChain := d.Get("cert_chain").(string)
	key := d.Get("key").(string)

	data := map[string]interface{}{
		"name":        &name,
		"description": &description,
		"cert":        &cert,
		"cert_chain":  &certChain,
		"key":         &key,
	}

	var newCertificate rancher.Certificate
	if err := client.Update("certificate", &certificate.Resource, data, &newCertificate); err != nil {
		return err
	}

	return resourceRancherCertificateRead(d, meta)
}

func resourceRancherCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Certificate: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	certificate, err := client.Certificate.ById(id)
	if err != nil {
		return err
	}

	if err := client.Certificate.Delete(certificate); err != nil {
		return fmt.Errorf("Error deleting Certificate: %s", err)
	}

	log.Printf("[DEBUG] Waiting for certificate (%s) to be removed", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    CertificateStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for certificate (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

func resourceRancherCertificateImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	envID, resourceID := splitID(d.Id())
	d.SetId(resourceID)
	if envID != "" {
		d.Set("environment_id", envID)
	} else {
		client, err := meta.(*Config).GlobalClient()
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		stack, err := client.Environment.ById(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		d.Set("environment_id", stack.AccountId)
	}
	return []*schema.ResourceData{d}, nil
}

// CertificateStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Certificate.
func CertificateStateRefreshFunc(client *rancher.RancherClient, certificateID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		cert, err := client.Certificate.ById(certificateID)

		if err != nil {
			return nil, "", err
		}

		return cert, cert.State, nil
	}
}

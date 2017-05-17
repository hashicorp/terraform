package google

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeSslCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSslCertificateCreate,
		Read:   resourceComputeSslCertificateRead,
		Delete: resourceComputeSslCertificateDelete,

		Schema: map[string]*schema.Schema{
			"certificate": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://cloud.google.com/compute/docs/reference/latest/sslCertificates#resource
					value := v.(string)
					if len(value) > 63 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 63 characters", k))
					}
					return
				},
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://cloud.google.com/compute/docs/reference/latest/sslCertificates#resource
					// uuid is 26 characters, limit the prefix to 37.
					value := v.(string)
					if len(value) > 37 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 37 characters, name is limited to 63", k))
					}
					return
				},
			},

			"private_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeSslCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	var certName string
	if v, ok := d.GetOk("name"); ok {
		certName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		certName = resource.PrefixedUniqueId(v.(string))
	} else {
		certName = resource.UniqueId()
	}

	// Build the certificate parameter
	cert := &compute.SslCertificate{
		Name:        certName,
		Certificate: d.Get("certificate").(string),
		PrivateKey:  d.Get("private_key").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		cert.Description = v.(string)
	}

	op, err := config.clientCompute.SslCertificates.Insert(
		project, cert).Do()

	if err != nil {
		return fmt.Errorf("Error creating ssl certificate: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Creating SslCertificate")
	if err != nil {
		return err
	}

	d.SetId(cert.Name)

	return resourceComputeSslCertificateRead(d, meta)
}

func resourceComputeSslCertificateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	cert, err := config.clientCompute.SslCertificates.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("SSL Certificate %q", d.Get("name").(string)))
	}

	d.Set("self_link", cert.SelfLink)
	d.Set("id", strconv.FormatUint(cert.Id, 10))

	return nil
}

func resourceComputeSslCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	op, err := config.clientCompute.SslCertificates.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting ssl certificate: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting SslCertificate")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

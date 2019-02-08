package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/securityservices"
)

const (
	minManilaMicroversion   = "2.7"
	minOUManilaMicroversion = "2.44"
)

func resourceSharedFilesystemSecurityServiceV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceSharedFilesystemSecurityServiceV2Create,
		Read:   resourceSharedFilesystemSecurityServiceV2Read,
		Update: resourceSharedFilesystemSecurityServiceV2Update,
		Delete: resourceSharedFilesystemSecurityServiceV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"active_directory", "kerberos", "ldap",
				}, true),
			},

			"dns_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"ou": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"user": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"domain": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"server": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceSharedFilesystemSecurityServiceV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	createOpts := securityservices.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Type:        securityservices.SecurityServiceType(d.Get("type").(string)),
		DNSIP:       d.Get("dns_ip").(string),
		User:        d.Get("user").(string),
		Domain:      d.Get("domain").(string),
		Server:      d.Get("server").(string),
	}

	if v, ok := d.GetOkExists("ou"); ok {
		createOpts.OU = v.(string)

		sfsClient.Microversion = minOUManilaMicroversion
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	createOpts.Password = d.Get("password").(string)
	securityservice, err := securityservices.Create(sfsClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating : %s", err)
	}

	d.SetId(securityservice.ID)

	return resourceSharedFilesystemSecurityServiceV2Read(d, meta)
}

func resourceSharedFilesystemSecurityServiceV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	if _, ok := d.GetOkExists("ou"); ok {
		sfsClient.Microversion = minOUManilaMicroversion
	}

	securityservice, err := securityservices.Get(sfsClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "securityservice")
	}

	// Workaround for resource import
	if securityservice.OU == "" {
		sfsClient.Microversion = minOUManilaMicroversion
		securityserviceOU, err := securityservices.Get(sfsClient, d.Id()).Extract()
		if err == nil {
			d.Set("ou", securityserviceOU.OU)
		}
	}

	nopassword := securityservice
	nopassword.Password = ""
	log.Printf("[DEBUG] Retrieved securityservice %s: %#v", d.Id(), nopassword)

	d.Set("name", securityservice.Name)
	d.Set("description", securityservice.Description)
	d.Set("type", securityservice.Type)
	d.Set("domain", securityservice.Domain)
	d.Set("dns_ip", securityservice.DNSIP)
	d.Set("user", securityservice.User)
	d.Set("server", securityservice.Server)
	// Computed
	d.Set("project_id", securityservice.ProjectID)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceSharedFilesystemSecurityServiceV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	var updateOpts securityservices.UpdateOpts
	// Name should always be sent, otherwise it is vanished by manila backend
	name := d.Get("name").(string)
	updateOpts.Name = &name
	if d.HasChange("description") {
		description := d.Get("description").(string)
		updateOpts.Description = &description
	}
	if d.HasChange("type") {
		updateOpts.Type = d.Get("type").(string)
	}
	if d.HasChange("dns_ip") {
		dnsIP := d.Get("dns_ip").(string)
		updateOpts.DNSIP = &dnsIP
	}
	if d.HasChange("ou") {
		ou := d.Get("ou").(string)
		updateOpts.OU = &ou

		sfsClient.Microversion = minOUManilaMicroversion
	}
	if d.HasChange("user") {
		user := d.Get("user").(string)
		updateOpts.User = &user
	}
	if d.HasChange("domain") {
		domain := d.Get("domain").(string)
		updateOpts.Domain = &domain
	}
	if d.HasChange("server") {
		server := d.Get("server").(string)
		updateOpts.Server = &server
	}

	log.Printf("[DEBUG] Updating securityservice %s with options: %#v", d.Id(), updateOpts)

	if d.HasChange("password") {
		password := d.Get("password").(string)
		updateOpts.Password = &password
	}

	_, err = securityservices.Update(sfsClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Unable to update securityservice %s: %s", d.Id(), err)
	}

	return resourceSharedFilesystemSecurityServiceV2Read(d, meta)
}

func resourceSharedFilesystemSecurityServiceV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	log.Printf("[DEBUG] Attempting to delete securityservice %s", d.Id())
	err = securityservices.Delete(sfsClient, d.Id()).ExtractErr()
	if err != nil {
		return CheckDeleted(d, err, "Error deleting securityservice")
	}

	return nil
}

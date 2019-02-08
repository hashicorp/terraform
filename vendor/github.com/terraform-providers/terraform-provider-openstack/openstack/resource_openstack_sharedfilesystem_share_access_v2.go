package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/errors"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
)

func resourceSharedFilesystemShareAccessV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceSharedFilesystemShareAccessV2Create,
		Read:   resourceSharedFilesystemShareAccessV2Read,
		Delete: resourceSharedFilesystemShareAccessV2Delete,
		Importer: &schema.ResourceImporter{
			State: resourceSharedFilesystemShareAccessV2Import,
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

			"share_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"access_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"ip", "user", "cert",
				}, true),
			},

			"access_to": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"access_level": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"rw", "ro",
				}, true),
			},
		},
	}
}

func resourceSharedFilesystemShareAccessV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	shareID := d.Get("share_id").(string)

	grantOpts := shares.GrantAccessOpts{
		AccessType:  d.Get("access_type").(string),
		AccessTo:    d.Get("access_to").(string),
		AccessLevel: d.Get("access_level").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", grantOpts)

	timeout := d.Timeout(schema.TimeoutCreate)

	log.Printf("[DEBUG] Attempting to grant access")
	var access *shares.AccessRight
	err = resource.Retry(timeout, func() *resource.RetryError {
		access, err = shares.GrantAccess(sfsClient, shareID, grantOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		detailedErr := errors.ErrorDetails{}
		e := errors.ExtractErrorInto(err, &detailedErr)
		if e != nil {
			return fmt.Errorf("Error granting access: %s: %s", err, e)
		}
		for k, msg := range detailedErr {
			return fmt.Errorf("Error granting access: %s (%d): %s", k, msg.Code, msg.Message)
		}
	}

	d.SetId(access.ID)

	pending := []string{"new", "queued_to_apply", "applying"}
	// Wait for access to become active before continuing
	err = waitForSFV2Access(sfsClient, shareID, access.ID, "active", pending, timeout)
	if err != nil {
		return err
	}

	return resourceSharedFilesystemShareAccessV2Read(d, meta)
}

func resourceSharedFilesystemShareAccessV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	shareID := d.Get("share_id").(string)

	access, err := shares.ListAccessRights(sfsClient, shareID).Extract()
	if err != nil {
		return CheckDeleted(d, err, "share_access")
	}

	for _, v := range access {
		if v.ID == d.Id() {
			log.Printf("[DEBUG] Retrieved %s share ACL: %#v", d.Id(), v)

			d.Set("access_type", v.AccessType)
			d.Set("access_to", v.AccessTo)
			d.Set("access_level", v.AccessLevel)
			d.Set("region", GetRegion(d, config))

			return nil
		}
	}

	log.Printf("[DEBUG] Unable to find %s share access", d.Id())
	d.SetId("")

	return nil
}

func resourceSharedFilesystemShareAccessV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	shareID := d.Get("share_id").(string)

	revokeOpts := shares.RevokeAccessOpts{AccessID: d.Id()}

	timeout := d.Timeout(schema.TimeoutDelete)

	log.Printf("[DEBUG] Attempting to revoke access %s", d.Id())
	err = resource.Retry(timeout, func() *resource.RetryError {
		err = shares.RevokeAccess(sfsClient, shareID, revokeOpts).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		e := CheckDeleted(d, err, "")
		if e == nil {
			return nil
		}
		detailedErr := errors.ErrorDetails{}
		e = errors.ExtractErrorInto(err, &detailedErr)
		if e != nil {
			return fmt.Errorf("Error waiting for OpenStack share ACL on %s to be removed: %s: %s", shareID, err, e)
		}
		for k, msg := range detailedErr {
			return fmt.Errorf("Error waiting for OpenStack share ACL on %s to be removed: %s (%d): %s", shareID, k, msg.Code, msg.Message)
		}
	}

	// Wait for access to become deleted before continuing
	pending := []string{"active", "new", "queued_to_deny", "denying"}
	err = waitForSFV2Access(sfsClient, shareID, d.Id(), "denied", pending, timeout)
	if err != nil {
		return err
	}

	return nil
}

func resourceSharedFilesystemShareAccessV2Import(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		err := fmt.Errorf("Invalid format specified for Openstack share ACL. Format must be <share id>/<ACL id>")
		return nil, err
	}

	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return nil, fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaMicroversion

	shareID := parts[0]
	accessID := parts[1]

	access, err := shares.ListAccessRights(sfsClient, shareID).Extract()
	if err != nil {
		return nil, fmt.Errorf("Unable to get %s Openstack share and its ACL's: %s", shareID, err)
	}

	for _, v := range access {
		if v.ID == accessID {
			log.Printf("[DEBUG] Retrieved %s share ACL: %#v", accessID, v)

			d.SetId(accessID)
			d.Set("share_id", shareID)
			d.Set("access_type", v.AccessType)
			d.Set("access_to", v.AccessTo)
			d.Set("access_level", v.AccessLevel)
			d.Set("region", GetRegion(d, config))

			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, fmt.Errorf("[DEBUG] Unable to find %s share access", accessID)
}

// Full list of the share access statuses: https://developer.openstack.org/api-ref/shared-file-system/?expanded=list-services-detail,list-access-rules-detail#list-access-rules
func waitForSFV2Access(sfsClient *gophercloud.ServiceClient, shareID string, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for access %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceSFV2AccessRefreshFunc(sfsClient, shareID, id),
		Timeout:    timeout,
		Delay:      1 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "denied":
				return nil
			default:
				return fmt.Errorf("Error: access %s not found: %s", id, err)
			}
		}
		return fmt.Errorf("Error waiting for access %s to become %s: %s", id, target, err)
	}

	return nil
}

func resourceSFV2AccessRefreshFunc(sfsClient *gophercloud.ServiceClient, shareID string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		access, err := shares.ListAccessRights(sfsClient, shareID).Extract()
		if err != nil {
			return nil, "", err
		}
		for _, v := range access {
			if v.ID == id {
				return v, v.State, nil
			}
		}
		return nil, "", gophercloud.ErrDefault404{}
	}
}

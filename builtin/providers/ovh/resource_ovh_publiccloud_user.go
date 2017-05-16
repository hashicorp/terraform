package ovh

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/ovh/go-ovh/ovh"
)

func resourcePublicCloudUser() *schema.Resource {
	return &schema.Resource{
		Create: resourcePublicCloudUserCreate,
		Read:   resourcePublicCloudUserRead,
		Delete: resourcePublicCloudUserDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_PROJECT_ID", nil),
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"openstack_rc": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourcePublicCloudUserCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	params := &PublicCloudUserCreateOpts{
		ProjectId:   projectId,
		Description: d.Get("description").(string),
	}

	r := &PublicCloudUserResponse{}

	log.Printf("[DEBUG] Will create public cloud user: %s", params)

	// Resource is partial because we will also compute Openstack RC & creds
	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/user", params.ProjectId)

	err := config.OVHClient.Post(endpoint, params, r)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %s:\n\t %q", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for User %s:", r)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"ok"},
		Refresh:    waitForPublicCloudUserActive(config.OVHClient, projectId, strconv.Itoa(r.Id)),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("waiting for user (%s): %s", params, err)
	}
	log.Printf("[DEBUG] Created User %s", r)

	readPublicCloudUser(d, r, true)

	openstackrc := make(map[string]string)
	err = publicCloudUserGetOpenstackRC(projectId, d.Id(), config.OVHClient, openstackrc)
	if err != nil {
		return fmt.Errorf("Creating openstack creds for user %s: %s", d.Id(), err)
	}

	d.Set("openstack_rc", &openstackrc)

	d.Partial(false)

	return nil
}

func resourcePublicCloudUserRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)

	d.Partial(true)
	r := &PublicCloudUserResponse{}

	log.Printf("[DEBUG] Will read public cloud user %s from project: %s", d.Id(), projectId)

	endpoint := fmt.Sprintf("/cloud/project/%s/user/%s", projectId, d.Id())

	err := config.OVHClient.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("calling Get %s:\n\t %q", endpoint, err)
	}

	readPublicCloudUser(d, r, false)

	openstackrc := make(map[string]string)
	err = publicCloudUserGetOpenstackRC(projectId, d.Id(), config.OVHClient, openstackrc)
	if err != nil {
		return fmt.Errorf("Reading openstack creds for user %s: %s", d.Id(), err)
	}

	d.Set("openstack_rc", &openstackrc)
	d.Partial(false)
	log.Printf("[DEBUG] Read Public Cloud User %s", r)
	return nil
}

func resourcePublicCloudUserDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	id := d.Id()

	log.Printf("[DEBUG] Will delete public cloud user %s from project: %s", id, projectId)

	endpoint := fmt.Sprintf("/cloud/project/%s/user/%s", projectId, id)

	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return fmt.Errorf("calling Delete %s:\n\t %q", endpoint, err)
	}

	log.Printf("[DEBUG] Deleting Public Cloud User %s from project %s:", id, projectId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Refresh:    waitForPublicCloudUserDelete(config.OVHClient, projectId, id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Deleting Public Cloud user %s from project %s", id, projectId)
	}
	log.Printf("[DEBUG] Deleted Public Cloud User %s from project %s", id, projectId)

	d.SetId("")

	return nil
}

func publicCloudUserExists(projectId, id string, c *ovh.Client) error {
	r := &PublicCloudUserResponse{}

	log.Printf("[DEBUG] Will read public cloud user for project: %s, id: %s", projectId, id)

	endpoint := fmt.Sprintf("/cloud/project/%s/user/%s", projectId, id)

	err := c.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("calling Get %s:\n\t %q", endpoint, err)
	}
	log.Printf("[DEBUG] Read public cloud user: %s", r)

	return nil
}

var publicCloudUserOSTenantName = regexp.MustCompile("export OS_TENANT_NAME=\"?([[:alnum:]]+)\"?")
var publicCloudUserOSTenantId = regexp.MustCompile("export OS_TENANT_ID=\"??([[:alnum:]]+)\"??")
var publicCloudUserOSAuthURL = regexp.MustCompile("export OS_AUTH_URL=\"??([[:^space:]]+)\"??")
var publicCloudUserOSUsername = regexp.MustCompile("export OS_USERNAME=\"?([[:alnum:]]+)\"?")

func publicCloudUserGetOpenstackRC(projectId, id string, c *ovh.Client, rc map[string]string) error {
	log.Printf("[DEBUG] Will read public cloud user openstack rc for project: %s, id: %s", projectId, id)

	endpoint := fmt.Sprintf("/cloud/project/%s/user/%s/openrc?region=to_be_overriden", projectId, id)

	r := &PublicCloudUserOpenstackRC{}

	err := c.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("calling Get %s:\n\t %q", endpoint, err)
	}

	authURL := publicCloudUserOSAuthURL.FindStringSubmatch(r.Content)
	if authURL == nil {
		return fmt.Errorf("couln't extract OS_AUTH_URL from content: \n\t%s", r.Content)
	}
	tenantName := publicCloudUserOSTenantName.FindStringSubmatch(r.Content)
	if tenantName == nil {
		return fmt.Errorf("couln't extract OS_TENANT_NAME from content: \n\t%s", r.Content)
	}
	tenantId := publicCloudUserOSTenantId.FindStringSubmatch(r.Content)
	if tenantId == nil {
		return fmt.Errorf("couln't extract OS_TENANT_ID from content: \n\t%s", r.Content)
	}
	username := publicCloudUserOSUsername.FindStringSubmatch(r.Content)
	if username == nil {
		return fmt.Errorf("couln't extract OS_USERNAME from content: \n\t%s", r.Content)
	}

	rc["OS_AUTH_URL"] = authURL[1]
	rc["OS_TENANT_ID"] = tenantId[1]
	rc["OS_TENANT_NAME"] = tenantName[1]
	rc["OS_USERNAME"] = username[1]

	return nil
}

func readPublicCloudUser(d *schema.ResourceData, r *PublicCloudUserResponse, setPassword bool) {
	d.Set("description", r.Description)
	d.Set("status", r.Status)
	d.Set("creation_date", r.CreationDate)
	d.Set("username", r.Username)
	if setPassword {
		d.Set("password", r.Password)
	}
	d.SetId(strconv.Itoa(r.Id))
}

func waitForPublicCloudUserActive(c *ovh.Client, projectId, PublicCloudUserId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudUserResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/user/%s", projectId, PublicCloudUserId)
		err := c.Get(endpoint, r)
		if err != nil {
			return r, "", err
		}

		log.Printf("[DEBUG] Pending User: %s", r)
		return r, r.Status, nil
	}
}

func waitForPublicCloudUserDelete(c *ovh.Client, projectId, PublicCloudUserId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudUserResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/user/%s", projectId, PublicCloudUserId)
		err := c.Get(endpoint, r)
		if err != nil {
			if err.(*ovh.APIError).Code == 404 {
				log.Printf("[DEBUG] user id %s on project %s deleted", PublicCloudUserId, projectId)
				return r, "deleted", nil
			} else {
				return r, "", err
			}
		}

		log.Printf("[DEBUG] Pending User: %s", r)
		return r, r.Status, nil
	}
}

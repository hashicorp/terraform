package google

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
)

// resourceGoogleProject returns a *schema.Resource that allows a customer
// to declare a Google Cloud Project resource.
func resourceGoogleProject() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Create: resourceGoogleProjectCreate,
		Read:   resourceGoogleProjectRead,
		Update: resourceGoogleProjectUpdate,
		Delete: resourceGoogleProjectDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		MigrateState: resourceGoogleProjectMigrateState,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Removed:  "The id field has been removed. Use project_id instead.",
			},
			"project_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"skip_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"org_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Removed:  "Use the 'google_project_iam_policy' resource to define policies for a Google Project",
			},
			"policy_etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Removed:  "Use the the 'google_project_iam_policy' resource to define policies for a Google Project",
			},
			"number": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"billing_account": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceGoogleProjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	var pid string
	var err error
	pid = d.Get("project_id").(string)

	log.Printf("[DEBUG]: Creating new project %q", pid)
	project := &cloudresourcemanager.Project{
		ProjectId: pid,
		Name:      d.Get("name").(string),
		Parent: &cloudresourcemanager.ResourceId{
			Id:   d.Get("org_id").(string),
			Type: "organization",
		},
	}

	op, err := config.clientResourceManager.Projects.Create(project).Do()
	if err != nil {
		return fmt.Errorf("Error creating project %s (%s): %s.", project.ProjectId, project.Name, err)
	}

	d.SetId(pid)

	// Wait for the operation to complete
	waitErr := resourceManagerOperationWait(config, op, "project to create")
	if waitErr != nil {
		// The resource wasn't actually created
		d.SetId("")
		return waitErr
	}

	// Set the billing account
	if v, ok := d.GetOk("billing_account"); ok {
		name := v.(string)
		ba := cloudbilling.ProjectBillingInfo{
			BillingAccountName: "billingAccounts/" + name,
		}
		_, err = config.clientBilling.Projects.UpdateBillingInfo(prefixedProject(pid), &ba).Do()
		if err != nil {
			d.Set("billing_account", "")
			if _err, ok := err.(*googleapi.Error); ok {
				return fmt.Errorf("Error setting billing account %q for project %q: %v", name, prefixedProject(pid), _err)
			}
			return fmt.Errorf("Error setting billing account %q for project %q: %v", name, prefixedProject(pid), err)
		}
	}

	return resourceGoogleProjectRead(d, meta)
}

func resourceGoogleProjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	pid := d.Id()

	// Read the project
	p, err := config.clientResourceManager.Projects.Get(pid).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Project %q", pid))
	}

	d.Set("project_id", pid)
	d.Set("number", strconv.FormatInt(int64(p.ProjectNumber), 10))
	d.Set("name", p.Name)

	if p.Parent != nil {
		d.Set("org_id", p.Parent.Id)
	}

	// Read the billing account
	ba, err := config.clientBilling.Projects.GetBillingInfo(prefixedProject(pid)).Do()
	if err != nil {
		return fmt.Errorf("Error reading billing account for project %q: %v", prefixedProject(pid), err)
	}
	if ba.BillingAccountName != "" {
		// BillingAccountName is contains the resource name of the billing account
		// associated with the project, if any. For example,
		// `billingAccounts/012345-567890-ABCDEF`. We care about the ID and not
		// the `billingAccounts/` prefix, so we need to remove that. If the
		// prefix ever changes, we'll validate to make sure it's something we
		// recognize.
		_ba := strings.TrimPrefix(ba.BillingAccountName, "billingAccounts/")
		if ba.BillingAccountName == _ba {
			return fmt.Errorf("Error parsing billing account for project %q. Expected value to begin with 'billingAccounts/' but got %s", prefixedProject(pid), ba.BillingAccountName)
		}
		d.Set("billing_account", _ba)
	}
	return nil
}

func prefixedProject(pid string) string {
	return "projects/" + pid
}

func resourceGoogleProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	pid := d.Id()

	// Read the project
	// we need the project even though refresh has already been called
	// because the API doesn't support patch, so we need the actual object
	p, err := config.clientResourceManager.Projects.Get(pid).Do()
	if err != nil {
		if v, ok := err.(*googleapi.Error); ok && v.Code == http.StatusNotFound {
			return fmt.Errorf("Project %q does not exist.", pid)
		}
		return fmt.Errorf("Error checking project %q: %s", pid, err)
	}

	// Project name has changed
	if ok := d.HasChange("name"); ok {
		p.Name = d.Get("name").(string)
		// Do update on project
		p, err = config.clientResourceManager.Projects.Update(p.ProjectId, p).Do()
		if err != nil {
			return fmt.Errorf("Error updating project %q: %s", p.Name, err)
		}
	}

	// Billing account has changed
	if ok := d.HasChange("billing_account"); ok {
		name := d.Get("billing_account").(string)
		ba := cloudbilling.ProjectBillingInfo{
			BillingAccountName: "billingAccounts/" + name,
		}
		_, err = config.clientBilling.Projects.UpdateBillingInfo(prefixedProject(pid), &ba).Do()
		if err != nil {
			d.Set("billing_account", "")
			if _err, ok := err.(*googleapi.Error); ok {
				return fmt.Errorf("Error updating billing account %q for project %q: %v", name, prefixedProject(pid), _err)
			}
			return fmt.Errorf("Error updating billing account %q for project %q: %v", name, prefixedProject(pid), err)
		}
	}
	return nil
}

func resourceGoogleProjectDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	// Only delete projects if skip_delete isn't set
	if !d.Get("skip_delete").(bool) {
		pid := d.Id()
		_, err := config.clientResourceManager.Projects.Delete(pid).Do()
		if err != nil {
			return fmt.Errorf("Error deleting project %q: %s", pid, err)
		}
	}
	d.SetId("")
	return nil
}

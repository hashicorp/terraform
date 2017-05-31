package github

import (
	"context"
	"log"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubIssueLabel() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubIssueLabelCreateOrUpdate,
		Read:   resourceGithubIssueLabelRead,
		Update: resourceGithubIssueLabelCreateOrUpdate,
		Delete: resourceGithubIssueLabelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"color": {
				Type:     schema.TypeString,
				Required: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceGithubIssueLabelCreateOrUpdate idempotently creates or updates an
// issue label. Issue labels are keyed off of their "name", so pre-existing
// issue labels result in a 422 HTTP error if they exist outside of Terraform.
// Normally this would not be an issue, except new repositories are created with
// a "default" set of labels, and those labels easily conflict with custom ones.
//
// This function will first check if the label exists, and then issue an update,
// otherwise it will create. This is also advantageous in that we get to use the
// same function for two schema funcs.

func resourceGithubIssueLabelCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	o := meta.(*Organization).name
	r := d.Get("repository").(string)
	n := d.Get("name").(string)
	c := d.Get("color").(string)

	label := &github.Label{
		Name:  &n,
		Color: &c,
	}

	log.Printf("[DEBUG] Querying label existence %s/%s (%s)", o, r, n)
	existing, _, _ := client.Issues.GetLabel(context.TODO(), o, r, n)

	if existing != nil {
		log.Printf("[DEBUG] Updating label: %s/%s (%s: %s)", o, r, n, c)

		// Pull out the original name. If we already have a resource, this is the
		// parsed ID. If not, it's the value given to the resource.
		var oname string
		if d.Id() == "" {
			oname = n
		} else {
			_, oname = parseTwoPartID(d.Id())
		}

		_, _, err := client.Issues.EditLabel(context.TODO(), o, r, oname, label)
		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] Creating label: %s/%s (%s: %s)", o, r, n, c)
		_, resp, err := client.Issues.CreateLabel(context.TODO(), o, r, label)
		if resp != nil {
			log.Printf("[DEBUG] Response from creating label: %s", *resp)
		}
		if err != nil {
			return err
		}
	}

	d.SetId(buildTwoPartID(&r, &n))

	return resourceGithubIssueLabelRead(d, meta)
}

func resourceGithubIssueLabelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, n := parseTwoPartID(d.Id())

	log.Printf("[DEBUG] Reading label: %s/%s", r, n)
	githubLabel, _, err := client.Issues.GetLabel(context.TODO(), meta.(*Organization).name, r, n)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("repository", r)
	d.Set("name", n)
	d.Set("color", githubLabel.Color)
	d.Set("url", githubLabel.URL)

	return nil
}

func resourceGithubIssueLabelDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)
	n := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting label: %s/%s", r, n)
	_, err := client.Issues.DeleteLabel(context.TODO(), meta.(*Organization).name, r, n)
	return err
}

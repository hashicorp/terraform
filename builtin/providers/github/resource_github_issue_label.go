package github

import (
	"context"
	"log"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubIssueLabel() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubIssueLabelCreate,
		Read:   resourceGithubIssueLabelRead,
		Update: resourceGithubIssueLabelUpdate,
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

func resourceGithubIssueLabelCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)
	n := d.Get("name").(string)
	c := d.Get("color").(string)
	label := github.Label{
		Name:  &n,
		Color: &c,
	}

	log.Printf("[DEBUG] Creating label: %#v", label)
	_, resp, err := client.Issues.CreateLabel(context.TODO(), meta.(*Organization).name, r, &label)
	log.Printf("[DEBUG] Response from creating label: %s", *resp)
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &n))

	return resourceGithubIssueLabelRead(d, meta)
}

func resourceGithubIssueLabelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, n := parseTwoPartID(d.Id())

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

func resourceGithubIssueLabelUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)
	n := d.Get("name").(string)
	c := d.Get("color").(string)

	_, originalName := parseTwoPartID(d.Id())
	_, _, err := client.Issues.EditLabel(context.TODO(), meta.(*Organization).name, r, originalName, &github.Label{
		Name:  &n,
		Color: &c,
	})
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &n))

	return resourceGithubIssueLabelRead(d, meta)
}

func resourceGithubIssueLabelDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)
	n := d.Get("name").(string)

	_, err := client.Issues.DeleteLabel(context.TODO(), meta.(*Organization).name, r, n)
	return err
}

package github

import (
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
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"color": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"url": &schema.Schema{
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

	_, _, err := client.Issues.CreateLabel(meta.(*Organization).name, r, &github.Label{
		Name:  &n,
		Color: &c,
	})
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &n))

	return resourceGithubIssueLabelRead(d, meta)
}

func resourceGithubIssueLabelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, n := parseTwoPartID(d.Id())

	githubLabel, _, err := client.Issues.GetLabel(meta.(*Organization).name, r, n)
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
	_, _, err := client.Issues.EditLabel(meta.(*Organization).name, r, originalName, &github.Label{
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

	_, err := client.Issues.DeleteLabel(meta.(*Organization).name, r, n)
	return err
}

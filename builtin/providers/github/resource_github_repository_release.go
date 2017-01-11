package github

import (
	"fmt"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepositoryRelease() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubRepositoryReleaseCreate,
		Read:   resourceGithubRepositoryReleaseRead,
		Update: resourceGithubRepositoryReleaseUpdate,
		Delete: resourceGithubRepositoryReleaseDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repo": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tag": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"committish": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"body": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "",
				Optional: true,
			},
			"draft": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},
		},
	}
}

func resourceGithubRepositoryReleaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	r := d.Get("repo").(string)
	t := d.Get("tag").(string)
	c := d.Get("committish").(string)
	n := d.Get("name").(string)
	b := d.Get("body").(string)
	dr := d.Get("draft").(bool)

	rel := &github.RepositoryRelease{
		TagName:         &t,
		TargetCommitish: &c,
		Name:            &n,
		Body:            &b,
		Draft:           &dr,
	}

	release, _, err := client.Repositories.CreateRelease(meta.(*Organization).name, r, rel)
	if err != nil {
		return fmt.Errorf("%v: from GitHub", err)
	}

	d.SetId(strconv.Itoa(*release.ID))
	d.Set("url", *release.URL)

	return nil
}

func resourceGithubRepositoryReleaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	r := d.Get("repo").(string)
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	release, _, err := client.Repositories.GetRelease(meta.(*Organization).name, r, id)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("owner", release.Author)
	d.Set("tag", release.TagName)
	d.Set("committish", release.TargetCommitish)
	d.Set("name", release.Name)
	d.Set("body", release.Body)
	d.Set("draft", release.Draft)

	return nil
}

func resourceGithubRepositoryReleaseUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	c := d.Get("committish").(string)
	t := d.Get("tag").(string)
	r := d.Get("repo").(string)
	n := d.Get("name").(string)
	b := d.Get("body").(string)
	dr := d.Get("draft").(bool)

	rel := &github.RepositoryRelease{
		TagName:         &t,
		Name:            &n,
		Body:            &b,
		TargetCommitish: &c,
		Draft:           &dr,
	}

	release, _, err := client.Repositories.EditRelease(meta.(*Organization).name, r, id, rel)
	if err != nil {
		return fmt.Errorf("%v: from GitHub", err)
	}

	d.SetId(strconv.Itoa(*release.ID))
	d.Set("url", *release.URL)

	return nil
}

func resourceGithubRepositoryReleaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	r := d.Get("repo").(string)

	_, err = client.Repositories.DeleteRelease(meta.(*Organization).name, r, id)

	return err
}

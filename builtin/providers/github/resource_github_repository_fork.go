package github

import (
	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepositoryFork() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubRepositoryForkCreate,
		Read:   resourceGithubRepositoryForkRead,

		Delete: resourceGithubRepositoryForkDelete,

		Schema: map[string]*schema.Schema{
			// owner specifies the owner of the repository
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// organization specifies the optional parameter to fork the
			// repository into the organization
			"organization": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceGithubRepositoryForkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients)
	o := d.Get("owner").(string)
	r := d.Get("repository").(string)
	p := d.Get("organization").(string)

	if err := client.Fork(o, r, p); err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &o))

	return resourceGithubRepositoryForkRead(d, meta)
}

func resourceGithubRepositoryForkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients).UserClient
	r := d.Get("repository").(string)

	repositories, _, err := client.Repositories.ListForks(meta.(*Clients).OrgName, r, &github.RepositoryListForksOptions{})
	if err != nil {
		return err
	}

	for _, repo := range repositories {
		if *repo.Name == "" {
			continue
		}

		d.Set("repository", *repo.Name)
	}

	return nil
}

func resourceGithubRepositoryForkDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients).UserClient
	o := d.Get("owner").(string)
	r := d.Get("repository").(string)

	_, err := client.Repositories.Delete(o, r)

	return err
}

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
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// option specifies the optional parameters to the
			// The organization to fork the repositories into.
			"options": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

// interfaceToStringSlice function is created to support
// forking multi repository at a time
func interfaceToStringSlice(s interface{}) []string {
	slice, ok := s.([]interface{})
	if !ok {
		return nil
	}

	sslice := make([]string, len(slice))
	for i := range slice {
		sslice[i] = slice[i].(string)
	}

	return sslice
}

func resourceGithubRepositoryForkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	u := d.Get("username").(string)
	r := d.Get("repository").(string)
	p := d.Get("option").(string)

	var opt *github.RepositoryCreateForkOptions
	if p != "" {
		opt = &github.RepositoryCreateForkOptions{Organization: p}
	}

	_, _, err := client.Repositories.CreateFork(meta.(*Organization).name, r, opt)
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &u))

	return resourceGithubRepositoryForkRead(d, meta)
}

func resourceGithubRepositoryForkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)

	repositories, _, err := client.Repositories.ListForks(meta.(*Organization).name, r, &github.RepositoryListForksOptions{})
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
	client := meta.(*Organization).client
	u := d.Get("username").(string)
	r := d.Get("repository").(string)

	_, err := client.Repositories.Delete(u, r)

	return err
}

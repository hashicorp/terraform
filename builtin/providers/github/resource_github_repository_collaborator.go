package github

import (
	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepositoryCollaborator() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubRepositoryCollaboratorCreate,
		Read:   resourceGithubRepositoryCollaboratorRead,
		// editing repository collaborators are not supported by github api so forcing new on any changes
		Delete: resourceGithubRepositoryCollaboratorDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
			"permission": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "push",
				ValidateFunc: validateValueFunc([]string{"pull", "push", "admin"}),
			},
		},
	}
}

func resourceGithubRepositoryCollaboratorCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	u := d.Get("username").(string)
	r := d.Get("repository").(string)
	p := d.Get("permission").(string)

	_, err := client.Repositories.AddCollaborator(meta.(*Organization).name, r, u,
		&github.RepositoryAddCollaboratorOptions{Permission: p})

	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&r, &u))

	return resourceGithubRepositoryCollaboratorRead(d, meta)
}

func resourceGithubRepositoryCollaboratorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, u := parseTwoPartID(d.Id())

	isCollaborator, _, err := client.Repositories.IsCollaborator(meta.(*Organization).name, r, u)

	if !isCollaborator || err != nil {
		d.SetId("")
		return nil
	}

	collaborators, _, err := client.Repositories.ListCollaborators(meta.(*Organization).name, r,
		&github.ListOptions{})

	if err != nil {
		return err
	}

	for _, c := range collaborators {
		if *c.Login == u {
			permName, err := getRepoPermission(c.Permissions)

			if err != nil {
				return err
			}

			d.Set("repository", r)
			d.Set("username", u)
			d.Set("permission", permName)

			return nil
		}
	}

	return nil
}

func resourceGithubRepositoryCollaboratorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	u := d.Get("username").(string)
	r := d.Get("repository").(string)

	_, err := client.Repositories.RemoveCollaborator(meta.(*Organization).name, r, u)

	return err
}

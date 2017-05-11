package github

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubTeamRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubTeamRepositoryCreate,
		Read:   resourceGithubTeamRepositoryRead,
		Update: resourceGithubTeamRepositoryUpdate,
		Delete: resourceGithubTeamRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permission": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "pull",
				ValidateFunc: validateValueFunc([]string{"pull", "push", "admin"}),
			},
			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGithubTeamRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).Client()
	t := d.Get("team_id").(string)
	r := d.Get("repository").(string)
	p := d.Get("permission").(string)

	_, err := client.Organizations.AddTeamRepo(context.TODO(), toGithubID(t), meta.(*Organization).name, r,
		&github.OrganizationAddTeamRepoOptions{Permission: p})

	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&t, &r))

	return resourceGithubTeamRepositoryRead(d, meta)
}

func resourceGithubTeamRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).Client()
	t, r := parseTwoPartID(d.Id())

	client.Transport.etag = d.Get("etag").(string)
	repo, rsp, repoErr := client.Organizations.IsTeamRepo(context.TODO(), toGithubID(t), meta.(*Organization).name, r)
	if rsp.StatusCode == 304 {
		// no changes
		return nil
	}

	if repoErr != nil {
		d.SetId("")
		return nil
	}

	repositoryName := repo.Name

	d.Set("team_id", t)
	d.Set("repository", repositoryName)
	d.Set("etag", rsp.Header.Get("ETag"))

	permName, permErr := getRepoPermission(repo.Permissions)

	if permErr != nil {
		return permErr
	}

	d.Set("permission", permName)

	return nil
}

func resourceGithubTeamRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).Client()
	t := d.Get("team_id").(string)
	r := d.Get("repository").(string)
	p := d.Get("permission").(string)

	// the go-github library's AddTeamRepo method uses the add/update endpoint from Github API
	_, err := client.Organizations.AddTeamRepo(context.TODO(), toGithubID(t), meta.(*Organization).name, r,
		&github.OrganizationAddTeamRepoOptions{Permission: p})

	if err != nil {
		return err
	}
	d.SetId(buildTwoPartID(&t, &r))

	return resourceGithubTeamRepositoryRead(d, meta)
}

func resourceGithubTeamRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).Client()
	t := d.Get("team_id").(string)
	r := d.Get("repository").(string)

	_, err := client.Organizations.RemoveTeamRepo(context.TODO(), toGithubID(t), meta.(*Organization).name, r)

	return err
}

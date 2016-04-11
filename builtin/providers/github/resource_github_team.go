package github

import (
	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubTeam() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubTeamCreate,
		Read:   resourceGithubTeamRead,
		Update: resourceGithubTeamUpdate,
		Delete: resourceGithubTeamDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"privacy": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "secret",
				ValidateFunc: validateValueFunc([]string{"secret", "closed"}),
			},
		},
	}
}

func resourceGithubTeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	n := d.Get("name").(string)
	desc := d.Get("description").(string)
	p := d.Get("privacy").(string)
	githubTeam, _, err := client.Organizations.CreateTeam(meta.(*Organization).name, &github.Team{
		Name:        &n,
		Description: &desc,
		Privacy:     &p,
	})
	if err != nil {
		return err
	}
	d.SetId(fromGithubID(githubTeam.ID))
	return resourceGithubTeamRead(d, meta)
}

func resourceGithubTeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	team, err := getGithubTeam(d, client)
	if err != nil {
		d.SetId("")
		return nil
	}
	d.Set("description", team.Description)
	d.Set("name", team.Name)
	d.Set("privacy", team.Privacy)
	return nil
}

func resourceGithubTeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	team, err := getGithubTeam(d, client)

	if err != nil {
		d.SetId("")
		return nil
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	privacy := d.Get("privacy").(string)
	team.Description = &description
	team.Name = &name
	team.Privacy = &privacy

	team, _, err = client.Organizations.EditTeam(*team.ID, team)
	if err != nil {
		return err
	}
	d.SetId(fromGithubID(team.ID))
	return resourceGithubTeamRead(d, meta)
}

func resourceGithubTeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	id := toGithubID(d.Id())
	_, err := client.Organizations.DeleteTeam(id)
	return err
}

func getGithubTeam(d *schema.ResourceData, github *github.Client) (*github.Team, error) {
	id := toGithubID(d.Id())
	team, _, err := github.Organizations.GetTeam(id)
	return team, err
}

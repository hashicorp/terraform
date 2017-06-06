package github

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGithubTeam() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGithubTeamRead,

		Schema: map[string]*schema.Schema{
			"slug": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"privacy": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"permission": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceGithubTeamRead(d *schema.ResourceData, meta interface{}) error {
	slug := d.Get("slug").(string)
	log.Printf("[INFO] Refreshing Gitub Team: %s", slug)

	client := meta.(*Organization).client

	team, err := getGithubTeamBySlug(client, meta.(*Organization).name, slug)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(*team.ID))
	d.Set("name", *team.Name)
	d.Set("description", *team.Description)
	d.Set("privacy", *team.Privacy)
	d.Set("permission", *team.Permission)

	return nil
}

func getGithubTeamBySlug(client *github.Client, org string, slug string) (team *github.Team, err error) {
	opt := &github.ListOptions{PerPage: 10}
	for {
		teams, resp, err := client.Organizations.ListTeams(context.TODO(), org, opt)
		if err != nil {
			return team, err
		}

		for _, t := range teams {
			if *t.Slug == slug {
				return t, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return team, fmt.Errorf("Could not find team with slug: %s", slug)
}

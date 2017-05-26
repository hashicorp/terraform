package github

import (
	"context"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubTeamMembership() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubTeamMembershipCreate,
		Read:   resourceGithubTeamMembershipRead,
		// editing team memberships are not supported by github api so forcing new on any changes
		Delete: resourceGithubTeamMembershipDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"username": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "member",
				ValidateFunc: validateValueFunc([]string{"member", "maintainer"}),
			},
		},
	}
}

func resourceGithubTeamMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	t := d.Get("team_id").(string)
	n := d.Get("username").(string)
	r := d.Get("role").(string)

	_, _, err := client.Organizations.AddTeamMembership(context.TODO(), toGithubID(t), n,
		&github.OrganizationAddTeamMembershipOptions{Role: r})

	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(&t, &n))

	return resourceGithubTeamMembershipRead(d, meta)
}

func resourceGithubTeamMembershipRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	t, n := parseTwoPartID(d.Id())

	membership, _, err := client.Organizations.GetTeamMembership(context.TODO(), toGithubID(t), n)

	if err != nil {
		d.SetId("")
		return nil
	}
	team, user := getTeamAndUserFromURL(membership.URL)

	d.Set("username", user)
	d.Set("role", membership.Role)
	d.Set("team_id", team)
	return nil
}

func resourceGithubTeamMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	t := d.Get("team_id").(string)
	n := d.Get("username").(string)

	_, err := client.Organizations.RemoveTeamMembership(context.TODO(), toGithubID(t), n)

	return err
}

func getTeamAndUserFromURL(url *string) (string, string) {
	var team, user string

	urlSlice := strings.Split(*url, "/")
	for v := range urlSlice {
		if urlSlice[v] == "teams" {
			team = urlSlice[v+1]
		}
		if urlSlice[v] == "memberships" {
			user = urlSlice[v+1]
		}
	}
	return team, user
}

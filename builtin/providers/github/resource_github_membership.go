package github

import (
	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubMembership() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubMembershipCreate,
		Read:   resourceGithubMembershipRead,
		Update: resourceGithubMembershipUpdate,
		Delete: resourceGithubMembershipDelete,

		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateValueFunc([]string{"member", "admin"}),
				Default:      "member",
			},
		},
	}
}

func resourceGithubMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	n := d.Get("username").(string)
	r := d.Get("role").(string)

	membership, _, err := client.Organizations.EditOrgMembership(n, meta.(*Organization).name,
		&github.Membership{Role: &r})
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(membership.Organization.Login, membership.User.Login))

	return resourceGithubMembershipRead(d, meta)
}

func resourceGithubMembershipRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	membership, _, err := client.Organizations.GetOrgMembership(d.Get("username").(string), meta.(*Organization).name)
	if err != nil {
		d.SetId("")
		return nil
	}
	username := membership.User.Login
	roleName := membership.Role

	d.Set("username", *username)
	d.Set("role", *roleName)
	return nil
}

func resourceGithubMembershipUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	n := d.Get("username").(string)
	r := d.Get("role").(string)

	_, _, err := client.Organizations.EditOrgMembership(n, meta.(*Organization).name, &github.Membership{
		Role: &r,
	})
	if err != nil {
		return err
	}
	return nil
}

func resourceGithubMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	n := d.Get("username").(string)

	_, err := client.Organizations.RemoveOrgMembership(n, meta.(*Organization).name)

	return err
}

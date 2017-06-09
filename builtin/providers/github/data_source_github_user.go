package github

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGithubUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGithubUserRead,

		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"login": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"avatar_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"gravatar_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"site_admin": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"company": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"blog": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"bio": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_repos": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"public_gists": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"followers": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"following": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"created_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceGithubUserRead(d *schema.ResourceData, meta interface{}) error {
	username := d.Get("username").(string)
	log.Printf("[INFO] Refreshing Gitub User: %s", username)

	client := meta.(*Organization).client

	user, _, err := client.Users.Get(context.TODO(), username)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(user.GetID()))
	d.Set("login", user.GetLogin())
	d.Set("avatar_url", user.GetAvatarURL())
	d.Set("gravatar_id", user.GetGravatarID())
	d.Set("site_admin", user.GetSiteAdmin())
	d.Set("company", user.GetCompany())
	d.Set("blog", user.GetBlog())
	d.Set("location", user.GetLocation())
	d.Set("name", user.GetName())
	d.Set("email", user.GetEmail())
	d.Set("bio", user.GetBio())
	d.Set("public_repos", user.GetPublicRepos())
	d.Set("public_gists", user.GetPublicGists())
	d.Set("followers", user.GetFollowers())
	d.Set("following", user.GetFollowing())
	d.Set("created_at", user.GetCreatedAt())
	d.Set("updated_at", user.GetUpdatedAt())

	return nil
}

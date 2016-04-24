package github

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_TOKEN", nil),
				Description: descriptions["token"],
			},
			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_ORGANIZATION", nil),
				Description: descriptions["organization"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"github_team":            resourceGithubTeam(),
			"github_team_membership": resourceGithubTeamMembership(),
			"github_team_repository": resourceGithubTeamRepository(),
			"github_membership":      resourceGithubMembership(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"token": "The OAuth token used to connect to GitHub.",

		"organization": "The GitHub organization name to manage.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Token:        d.Get("token").(string),
		Organization: d.Get("organization").(string),
	}

	return config.Client()
}

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
				Optional:    true,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_TOKEN", nil),
				Description: descriptions["token"],
			},
			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_ORGANIZATION", nil),
				Description: descriptions["organization"],
			},
			"base_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_BASE_URL", ""),
				Description: descriptions["base_url"],
			},
			"user_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_USER_KEY", nil),
				Description: descriptions["user_key"],
			},
			"organization_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_ORGANIZATION_KEY", nil),
				Description: descriptions["organization_key"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"github_team":                    resourceGithubTeam(),
			"github_team_membership":         resourceGithubTeamMembership(),
			"github_team_repository":         resourceGithubTeamRepository(),
			"github_membership":              resourceGithubMembership(),
			"github_repository":              resourceGithubRepository(),
			"github_repository_collaborator": resourceGithubRepositoryCollaborator(),
			"github_issue_label":             resourceGithubIssueLabel(),
			"github_repository_fork":         resourceGithubRepositoryFork(),
			"github_repository_sshkey":       resourceGithubRepositorySSHKey(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"token": "The OAuth token used to connect to GitHub.",

		"organization": "The GitHub organization name to manage.",

		"base_url": "The GitHub Base API URL",

		"user_key": "The OAuth token used to connect to GitHub for user.",

		"organization_key": "The OAuth token used to connect to GitHub for owner of organization.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Token:           d.Get("token").(string),
		Organization:    d.Get("organization").(string),
		BaseURL:         d.Get("base_url").(string),
		UserKey:         d.Get("user_key").(string),
		OrganizationKey: d.Get("organization_key").(string),
	}

	return config.Clients()
}

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
			"base_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_BASE_URL", ""),
				Description: descriptions["base_url"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"github_team":                    resourceGithubTeam(),
			"github_team_membership":         resourceGithubTeamMembership(),
			"github_team_repository":         resourceGithubTeamRepository(),
			"github_membership":              resourceGithubMembership(),
			"github_repository":              resourceGithubRepository(),
			"github_repository_webhook":      resourceGithubRepositoryWebhook(),
			"github_organization_webhook":    resourceGithubOrganizationWebhook(),
			"github_repository_collaborator": resourceGithubRepositoryCollaborator(),
			"github_issue_label":             resourceGithubIssueLabel(),
			"github_branch_protection":       resourceGithubBranchProtection(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"github_user": dataSourceGithubUser(),
			"github_team": dataSourceGithubTeam(),
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
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Token:        d.Get("token").(string),
		Organization: d.Get("organization").(string),
		BaseURL:      d.Get("base_url").(string),
	}

	return config.Client()
}

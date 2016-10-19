package gitlab

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-gitlab"
)

func resourceGitlabProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceGitlabProjectCreate,
		Read:   resourceGitlabProjectRead,
		Update: resourceGitlabProjectUpdate,
		Delete: resourceGitlabProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"default_branch": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"issues_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"merge_requests_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"wiki_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"snippets_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"visibility_level": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateValueFunc([]string{"private", "internal", "public"}),
				Default:      "private",
			},

			"ssh_url_to_repo": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"http_url_to_repo": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"web_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGitlabProjectCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	options := &gitlab.CreateProjectOptions{
		Name:                 gitlab.String(d.Get("name").(string)),
		Description:          gitlab.String(d.Get("description").(string)),
		IssuesEnabled:        gitlab.Bool(d.Get("issues_enabled").(bool)),
		MergeRequestsEnabled: gitlab.Bool(d.Get("merge_requests_enabled").(bool)),
		WikiEnabled:          gitlab.Bool(d.Get("wiki_enabled").(bool)),
		SnippetsEnabled:      gitlab.Bool(d.Get("snippets_enabled").(bool)),
		VisibilityLevel:      stringToVisibilityLevel(d.Get("visibility_level").(string)),
	}

	project, _, err := client.Projects.CreateProject(options)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%d", project.ID))

	return resourceGitlabProjectRead(d, meta)
}

func resourceGitlabProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	project, _, err := client.Projects.GetProject(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", project.Name)
	d.Set("description", project.Description)
	d.Set("default_branch", project.DefaultBranch)
	d.Set("issues_enabled", project.IssuesEnabled)
	d.Set("merge_requests_enabled", project.MergeRequestsEnabled)
	d.Set("wiki_enabled", project.WikiEnabled)
	d.Set("snippets_enabled", project.SnippetsEnabled)
	d.Set("visibility_level", visibilityLevelToString(project.VisibilityLevel))

	d.Set("ssh_url_to_repo", project.SSHURLToRepo)
	d.Set("http_url_to_repo", project.HTTPURLToRepo)
	d.Set("web_url", project.WebURL)

	return nil
}

func resourceGitlabProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	edit := &gitlab.EditProjectOptions{
		Name:                 gitlab.String(d.Get("name").(string)),
		Description:          gitlab.String(d.Get("description").(string)),
		DefaultBranch:        gitlab.String(d.Get("default_branch").(string)),
		IssuesEnabled:        gitlab.Bool(d.Get("issues_enabled").(bool)),
		MergeRequestsEnabled: gitlab.Bool(d.Get("merge_requests_enabled").(bool)),
		WikiEnabled:          gitlab.Bool(d.Get("wiki_enabled").(bool)),
		SnippetsEnabled:      gitlab.Bool(d.Get("snippets_enabled").(bool)),
		VisibilityLevel:      stringToVisibilityLevel(d.Get("visibility_level").(string)),
	}
	_, _, err := client.Projects.EditProject(d.Id(), edit)
	if err != nil {
		return err
	}

	return resourceGitlabProjectRead(d, meta)
}

func resourceGitlabProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	_, err := client.Projects.DeleteProject(d.Id())
	return err
}

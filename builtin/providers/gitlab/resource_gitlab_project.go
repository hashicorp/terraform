package gitlab

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	gitlab "github.com/xanzy/go-gitlab"
)

func resourceGitlabProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceGitlabProjectCreate,
		Read:   resourceGitlabProjectRead,
		Update: resourceGitlabProjectUpdate,
		Delete: resourceGitlabProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace_id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"default_branch": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"issues_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"merge_requests_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"wiki_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"snippets_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"visibility_level": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"private", "internal", "public"}, true),
				Default:      "private",
			},

			"ssh_url_to_repo": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"http_url_to_repo": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"web_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGitlabProjectSetToState(d *schema.ResourceData, project *gitlab.Project) {
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
}

func resourceGitlabProjectCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	options := &gitlab.CreateProjectOptions{
		Name:                 gitlab.String(d.Get("name").(string)),
		IssuesEnabled:        gitlab.Bool(d.Get("issues_enabled").(bool)),
		MergeRequestsEnabled: gitlab.Bool(d.Get("merge_requests_enabled").(bool)),
		WikiEnabled:          gitlab.Bool(d.Get("wiki_enabled").(bool)),
		SnippetsEnabled:      gitlab.Bool(d.Get("snippets_enabled").(bool)),
	}

	if v, ok := d.GetOk("namespace_id"); ok {
		options.NamespaceID = gitlab.Int(v.(int))
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.String(v.(string))
	}

	if v, ok := d.GetOk("visibility_level"); ok {
		options.VisibilityLevel = stringToVisibilityLevel(v.(string))
	}

	log.Printf("[DEBUG] create gitlab project %q", options.Name)

	project, _, err := client.Projects.CreateProject(options)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%d", project.ID))

	return resourceGitlabProjectRead(d, meta)
}

func resourceGitlabProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	log.Printf("[DEBUG] read gitlab project %s", d.Id())

	project, response, err := client.Projects.GetProject(d.Id())
	if err != nil {
		if response.StatusCode == 404 {
			log.Printf("[WARN] removing project %s from state because it no longer exists in gitlab", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	resourceGitlabProjectSetToState(d, project)
	return nil
}

func resourceGitlabProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)

	options := &gitlab.EditProjectOptions{}

	if d.HasChange("name") {
		options.Name = gitlab.String(d.Get("name").(string))
	}

	if d.HasChange("description") {
		options.Description = gitlab.String(d.Get("description").(string))
	}

	if d.HasChange("default_branch") {
		options.DefaultBranch = gitlab.String(d.Get("description").(string))
	}

	if d.HasChange("visibility_level") {
		options.VisibilityLevel = stringToVisibilityLevel(d.Get("visibility_level").(string))
	}

	if d.HasChange("issues_enabled") {
		options.IssuesEnabled = gitlab.Bool(d.Get("issues_enabled").(bool))
	}

	if d.HasChange("merge_requests_enabled") {
		options.MergeRequestsEnabled = gitlab.Bool(d.Get("merge_requests_enabled").(bool))
	}

	if d.HasChange("wiki_enabled") {
		options.WikiEnabled = gitlab.Bool(d.Get("wiki_enabled").(bool))
	}

	if d.HasChange("snippets_enabled") {
		options.SnippetsEnabled = gitlab.Bool(d.Get("snippets_enabled").(bool))
	}

	log.Printf("[DEBUG] update gitlab project %s", d.Id())

	_, _, err := client.Projects.EditProject(d.Id(), options)
	if err != nil {
		return err
	}

	return resourceGitlabProjectRead(d, meta)
}

func resourceGitlabProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	log.Printf("[DEBUG] Delete gitlab project %s", d.Id())

	_, err := client.Projects.DeleteProject(d.Id())
	return err
}

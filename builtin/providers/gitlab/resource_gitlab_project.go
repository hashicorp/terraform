package gitlab

import (
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	gitlab "github.com/xanzy/go-gitlab"
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
				Default:  true,
			},
			"merge_requests_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"wiki_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"snippets_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
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

func resourceGitlabProjectUpdateFromAPI(d *schema.ResourceData, project *gitlab.Project) {
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
		Name: gitlab.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		options.Description = gitlab.String(v.(string))
	}

	if v, ok := d.GetOk("issues_enabled"); ok {
		options.IssuesEnabled = gitlab.Bool(v.(bool))
	}

	if v, ok := d.GetOk("merge_requests_enabled"); ok {
		options.MergeRequestsEnabled = gitlab.Bool(v.(bool))
	}

	if v, ok := d.GetOk("wiki_enabled"); ok {
		options.WikiEnabled = gitlab.Bool(v.(bool))
	}

	if v, ok := d.GetOk("snippets_enabled"); ok {
		options.SnippetsEnabled = gitlab.Bool(v.(bool))
	}

	if v, ok := d.GetOk("visibility_level"); ok {
		options.VisibilityLevel = stringToVisibilityLevel(v.(string))
	}

	log.Printf("[DEBUG] making create request with options %+v", options)

	project, _, err := client.Projects.CreateProject(options)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] created project %+v", project)

	d.SetId(fmt.Sprintf("%d", project.ID))

	return resourceGitlabProjectUpdateBooleans(d, meta)
}

func resourceGitlabProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	project, _, err := client.Projects.GetProject(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] read state of project %+v", project)
	resourceGitlabProjectUpdateFromAPI(d, project)
	return nil
}

// Workaround for https://gitlab.com/gitlab-org/gitlab-ce/issues/22831
type buggyBools struct {
	IssuesEnabled        *string `json:"issues_enabled,omitempty"`
	WikiEnabled          *string `json:"wiki_enabled,omitempty"`
	MergeRequestsEnabled *string `json:"merge_requests_enabled,omitempty"`
	SnippetsEnabled      *string `json:"snippets_enabled,omitempty"`
}

func resourceGitlabProjectUpdateBooleans(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	options := &buggyBools{}

	if d.HasChange("issues_enabled") {
		v := strconv.FormatBool(d.Get("issues_enabled").(bool))
		options.IssuesEnabled = &v
	}

	if d.HasChange("merge_requests_enabled") {
		v := strconv.FormatBool(d.Get("merge_requests_enabled").(bool))
		options.MergeRequestsEnabled = &v
	}

	if d.HasChange("wiki_enabled") {
		v := strconv.FormatBool(d.Get("wiki_enabled").(bool))
		options.WikiEnabled = &v
	}

	if d.HasChange("snippets_enabled") {
		v := strconv.FormatBool(d.Get("snippets_enabled").(bool))
		options.SnippetsEnabled = &v
	}

	if !reflect.DeepEqual(options, &buggyBools{}) {
		log.Printf("[DEBUG] booledit with options %+v", options)

		// partial reimplementation of EditProject
		req, err := client.NewRequest("PUT", fmt.Sprintf("projects/%s", d.Id()), options)
		if err != nil {
			return err
		}

		project := &gitlab.Project{}
		resp, err := client.Do(req, project)

		if err != nil {
			return err
		}

		log.Printf("[DEBUG] updated %+v %+v", project, resp)

		resourceGitlabProjectUpdateFromAPI(d, project)
	}

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

	if !reflect.DeepEqual(options, &gitlab.EditProjectOptions{}) {
		log.Printf("[DEBUG] edit with options %+v", options)

		project, _, err := client.Projects.EditProject(d.Id(), options)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] project edited %+v", project)

		resourceGitlabProjectUpdateFromAPI(d, project)
	}

	return resourceGitlabProjectUpdateBooleans(d, meta)
}

func resourceGitlabProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	_, err := client.Projects.DeleteProject(d.Id())
	return err
}

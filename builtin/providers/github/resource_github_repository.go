package github

import (
	"log"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepository() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubRepositoryCreate,
		Read:   resourceGithubRepositoryRead,
		Update: resourceGithubRepositoryUpdate,
		Delete: resourceGithubRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"homepage_url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"private": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"has_issues": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"has_wiki": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"has_downloads": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"auto_init": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"full_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_branch": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_clone_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"svn_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_clone_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"http_clone_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGithubRepositoryObject(d *schema.ResourceData) *github.Repository {
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	homepageUrl := d.Get("homepage_url").(string)
	private := d.Get("private").(bool)
	hasIssues := d.Get("has_issues").(bool)
	hasWiki := d.Get("has_wiki").(bool)
	hasDownloads := d.Get("has_downloads").(bool)
	autoInit := d.Get("auto_init").(bool)

	repo := &github.Repository{
		Name:         &name,
		Description:  &description,
		Homepage:     &homepageUrl,
		Private:      &private,
		HasIssues:    &hasIssues,
		HasWiki:      &hasWiki,
		HasDownloads: &hasDownloads,
		AutoInit:     &autoInit,
	}

	return repo
}

func resourceGithubRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client

	repoReq := resourceGithubRepositoryObject(d)
	log.Printf("[DEBUG] create github repository %s/%s", meta.(*Organization).name, *repoReq.Name)
	repo, _, err := client.Repositories.Create(meta.(*Organization).name, repoReq)
	if err != nil {
		return err
	}
	d.SetId(*repo.Name)

	return resourceGithubRepositoryRead(d, meta)
}

func resourceGithubRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoName := d.Id()

	log.Printf("[DEBUG] read github repository %s/%s", meta.(*Organization).name, repoName)
	repo, resp, err := client.Repositories.Get(meta.(*Organization).name, repoName)
	if err != nil {
		if resp.StatusCode == 404 {
			log.Printf(
				"[WARN] removing %s/%s from state because it no longer exists in github",
				meta.(*Organization).name,
				repoName,
			)
			d.SetId("")
			return nil
		}
		return err
	}
	d.Set("name", repoName)
	d.Set("description", repo.Description)
	d.Set("homepage_url", repo.Homepage)
	d.Set("private", repo.Private)
	d.Set("has_issues", repo.HasIssues)
	d.Set("has_wiki", repo.HasWiki)
	d.Set("has_downloads", repo.HasDownloads)
	d.Set("full_name", repo.FullName)
	d.Set("default_branch", repo.DefaultBranch)
	d.Set("ssh_clone_url", repo.SSHURL)
	d.Set("svn_url", repo.SVNURL)
	d.Set("git_clone_url", repo.GitURL)
	d.Set("http_clone_url", repo.CloneURL)
	return nil
}

func resourceGithubRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoReq := resourceGithubRepositoryObject(d)
	repoName := d.Id()
	log.Printf("[DEBUG] update github repository %s/%s", meta.(*Organization).name, repoName)
	repo, _, err := client.Repositories.Edit(meta.(*Organization).name, repoName, repoReq)
	if err != nil {
		return err
	}
	d.SetId(*repo.Name)

	return resourceGithubRepositoryRead(d, meta)
}

func resourceGithubRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	repoName := d.Id()
	log.Printf("[DEBUG] delete github repository %s/%s", meta.(*Organization).name, repoName)
	_, err := client.Repositories.Delete(meta.(*Organization).name, repoName)
	return err
}

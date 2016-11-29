package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

type CloneUrl struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

type Repository struct {
	SCM         string `json:"scm,omitempty"`
	HasWiki     bool   `json:"has_wiki,omitempty"`
	HasIssues   bool   `json:"has_issues,omitempty"`
	Website     string `json:"website,omitempty"`
	IsPrivate   bool   `json:"is_private,omitempty"`
	ForkPolicy  string `json:"fork_policy,omitempty"`
	Language    string `json:"language,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	Project     struct {
		Key string `json:"key,omitempty"`
	} `json:"project,omitempty"`
	Links struct {
		Clone []CloneUrl `json:"clone,omitempty"`
	} `json:"links,omitempty"`
}

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Update: resourceRepositoryUpdate,
		Read:   resourceRepositoryRead,
		Delete: resourceRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"scm": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "git",
			},
			"has_wiki": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"has_issues": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"website": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"clone_ssh": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"clone_https": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_private": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"fork_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "allow_forks",
			},
			"language": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newRepositoryFromResource(d *schema.ResourceData) *Repository {
	repo := &Repository{
		Name:        d.Get("name").(string),
		Language:    d.Get("language").(string),
		IsPrivate:   d.Get("is_private").(bool),
		Description: d.Get("description").(string),
		ForkPolicy:  d.Get("fork_policy").(string),
		HasWiki:     d.Get("has_wiki").(bool),
		HasIssues:   d.Get("has_issues").(bool),
		SCM:         d.Get("scm").(string),
		Website:     d.Get("website").(string),
	}

	repo.Project.Key = d.Get("project_key").(string)
	return repo
}

func resourceRepositoryUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repository := newRepositoryFromResource(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(repository)

	repository_response, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	if repository_response.StatusCode == 200 {
		decoder := json.NewDecoder(repository_response.Body)
		err = decoder.Decode(&repository)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Failed to put: %d", repository_response.StatusCode)
	}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repo := newRepositoryFromResource(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(repo)

	log.Printf("Sending %s \n", jsonpayload)

	repo_req, err := client.Post(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	), jsonpayload)

	decoder := json.NewDecoder(repo_req.Body)
	err = decoder.Decode(&repo)
	if err != nil {
		return err
	}

	log.Printf("Received %s \n", repo_req.Body)

	if repo_req.StatusCode != 200 {
		return fmt.Errorf("Failed to create repository got status code %d", repo_req.StatusCode)
	}

	d.SetId(string(fmt.Sprintf("%s/%s", d.Get("owner").(string), d.Get("name").(string))))

	return resourceRepositoryRead(d, m)
}
func resourceRepositoryRead(d *schema.ResourceData, m interface{}) error {

	client := m.(*BitbucketClient)
	repo_req, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	))

	if err != nil {
		return err
	}

	var repo Repository

	decoder := json.NewDecoder(repo_req.Body)
	err = decoder.Decode(&repo)
	if err != nil {
		return err
	}

	d.Set("scm", repo.SCM)
	d.Set("is_private", repo.IsPrivate)
	d.Set("has_wiki", repo.HasWiki)
	d.Set("has_issues", repo.HasIssues)
	d.Set("name", repo.Name)
	d.Set("language", repo.Language)
	d.Set("fork_policy", repo.ForkPolicy)
	d.Set("website", repo.Website)
	d.Set("description", repo.Description)
	d.Set("project_key", repo.Project.Key)

	for _, clone_url := range repo.Links.Clone {
		if clone_url.Name == "https" {
			d.Set("clone_https", clone_url.Href)
		} else {
			d.Set("clone_ssh", clone_url.Href)
		}
	}

	return nil
}

func resourceRepositoryDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	delete_response, err := client.Delete(fmt.Sprintf("2.0/repositories/%s/%s",
		d.Get("owner").(string),
		d.Get("name").(string),
	))

	if err != nil {
		return err
	}

	if delete_response.StatusCode != 204 {
		return fmt.Errorf("Failed to delete the repository got status code %d", delete_response.StatusCode)
	}

	return nil
}

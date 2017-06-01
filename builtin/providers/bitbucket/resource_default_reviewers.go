package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

type Reviewer struct {
	DisplayName string `json:"display_name,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	Username    string `json:"username,omitempty"`
	Type        string `json:"type,omitempty"`
}

type PaginatedReviewers struct {
	Values []Reviewer `json:"values,omitempty"`
}

func resourceDefaultReviewers() *schema.Resource {
	return &schema.Resource{
		Create: resourceDefaultReviewersCreate,
		Read:   resourceDefaultReviewersRead,
		Delete: resourceDefaultReviewersDelete,

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"reviewers": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				Set:      schema.HashString,
				ForceNew: true,
			},
		},
	}
}

func resourceDefaultReviewersCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)

	for _, user := range d.Get("reviewers").(*schema.Set).List() {
		reviewerResp, err := client.PutOnly(fmt.Sprintf("2.0/repositories/%s/%s/default-reviewers/%s",
			d.Get("owner").(string),
			d.Get("repository").(string),
			user,
		))

		if err != nil {
			return err
		}

		if reviewerResp.StatusCode != 200 {
			return fmt.Errorf("Failed to create reviewer %s got code %d", user.(string), reviewerResp.StatusCode)
		}

		defer reviewerResp.Body.Close()
	}

	d.SetId(fmt.Sprintf("%s/%s/reviewers", d.Get("owner").(string), d.Get("repository").(string)))
	return resourceDefaultReviewersRead(d, m)
}
func resourceDefaultReviewersRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)

	reviewersResponse, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/default-reviewers",
		d.Get("owner").(string),
		d.Get("repository").(string),
	))

	var reviewers PaginatedReviewers

	decoder := json.NewDecoder(reviewersResponse.Body)
	err = decoder.Decode(&reviewers)
	if err != nil {
		return err
	}

	terraformReviewers := make([]string, 0, len(reviewers.Values))

	for _, reviewer := range reviewers.Values {
		terraformReviewers = append(terraformReviewers, reviewer.Username)
	}

	d.Set("reviewers", terraformReviewers)

	return nil
}
func resourceDefaultReviewersDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)

	for _, user := range d.Get("reviewers").(*schema.Set).List() {
		resp, err := client.Delete(fmt.Sprintf("2.0/repositories/%s/%s/default-reviewers/%s",
			d.Get("owner").(string),
			d.Get("repository").(string),
			user.(string),
		))

		if err != nil {
			return err
		}

		if resp.StatusCode != 204 {
			return fmt.Errorf("[%d] Could not delete %s from default reviewer",
				resp.StatusCode,
				user.(string),
			)
		}
		defer resp.Body.Close()
	}
	return nil
}

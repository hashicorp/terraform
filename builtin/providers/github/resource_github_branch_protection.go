package github

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubBranchProtection() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubBranchProtectionCreate,
		Read:   resourceGithubBranchProtectionRead,
		Update: resourceGithubBranchProtectionUpdate,
		Delete: resourceGithubBranchProtectionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"branch": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"required_status_checks": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"strict": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"contexts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"required_pull_request_reviews": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dismissal_restrictions": {
							Type:     schema.TypeList,
							Optional: true,
							DefaultFunc: func() (interface{}, error) {
								return []string{}, nil
							},
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"users": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"teams": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"dismiss_stale_reviews": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"enforce_admins": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"restrictions": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"users": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"teams": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceGithubBranchProtectionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r := d.Get("repository").(string)
	b := d.Get("branch").(string)

	protectionRequest, err := buildProtectionRequest(d)
	if err != nil {
		return err
	}

	_, _, err = client.Repositories.UpdateBranchProtection(context.TODO(), meta.(*Organization).name, r, b, protectionRequest)
	if err != nil {
		return err
	}
	d.SetId(buildTwoPartID(&r, &b))

	return resourceGithubBranchProtectionRead(d, meta)
}

func resourceGithubBranchProtectionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, b := parseTwoPartID(d.Id())

	githubProtection, _, err := client.Repositories.GetBranchProtection(context.TODO(), meta.(*Organization).name, r, b)
	if err != nil {
		if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("repository", r)
	d.Set("branch", b)

	d.Set("enforce_admins", githubProtection.EnforceAdmins.Enabled)
	rsc := githubProtection.RequiredStatusChecks
	if rsc != nil {
		d.Set("required_status_checks", []interface{}{
			map[string]interface{}{
				"strict":   rsc.Strict,
				"contexts": rsc.Contexts,
			},
		})
	} else {
		d.Set("required_status_checks", []interface{}{})
	}

	rprr := githubProtection.RequiredPullRequestReviews
	if rprr != nil {
		var userLogins []string
		var teamSlugs []string
		dr := rprr.DismissalRestrictions
		if dr != nil {
			for _, u := range dr.Users {
				if u.Login != nil {
					userLogins = append(userLogins, *u.Login)
				}
			}
			for _, t := range dr.Teams {
				if t.Slug != nil {
					teamSlugs = append(teamSlugs, *t.Slug)
				}
			}
		}

		d.Set("required_pull_request_reviews", []interface{}{
			map[string]interface{}{
				"dismiss_stale_reviews": rprr.DismissStaleReviews,
				"dismissal_restrictions": []interface{}{
					map[string]interface{}{
						"users": userLogins,
						"teams": teamSlugs,
					},
				},
			},
		})
	} else {
		d.Set("required_pull_request_reviews", []interface{}{})
	}

	restrictions := githubProtection.Restrictions
	if restrictions != nil {
		var userLogins []string
		for _, u := range restrictions.Users {
			if u.Login != nil {
				userLogins = append(userLogins, *u.Login)
			}
		}
		var teamSlugs []string
		for _, t := range restrictions.Teams {
			if t.Slug != nil {
				teamSlugs = append(teamSlugs, *t.Slug)
			}
		}

		d.Set("restrictions", []interface{}{
			map[string]interface{}{
				"users": userLogins,
				"teams": teamSlugs,
			},
		})
	} else {
		d.Set("restrictions", []interface{}{})
	}

	return nil
}

func resourceGithubBranchProtectionUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, b := parseTwoPartID(d.Id())

	protectionRequest, err := buildProtectionRequest(d)
	if err != nil {
		return err
	}

	_, _, err = client.Repositories.UpdateBranchProtection(context.TODO(), meta.(*Organization).name, r, b, protectionRequest)
	if err != nil {
		return err
	}
	d.SetId(buildTwoPartID(&r, &b))

	return resourceGithubBranchProtectionRead(d, meta)
}

func resourceGithubBranchProtectionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	r, b := parseTwoPartID(d.Id())

	_, err := client.Repositories.RemoveBranchProtection(context.TODO(), meta.(*Organization).name, r, b)
	return err
}

func buildProtectionRequest(d *schema.ResourceData) (*github.ProtectionRequest, error) {
	protectionRequest := new(github.ProtectionRequest)

	if v, ok := d.GetOk("enforce_admins"); ok {
		vL := v.(bool)
		protectionRequest.EnforceAdmins = vL
	}

	if v, ok := d.GetOk("required_status_checks"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return nil, errors.New("cannot specify required_status_checks more than one time")
		}

		for _, v := range vL {
			m := v.(map[string]interface{})

			rsc := new(github.RequiredStatusChecks)
			rsc.Strict = m["strict"].(bool)

			rsc.Contexts = []string{}
			if contexts, ok := m["contexts"].([]interface{}); ok {
				for _, c := range contexts {
					rsc.Contexts = append(rsc.Contexts, c.(string))
				}
			}

			protectionRequest.RequiredStatusChecks = rsc
		}
	}

	if v, ok := d.GetOk("required_pull_request_reviews"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return nil, errors.New("cannot specify required_pull_request_reviews more than one time")
		}

		for _, v := range vL {
			m := v.(map[string]interface{})

			rprr := new(github.RequiredPullRequestReviewsRequest)
			rprr.DismissStaleReviews = m["dismiss_stale_reviews"].(bool)

			if v, ok := m["dismissal_restrictions"]; ok {
				vL := v.([]interface{})
				if len(vL) > 1 {
					return nil, errors.New("cannot specify restrictions more than one time")
				}

				for _, v := range vL {
					m := v.(map[string]interface{})

					restrictions := new(github.RestrictionsRequest)

					restrictions.Users = []string{}
					if users, ok := m["users"].([]interface{}); ok {
						for _, u := range users {
							restrictions.Users = append(restrictions.Users, u.(string))
						}
					}

					restrictions.Teams = []string{}
					if teams, ok := m["teams"].([]interface{}); ok {
						for _, t := range teams {
							restrictions.Teams = append(restrictions.Teams, t.(string))
						}
					}

					rprr.DismissalRestrictionsRequest = restrictions
				}
			}

			protectionRequest.RequiredPullRequestReviews = rprr
		}
	}

	if v, ok := d.GetOk("restrictions"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return nil, errors.New("cannot specify restrictions more than one time")
		}

		for _, v := range vL {
			m := v.(map[string]interface{})

			restrictions := new(github.RestrictionsRequest)

			restrictions.Users = []string{}
			if users, ok := m["users"].([]interface{}); ok {
				for _, u := range users {
					restrictions.Users = append(restrictions.Users, u.(string))
				}
			}

			restrictions.Teams = []string{}
			if teams, ok := m["teams"].([]interface{}); ok {
				for _, t := range teams {
					restrictions.Teams = append(restrictions.Teams, t.(string))
				}
			}

			protectionRequest.Restrictions = restrictions
		}
	}

	return protectionRequest, nil
}

package github

import (
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepositorySSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubRepositorySSHKeyCreate,
		Read:   resourceGithubRepositorySSHKeyRead,
		Delete: resourceGithubRepositorySSHKeyDelete,

		Schema: map[string]*schema.Schema{
			// owner specifies the owner of the repository
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sshkey": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// isErr422ValidationFailed return true if error contains the string:
// '422 Validation Failed'. This error is special cased so we can ignore it on
// when it occurs during rebuilding of stack template.
func isErr422ValidationFailed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "422 Validation Failed")
}

func resourceGithubRepositorySSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients).UserClient
	title := d.Get("title").(string)
	keySSH := d.Get("sshkey").(string)

	key := &github.Key{
		Title: &title,
		Key:   &keySSH,
	}

	// CreateKey creates a public key. Requires that you are authenticated via Basic Auth,
	// or OAuth with at least `write:public_key` scope.
	//
	// If SSH key is already set up, when u try to add same SSHKEY then
	//you are gonna get 422: Validation error.
	responseKey, _, err := client.Users.CreateKey(key)
	if err != nil && !isErr422ValidationFailed(err) {
		return err
	}

	d.SetId(strconv.Itoa(*responseKey.ID))

	return resourceGithubRepositorySSHKeyRead(d, meta)
}

func resourceGithubRepositorySSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients).UserClient
	id := d.Id()
	i, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	key, _, err := client.Users.GetKey(i)
	if err != nil || key == nil {
		d.SetId("")
		return nil
	}

	return nil
}

func resourceGithubRepositorySSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Clients).UserClient
	id := d.Id()
	if id == "" {
		return nil
	}
	i, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	_, err = client.Users.DeleteKey(i)

	return err
}

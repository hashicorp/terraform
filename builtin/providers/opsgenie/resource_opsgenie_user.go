package opsgenie

import (
	"log"

	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/opsgenie/opsgenie-go-sdk/user"
)

func resourceOpsGenieUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpsGenieUserCreate,
		Read:   resourceOpsGenieUserRead,
		Update: resourceOpsGenieUserUpdate,
		Delete: resourceOpsGenieUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"username": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateOpsGenieUserUsername,
			},
			"full_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOpsGenieUserFullName,
			},
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOpsGenieUserRole,
			},
			"locale": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "en_US",
			},
			"timezone": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "America/New_York",
			},
		},
	}
}

func resourceOpsGenieUserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).users

	username := d.Get("username").(string)
	fullName := d.Get("full_name").(string)
	role := d.Get("role").(string)
	locale := d.Get("locale").(string)
	timeZone := d.Get("timezone").(string)

	createRequest := user.CreateUserRequest{
		Username: username,
		Fullname: fullName,
		Role:     role,
		Locale:   locale,
		Timezone: timeZone,
	}

	log.Printf("[INFO] Creating OpsGenie user '%s'", username)
	createResponse, err := client.Create(createRequest)
	if err != nil {
		return err
	}

	err = checkOpsGenieResponse(createResponse.Code, createResponse.Status)
	if err != nil {
		return err
	}

	getRequest := user.GetUserRequest{
		Username: username,
	}

	getResponse, err := client.Get(getRequest)
	if err != nil {
		return err
	}

	d.SetId(getResponse.Id)

	return resourceOpsGenieUserRead(d, meta)
}

func resourceOpsGenieUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).users

	listRequest := user.ListUsersRequest{}
	listResponse, err := client.List(listRequest)
	if err != nil {
		return err
	}

	var found *user.GetUserResponse
	for _, user := range listResponse.Users {
		if user.Id == d.Id() {
			found = &user
			break
		}
	}

	if found == nil {
		d.SetId("")
		log.Printf("[INFO] User %q not found. Removing from state", d.Get("username").(string))
		return nil
	}

	getRequest := user.GetUserRequest{
		Id: d.Id(),
	}

	getResponse, err := client.Get(getRequest)
	if err != nil {
		return err
	}

	d.Set("username", getResponse.Username)
	d.Set("full_name", getResponse.Fullname)
	d.Set("role", getResponse.Role)
	d.Set("locale", getResponse.Locale)
	d.Set("timezone", getResponse.Timezone)

	return nil
}

func resourceOpsGenieUserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).users

	username := d.Get("username").(string)
	fullName := d.Get("full_name").(string)
	role := d.Get("role").(string)
	locale := d.Get("locale").(string)
	timeZone := d.Get("timezone").(string)

	log.Printf("[INFO] Updating OpsGenie user '%s'", username)

	updateRequest := user.UpdateUserRequest{
		Id:       d.Id(),
		Fullname: fullName,
		Role:     role,
		Locale:   locale,
		Timezone: timeZone,
	}

	updateResponse, err := client.Update(updateRequest)
	if err != nil {
		return err
	}

	err = checkOpsGenieResponse(updateResponse.Code, updateResponse.Status)
	if err != nil {
		return err
	}

	return nil
}

func resourceOpsGenieUserDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting OpsGenie user '%s'", d.Get("username").(string))
	client := meta.(*OpsGenieClient).users

	deleteRequest := user.DeleteUserRequest{
		Id: d.Id(),
	}

	_, err := client.Delete(deleteRequest)
	if err != nil {
		return err
	}

	return nil
}

func validateOpsGenieUserUsername(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) >= 100 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 100 characters: %q %d", k, value, len(value)))
	}

	return
}

func validateOpsGenieUserFullName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) >= 512 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 512 characters: %q %d", k, value, len(value)))
	}

	return
}

func validateOpsGenieUserRole(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) >= 512 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 512 characters: %q %d", k, value, len(value)))
	}

	return
}

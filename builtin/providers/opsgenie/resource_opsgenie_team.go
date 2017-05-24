package opsgenie

import (
	"log"

	"fmt"
	"strings"

	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/opsgenie/opsgenie-go-sdk/team"
)

func resourceOpsGenieTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpsGenieTeamCreate,
		Read:   resourceOpsGenieTeamRead,
		Update: resourceOpsGenieTeamUpdate,
		Delete: resourceOpsGenieTeamDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOpsGenieTeamName,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"member": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Type:     schema.TypeString,
							Required: true,
						},

						"role": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "user",
							ValidateFunc: validateOpsGenieTeamRole,
						},
					},
				},
			},
		},
	}
}

func resourceOpsGenieTeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).teams

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	createRequest := team.CreateTeamRequest{
		Name:        name,
		Description: description,
		Members:     expandOpsGenieTeamMembers(d),
	}

	log.Printf("[INFO] Creating OpsGenie team '%s'", name)

	createResponse, err := client.Create(createRequest)
	if err != nil {
		return err
	}

	err = checkOpsGenieResponse(createResponse.Code, createResponse.Status)
	if err != nil {
		return err
	}

	getRequest := team.GetTeamRequest{
		Name: name,
	}

	getResponse, err := client.Get(getRequest)
	if err != nil {
		return err
	}

	d.SetId(getResponse.Id)

	return resourceOpsGenieTeamRead(d, meta)
}

func resourceOpsGenieTeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).teams

	listRequest := team.ListTeamsRequest{}
	listResponse, err := client.List(listRequest)
	if err != nil {
		return err
	}

	var found *team.GetTeamResponse
	for _, team := range listResponse.Teams {
		if team.Id == d.Id() {
			found = &team
			break
		}
	}

	if found == nil {
		d.SetId("")
		log.Printf("[INFO] Team %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	getRequest := team.GetTeamRequest{
		Id: d.Id(),
	}

	getResponse, err := client.Get(getRequest)
	if err != nil {
		return err
	}

	d.Set("name", getResponse.Name)
	d.Set("description", getResponse.Description)
	d.Set("member", flattenOpsGenieTeamMembers(getResponse.Members))

	return nil
}

func resourceOpsGenieTeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).teams
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	updateRequest := team.UpdateTeamRequest{
		Id:          d.Id(),
		Name:        name,
		Description: description,
		Members:     expandOpsGenieTeamMembers(d),
	}

	log.Printf("[INFO] Updating OpsGenie team '%s'", name)

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

func resourceOpsGenieTeamDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting OpsGenie team '%s'", d.Get("name").(string))
	client := meta.(*OpsGenieClient).teams

	deleteRequest := team.DeleteTeamRequest{
		Id: d.Id(),
	}

	_, err := client.Delete(deleteRequest)
	if err != nil {
		return err
	}

	return nil
}

func flattenOpsGenieTeamMembers(input []team.Member) []interface{} {
	members := make([]interface{}, 0, len(input))
	for _, inputMember := range input {
		outputMember := make(map[string]interface{})
		outputMember["username"] = inputMember.User
		outputMember["role"] = inputMember.Role

		members = append(members, outputMember)
	}

	return members
}

func expandOpsGenieTeamMembers(d *schema.ResourceData) []team.Member {
	input := d.Get("member").([]interface{})

	members := make([]team.Member, 0, len(input))
	if input == nil {
		return members
	}

	for _, v := range input {
		config := v.(map[string]interface{})

		username := config["username"].(string)
		role := config["role"].(string)

		member := team.Member{
			User: username,
			Role: role,
		}

		members = append(members, member)
	}

	return members
}

func validateOpsGenieTeamName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alpha numeric characters and underscores are allowed in %q: %q", k, value))
	}

	if len(value) >= 100 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 100 characters: %q %d", k, value, len(value)))
	}

	return
}

func validateOpsGenieTeamRole(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	families := map[string]bool{
		"admin": true,
		"user":  true,
	}

	if !families[value] {
		errors = append(errors, fmt.Errorf("OpsGenie Team Role can only be 'Admin' or 'User'"))
	}

	return
}

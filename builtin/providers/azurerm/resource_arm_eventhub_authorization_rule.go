package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/eventhub"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmEventHubAuthorizationRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmEventHubAuthorizationRuleCreateUpdate,
		Read:   resourceArmEventHubAuthorizationRuleRead,
		Update: resourceArmEventHubAuthorizationRuleCreateUpdate,
		Delete: resourceArmEventHubAuthorizationRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"namespace_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"eventhub_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"listen": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"send": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"manage": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"primary_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_connection_string": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_connection_string": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmEventHubAuthorizationRuleCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).eventHubClient
	log.Printf("[INFO] preparing arguments for AzureRM EventHub Authorization Rule creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	namespaceName := d.Get("namespace_name").(string)
	eventHubName := d.Get("eventhub_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	rights, err := expandEventHubAuthorizationRuleAccessRights(d)
	if err != nil {
		return err
	}

	parameters := eventhub.SharedAccessAuthorizationRuleCreateOrUpdateParameters{
		Name:     &name,
		Location: &location,
		SharedAccessAuthorizationRuleProperties: &eventhub.SharedAccessAuthorizationRuleProperties{
			Rights: rights,
		},
	}

	_, err = client.CreateOrUpdateAuthorizationRule(resGroup, namespaceName, eventHubName, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.GetAuthorizationRule(resGroup, namespaceName, eventHubName, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read EventHub Authorization Rule %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmEventHubAuthorizationRuleRead(d, meta)
}

func resourceArmEventHubAuthorizationRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).eventHubClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	eventHubName := id.Path["eventhubs"]
	name := id.Path["authorizationRules"]

	resp, err := client.GetAuthorizationRule(resGroup, namespaceName, eventHubName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure EventHub Authorization Rule %s: %+v", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	keysResp, err := client.ListKeys(resGroup, namespaceName, eventHubName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure EventHub Authorization Rule List Keys %s: %+v", name, err)
	}

	d.Set("name", name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("eventhub_name", eventHubName)
	d.Set("namespace_name", namespaceName)
	d.Set("resource_group_name", resGroup)

	flattenEventHubAuthorizationRuleAccessRights(d, resp)

	d.Set("primary_key", keysResp.PrimaryKey)
	d.Set("primary_connection_string", keysResp.PrimaryConnectionString)
	d.Set("secondary_key", keysResp.SecondaryKey)
	d.Set("secondary_connection_string", keysResp.SecondaryConnectionString)

	return nil
}

func resourceArmEventHubAuthorizationRuleDelete(d *schema.ResourceData, meta interface{}) error {
	eventhubClient := meta.(*ArmClient).eventHubClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	eventHubName := id.Path["eventhubs"]
	name := id.Path["authorizationRules"]

	resp, err := eventhubClient.DeleteAuthorizationRule(resGroup, namespaceName, eventHubName, name)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of EventHub Authorization Rule '%s': %+v", name, err)
	}

	return nil
}

func expandEventHubAuthorizationRuleAccessRights(d *schema.ResourceData) (*[]eventhub.AccessRights, error) {
	canSend := d.Get("send").(bool)
	canListen := d.Get("listen").(bool)
	canManage := d.Get("manage").(bool)
	rights := []eventhub.AccessRights{}
	if canListen {
		rights = append(rights, eventhub.Listen)
	}

	if canSend {
		rights = append(rights, eventhub.Send)
	}

	if canManage {
		rights = append(rights, eventhub.Manage)
	}

	if len(rights) == 0 {
		return nil, fmt.Errorf("At least one Authorization Rule State must be enabled (e.g. Listen/Manage/Send)")
	}

	if canManage && !(canListen && canSend) {
		return nil, fmt.Errorf("In order to enable the 'Manage' Authorization Rule - both the 'Listen' and 'Send' rules must be enabled")
	}

	return &rights, nil
}

func flattenEventHubAuthorizationRuleAccessRights(d *schema.ResourceData, resp eventhub.SharedAccessAuthorizationRuleResource) {

	var canListen = false
	var canSend = false
	var canManage = false

	for _, right := range *resp.Rights {
		switch right {
		case eventhub.Listen:
			canListen = true
		case eventhub.Send:
			canSend = true
		case eventhub.Manage:
			canManage = true
		default:
			log.Printf("[DEBUG] Unknown Authorization Rule Right '%s'", right)
		}
	}

	d.Set("listen", canListen)
	d.Set("send", canSend)
	d.Set("manage", canManage)
}

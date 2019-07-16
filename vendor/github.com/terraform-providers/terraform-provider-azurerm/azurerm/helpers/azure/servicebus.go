package azure

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
)

//validation
func ValidateServiceBusNamespaceName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile("^[a-zA-Z][-a-zA-Z0-9]{4,48}[a-zA-Z0-9]$"),
		"The namespace name can contain only letters, numbers, and hyphens. The namespace must start with a letter, and it must end with a letter or number and be between 6 and 50 characters long.",
	)
}

func ValidateServiceBusQueueName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`^[a-zA-Z0-9][\w-./~]{0,258}([a-zA-Z0-9])?$`),
		"The topic name can contain only letters, numbers, periods, hyphens, tildas, forward slashes and underscores. The namespace must start and end with a letter or number and be less then 260 characters long.",
	)
}

func ValidateServiceBusSubscriptionName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile("^[a-zA-Z0-9][-._a-zA-Z0-9]{0,48}([a-zA-Z0-9])?$"),
		"The name can contain only letters, numbers, periods, hyphens and underscores. The name must start and end with a letter or number and be less then 50 characters long.",
	)
}

func ValidateServiceBusTopicName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile("^[a-zA-Z][-._~a-zA-Z0-9]{0,258}([a-zA-Z0-9])?$"),
		"The topic name can contain only letters, numbers, periods, hyphens, tildas and underscores. The namespace must start with a letter, and it must end with a letter or number and be less then 260 characters long.",
	)
}

func ValidateServiceBusAuthorizationRuleName() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile("^[a-zA-Z0-9][-._a-zA-Z0-9]{0,48}([a-zA-Z0-9])?$"),
		"The name can contain only letters, numbers, periods, hyphens and underscores. The name must start and end with a letter or number and be less the 50 characters long.",
	)
}

func ExpandServiceBusAuthorizationRuleRights(d *schema.ResourceData) *[]servicebus.AccessRights {
	rights := make([]servicebus.AccessRights, 0)

	if d.Get("listen").(bool) {
		rights = append(rights, servicebus.Listen)
	}

	if d.Get("send").(bool) {
		rights = append(rights, servicebus.Send)
	}

	if d.Get("manage").(bool) {
		rights = append(rights, servicebus.Manage)
	}

	return &rights
}

func FlattenServiceBusAuthorizationRuleRights(rights *[]servicebus.AccessRights) (listen, send, manage bool) {
	//zero (initial) value for a bool in go is false

	if rights != nil {
		for _, right := range *rights {
			switch right {
			case servicebus.Listen:
				listen = true
			case servicebus.Send:
				send = true
			case servicebus.Manage:
				manage = true
			default:
				log.Printf("[DEBUG] Unknown Authorization Rule Right '%s'", right)
			}
		}
	}

	return listen, send, manage
}

//shared schema
func MergeSchema(a map[string]*schema.Schema, b map[string]*schema.Schema) map[string]*schema.Schema {
	s := map[string]*schema.Schema{}

	for k, v := range a {
		s[k] = v
	}

	for k, v := range b {
		s[k] = v
	}

	return s
}

func ServiceBusAuthorizationRuleSchemaFrom(s map[string]*schema.Schema) map[string]*schema.Schema {

	authSchema := map[string]*schema.Schema{
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
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},

		"primary_connection_string": {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},

		"secondary_key": {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},

		"secondary_connection_string": {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
	}
	return MergeSchema(s, authSchema)
}

func ServiceBusAuthorizationRuleCustomizeDiff(d *schema.ResourceDiff, _ interface{}) error {
	listen, hasListen := d.GetOk("listen")
	send, hasSend := d.GetOk("send")
	manage, hasManage := d.GetOk("manage")

	if !hasListen && !hasSend && !hasManage {
		return fmt.Errorf("One of the `listen`, `send` or `manage` properties needs to be set")
	}

	if manage.(bool) && !listen.(bool) && !send.(bool) {
		return fmt.Errorf("if `manage` is set both `listen` and `send` must be set to true too")
	}

	return nil
}

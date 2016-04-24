package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
	"github.com/jen20/riviera/sql"
)

func resourceArmSqlFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSqlFirewallRuleCreate,
		Read:   resourceArmSqlFirewallRuleRead,
		Update: resourceArmSqlFirewallRuleCreate,
		Delete: resourceArmSqlFirewallRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"server_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"start_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"end_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceArmSqlFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = &sql.CreateOrUpdateFirewallRule{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ServerName:        d.Get("server_name").(string),
		StartIPAddress:    azure.String(d.Get("start_ip_address").(string)),
		EndIPAddress:      azure.String(d.Get("end_ip_address").(string)),
	}

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating SQL Server Firewall Rule: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating SQL Server Firewall Rule: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &sql.GetFirewallRule{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ServerName:        d.Get("server_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Server Firewall Rule: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading SQL Server Firewall Rule: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetFirewallRuleResponse)
	d.SetId(*resp.ID)

	return resourceArmSqlFirewallRuleRead(d, meta)
}

func resourceArmSqlFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &sql.GetFirewallRule{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Server Firewall Rule: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading SQL Server Firewall Rule %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading SQL Server Firewall Rule: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetFirewallRuleResponse)

	d.Set("start_ip_address", resp.StartIPAddress)
	d.Set("end_ip_address", resp.EndIPAddress)

	return nil
}

func resourceArmSqlFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &sql.DeleteFirewallRule{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting SQL Server Firewall Rule: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting SQL Server Firewall Rule: %s", deleteResponse.Error)
	}

	return nil
}

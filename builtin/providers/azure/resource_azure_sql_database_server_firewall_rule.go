package azure

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureSqlDatabaseServerFirewallRule returns the *schema.Resource
// associated to a firewall rule of a database server in Azure.
func resourceAzureSqlDatabaseServerFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSqlDatabaseServerFirewallRuleCreate,
		Read:   resourceAzureSqlDatabaseServerFirewallRuleRead,
		Update: resourceAzureSqlDatabaseServerFirewallRuleUpdate,
		Delete: resourceAzureSqlDatabaseServerFirewallRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"database_server_names": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"start_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"end_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

// resourceAzureSqlDatabaseServerFirewallRuleCreate does all the necessary API
// calls to create the SQL Database Server Firewall Rule on Azure.
func resourceAzureSqlDatabaseServerFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	name := d.Get("name").(string)
	params := sql.FirewallRuleCreateParams{
		Name:           name,
		StartIPAddress: d.Get("start_ip").(string),
		EndIPAddress:   d.Get("end_ip").(string),
	}

	// loop over all the database servers and apply the firewall rule to each:
	serverNames := d.Get("database_server_names").(*schema.Set).List()
	for _, srv := range serverNames {
		serverName := srv.(string)

		log.Printf("[INFO] Sending Azure Database Server Firewall Rule %q creation request for Server %q.", name, serverName)
		if err := sqlClient.CreateFirewallRule(serverName, params); err != nil {
			return fmt.Errorf("Error creating Azure Database Server Firewall Rule %q for Server %q: %s", name, serverName, err)
		}
	}

	d.SetId(name)
	return nil
}

// resourceAzureSqlDatabaseServerFirewallRuleRead does all the necessary API
// calls to read the state of the SQL Database Server Firewall Rule on Azure.
func resourceAzureSqlDatabaseServerFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	name := d.Get("name").(string)
	remaining := schema.NewSet(schema.HashString, nil)

	// for each of our servers; check to see if the rule is still present:
	var found bool
	for _, srv := range d.Get("database_server_names").(*schema.Set).List() {
		serverName := srv.(string)

		log.Printf("[INFO] Sending Azure Database Server Firewall Rule list query for server %q.", serverName)
		rules, err := sqlClient.ListFirewallRules(serverName)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				// it means that the database server this rule belonged to has
				// been deleted in the meantime.
				continue
			} else {
				return fmt.Errorf("Error getting Azure Firewall Rules for Database Server %q: %s", serverName, err)
			}

		}

		// look for our rule:
		for _, rule := range rules.FirewallRules {
			if rule.Name == name {
				found = true
				remaining.Add(serverName)

				break
			}
		}
	}

	// check to see if there is still any Database Server still having this rule:
	if !found {
		d.SetId("")
		return nil
	}

	// else; update the list of Database Servers still having this rule:
	d.Set("database_server_names", remaining)
	return nil
}

// resourceAzureSqlDatabaseServerFirewallRuleUpdate does all the necessary API
// calls to update the state of the SQL Database Server Firewall Rule on Azure.
func resourceAzureSqlDatabaseServerFirewallRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	var found bool
	name := d.Get("name").(string)
	updateParams := sql.FirewallRuleUpdateParams{
		Name:           name,
		StartIPAddress: d.Get("start_ip").(string),
		EndIPAddress:   d.Get("end_ip").(string),
	}

	// for each of the Database Servers our rules concerns; issue the update:
	remaining := schema.NewSet(schema.HashString, nil)
	for _, srv := range d.Get("database_server_names").(*schema.Set).List() {
		serverName := srv.(string)

		log.Printf("[INFO] Issuing Azure Database Server Firewall Rule list for Database Server %q: %s.", name, serverName)
		rules, err := sqlClient.ListFirewallRules(serverName)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				// it means that the database server this rule belonged to has
				// been deleted in the meantime.
				continue
			} else {
				return fmt.Errorf("Error getting Azure Firewall Rules for Database Server %q: %s", serverName, err)
			}

		}

		// look for our rule:
		for _, rule := range rules.FirewallRules {
			if rule.Name == name {
				// take note of the fact that this Database Server still has
				// this rule:
				found = true
				remaining.Add(serverName)

				// go ahead and update the rule:
				log.Printf("[INFO] Issuing update of Azure Database Server Firewall Rule %q in Server %q.", name, serverName)
				if err := sqlClient.UpdateFirewallRule(serverName, name, updateParams); err != nil {
					return fmt.Errorf("Error updating Azure Database Server Firewall Rule %q for Server %q: %s", name, serverName, err)
				}

				break
			}
		}
	}

	// check to see if the rule is still exists on any of the servers:
	if !found {
		d.SetId("")
		return nil
	}

	// else; update the list with the remaining Servers:
	d.Set("database_server_names", remaining)
	return nil
}

// resourceAzureSqlDatabaseServerFirewallRuleDelete does all the necessary API
// calls to delete the SQL Database Server Firewall Rule on Azure.
func resourceAzureSqlDatabaseServerFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	name := d.Get("name").(string)
	for _, srv := range d.Get("database_server_names").(*schema.Set).List() {
		serverName := srv.(string)

		log.Printf("[INFO] Sending Azure Database Server Firewall Rule list query for Server %q.", serverName)
		rules, err := sqlClient.ListFirewallRules(serverName)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				// it means that the database server this rule belonged to has
				// been deleted in the meantime.
				continue
			} else {
				return fmt.Errorf("Error getting Azure Firewall Rules for Database Server %q: %s", serverName, err)
			}

		}

		// look for our rule:
		for _, rule := range rules.FirewallRules {
			if rule.Name == name {
				// go ahead and delete the rule:
				log.Printf("[INFO] Issuing deletion of Azure Database Server Firewall Rule %q in Server %q.", name, serverName)
				if err := sqlClient.DeleteFirewallRule(serverName, name); err != nil {
					if strings.Contains(err.Error(), "Cannot open server") {
						break
					}
					return fmt.Errorf("Error deleting Azure Database Server Firewall Rule %q for Server %q: %s", name, serverName, err)
				}

				break
			}
		}

	}

	return nil
}

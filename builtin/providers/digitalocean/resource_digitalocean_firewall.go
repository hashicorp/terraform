package digitalocean

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanFirewallCreate,
		Read:   resourceDigitalOceanFirewallRead,
		Update: resourceDigitalOceanFirewallUpdate,
		Delete: resourceDigitalOceanFirewallDelete,
		Exists: resourceDigitalOceanFirewallExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"pending_changes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"droplet_id": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"removing": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"status": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"droplet_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"tags": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"inbound_rules": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"port_range": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"source_addresses": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"source_tags": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"source_droplet_ids": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeInt},
							Optional: true,
						},
						"source_load_balancer_uids": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
					},
				},
			},

			"outbound_rules": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"port_range": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"destination_addresses": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"destination_tags": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"destination_droplet_ids": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeInt},
							Optional: true,
						},
						"destination_load_balancer_uids": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceDigitalOceanFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	opts, err := firewallRequest(d, client)
	if err != nil {
		return fmt.Errorf("Error in firewall request: %s", err)
	}

	log.Printf("[DEBUG] Firewall create configuration: %#v", opts)

	firewall, _, err := client.Firewalls.Create(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("Error creating firewall: %s", err)
	}

	// Assign the firewall id
	d.SetId(firewall.ID)

	log.Printf("[INFO] Firewall ID: %s", d.Id())

	return resourceDigitalOceanFirewallRead(d, meta)
}

func resourceDigitalOceanFirewallRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Retrieve the firewall properties for updating the state
	firewall, resp, err := client.Firewalls.Get(context.Background(), d.Id())
	if err != nil {
		// check if the firewall no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] DigitalOcean Firewall (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving firewall: %s", err)
	}

	d.Set("status", firewall.Status)
	d.Set("create_at", firewall.Created)
	d.Set("pending_changes", firewallPendingChanges(d, firewall))
	d.Set("name", firewall.Name)
	d.Set("droplet_ids", firewall.DropletIDs)
	d.Set("tags", firewall.Tags)
	d.Set("inbound_rules", matchFirewallInboundRules(d, firewall))
	d.Set("outbound_rules", matchFirewallOutboundRules(d, firewall))

	return nil
}

func resourceDigitalOceanFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	opts, err := firewallRequest(d, client)
	if err != nil {
		return fmt.Errorf("Error in firewall request: %s", err)
	}

	log.Printf("[DEBUG] Firewall update configuration: %#v", opts)

	_, _, err = client.Firewalls.Update(context.Background(), d.Id(), opts)
	if err != nil {
		return fmt.Errorf("Error updating firewall: %s", err)
	}

	return resourceDigitalOceanFirewallRead(d, meta)
}

func resourceDigitalOceanFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting firewall: %s", d.Id())

	// Destroy the droplet
	_, err := client.Firewalls.Delete(context.Background(), d.Id())

	// Handle remotely destroyed droplets
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting firewall: %s", err)
	}

	return nil
}

func resourceDigitalOceanFirewallExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Exists firewall: %s", d.Id())

	// Retrieve the firewall properties for updating the state
	_, resp, err := client.Firewalls.Get(context.Background(), d.Id())
	if err != nil {
		// check if the firewall no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] DigitalOcean Firewall (%s) not found", d.Id())
			d.SetId("")
			return false, nil
		}

		return false, fmt.Errorf("Error retrieving firewall: %s", err)
	}

	return true, nil
}

func firewallRequest(d *schema.ResourceData, client *godo.Client) (*godo.FirewallRequest, error) {
	// Build up our firewall request
	opts := &godo.FirewallRequest{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("droplet_ids"); ok {
		var droplets []int
		for _, id := range v.([]interface{}) {
			i, err := strconv.Atoi(id.(string))
			if err != nil {
				return nil, err
			}
			droplets = append(droplets, i)
		}
		opts.DropletIDs = droplets
	}

	if v, ok := d.GetOk("tags"); ok {
		var tags []string
		for _, tag := range v.([]interface{}) {
			tags = append(tags, tag.(string))
		}
		opts.Tags = tags
	}

	// Get inbound_rules
	opts.InboundRules = firewallInboundRules(d)

	// Get outbound_rules
	opts.OutboundRules = firewallOutboundRules(d)

	return opts, nil
}

func firewallInboundRules(d *schema.ResourceData) []godo.InboundRule {
	rules := make([]godo.InboundRule, 0, len(d.Get("inbound_rules").([]interface{})))
	for i, rawRule := range d.Get("inbound_rules").([]interface{}) {
		var src godo.Sources

		rule := rawRule.(map[string]interface{})
		key := fmt.Sprintf("inbound_rules.%d.source_addresses", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				src.Addresses = append(src.Addresses, v.(string))
			}
		}
		key = fmt.Sprintf("inbound_rules.%d.source_tags", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				src.Tags = append(src.Tags, v.(string))
			}
		}
		key = fmt.Sprintf("inbound_rules.%d.source_droplet_ids", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				src.DropletIDs = append(src.DropletIDs, v.(int))
			}
		}
		key = fmt.Sprintf("inbound_rules.%d.source_load_balancer_uids", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				src.LoadBalancerUIDs = append(src.LoadBalancerUIDs, v.(string))
			}
		}
		r := godo.InboundRule{
			Protocol:  rule["protocol"].(string),
			PortRange: rule["port_range"].(string),
			Sources:   &src,
		}
		rules = append(rules, r)
	}
	return rules
}

func firewallOutboundRules(d *schema.ResourceData) []godo.OutboundRule {
	rules := make([]godo.OutboundRule, 0, len(d.Get("outbound_rules").([]interface{})))
	for i, rawRule := range d.Get("outbound_rules").([]interface{}) {
		var dest godo.Destinations

		rule := rawRule.(map[string]interface{})
		key := fmt.Sprintf("outbound_rules.%d.destination_addresses", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				dest.Addresses = append(dest.Addresses, v.(string))
			}
		}
		key = fmt.Sprintf("outbound_rules.%d.destination_tags", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				dest.Tags = append(dest.Tags, v.(string))
			}
		}
		key = fmt.Sprintf("outbound_rules.%d.destination_droplet_ids", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				dest.DropletIDs = append(dest.DropletIDs, v.(int))
			}
		}
		key = fmt.Sprintf("outbound_rules.%d.destination_load_balancer_uids", i)
		if vv, ok := d.GetOk(key); ok {
			for _, v := range vv.([]interface{}) {
				dest.LoadBalancerUIDs = append(dest.LoadBalancerUIDs, v.(string))
			}
		}
		r := godo.OutboundRule{
			Protocol:     rule["protocol"].(string),
			PortRange:    rule["port_range"].(string),
			Destinations: &dest,
		}
		rules = append(rules, r)
	}
	return rules
}

func firewallPendingChanges(d *schema.ResourceData, firewall *godo.Firewall) []interface{} {
	remote := make([]interface{}, 0, len(firewall.PendingChanges))
	for _, change := range firewall.PendingChanges {
		rawChange := map[string]interface{}{
			"droplet_id": change.DropletID,
			"removing":   change.Removing,
			"status":     change.Status,
		}
		remote = append(remote, rawChange)
	}
	return remote
}

func matchFirewallInboundRules(d *schema.ResourceData, firewall *godo.Firewall) []interface{} {
	// Prepare the data.
	local := d.Get("inbound_rules").([]interface{})
	remote := make([]interface{}, 0, len(firewall.InboundRules))
	remoteMap := make(map[int]map[string]interface{})
	for _, rule := range firewall.InboundRules {
		rawRule := map[string]interface{}{
			"protocol":                  rule.Protocol,
			"port_range":                rule.PortRange,
			"source_droplet_ids":        rule.Sources.DropletIDs,
			"source_tags":               rule.Sources.Tags,
			"source_addresses":          rule.Sources.Addresses,
			"source_load_balancer_uids": rule.Sources.LoadBalancerUIDs,
		}
		remote = append(remote, rawRule)
		hash := hashFirewallRule(rule.Protocol, rule.PortRange)
		remoteMap[hash] = rawRule
	}

	// Handle special cases, both using the remote rules.
	if len(remote) == 0 || len(local) == 0 {
		return remote
	}

	// Update the local rules to only contains rules match
	// to the remote rules.
	match := make([]interface{}, 0, len(firewall.InboundRules))
	for _, rawRule := range local {
		local := rawRule.(map[string]interface{})
		protocol := local["protocol"].(string)
		portRange := local["port_range"].(string)
		hash := hashFirewallRule(protocol, portRange)
		remote, ok := remoteMap[hash]
		if !ok {
			// No entry in the remote, remove it.
			continue
		}

		// matches source lists.
		key := "source_droplet_ids"
		local[key] = matchFirewallIntLists(key, local, remote)
		keys := []string{
			"source_tags",
			"source_addresses",
			"source_load_balancer_uids",
		}
		for _, key := range keys {
			local[key] = matchFirewallStringLists(key, local, remote)
		}

		match = append(match, local)
		delete(remoteMap, hash)
	}

	// Append the remaining remote rules.
	for _, rawRule := range remoteMap {
		match = append(match, rawRule)
	}

	return match
}

func matchFirewallOutboundRules(d *schema.ResourceData, firewall *godo.Firewall) []interface{} {
	// Prepare the data.
	local := d.Get("outbound_rules").([]interface{})
	remote := make([]interface{}, 0, len(firewall.OutboundRules))
	remoteMap := make(map[int]map[string]interface{})
	for _, rule := range firewall.OutboundRules {
		rawRule := map[string]interface{}{
			"protocol":                       rule.Protocol,
			"port_range":                     rule.PortRange,
			"destination_droplet_ids":        rule.Destinations.DropletIDs,
			"destination_tags":               rule.Destinations.Tags,
			"destination_addresses":          rule.Destinations.Addresses,
			"destination_load_balancer_uids": rule.Destinations.LoadBalancerUIDs,
		}
		remote = append(remote, rawRule)
		hash := hashFirewallRule(rule.Protocol, rule.PortRange)
		remoteMap[hash] = rawRule
	}

	// Handle special cases, both using the remote rules.
	if len(remote) == 0 || len(local) == 0 {
		return remote
	}

	// Update the local rules to only contains rules match
	// to the remote rules.
	match := make([]interface{}, 0, len(firewall.OutboundRules))
	for _, rawRule := range local {
		local := rawRule.(map[string]interface{})
		protocol := local["protocol"].(string)
		portRange := local["port_range"].(string)
		hash := hashFirewallRule(protocol, portRange)
		remote, ok := remoteMap[hash]
		if !ok {
			// No entry in the remote, remove it.
			continue
		}

		// matches destination lists.
		key := "destination_droplet_ids"
		local[key] = matchFirewallIntLists(key, local, remote)
		keys := []string{
			"destination_tags",
			"destination_addresses",
			"destination_load_balancer_uids",
		}
		for _, key := range keys {
			local[key] = matchFirewallStringLists(key, local, remote)
		}

		match = append(match, local)
		delete(remoteMap, hash)
	}

	// Append the remaining remote rules.
	for _, rawRule := range remoteMap {
		match = append(match, rawRule)
	}

	return match
}

func matchFirewallIntLists(key string, local, remote map[string]interface{}) []interface{} {
	remoteSize := len(remote[key].([]int))
	remoteSet := make(map[int]bool)
	matchedList := make([]interface{}, 0, remoteSize)

	// Create a remote set out of the list for the quick comparison.
	for _, i := range remote[key].([]int) {
		remoteSet[i] = true
	}

	// Add only the item which exists in the remote list.
	for _, i := range local[key].([]interface{}) {
		if _, ok := remoteSet[i.(int)]; !ok {
			continue
		}
		matchedList = append(matchedList, i)
		delete(remoteSet, i.(int))
	}

	// Append items only exists in the remote list.
	for i := range remoteSet {
		matchedList = append(matchedList, i)
	}

	return matchedList
}

func matchFirewallStringLists(key string, local, remote map[string]interface{}) []interface{} {
	remoteSize := len(remote[key].([]string))
	remoteList := make([]interface{}, 0, remoteSize)
	matchedList := make([]interface{}, 0, remoteSize)

	// Create a remote set out of the list for the quick comparison.
	for _, s := range remote[key].([]string) {
		remoteList = append(remoteList, s)
	}
	remoteSet := schema.NewSet(schema.HashString, remoteList)

	// Add only the item which exists in the remote list.
	for _, s := range local[key].([]interface{}) {
		if !remoteSet.Contains(s.(string)) {
			continue
		}
		matchedList = append(matchedList, s)
		remoteSet.Remove(s)
	}

	// Append items only exists in the remote list.
	for _, s := range remoteSet.List() {
		matchedList = append(matchedList, s)
	}

	return matchedList
}

func hashFirewallRule(protocol, portRange string) int {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-%s", protocol, portRange))
	return hashcode.String(buf.String())
}

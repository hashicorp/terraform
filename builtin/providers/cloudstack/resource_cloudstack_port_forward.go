package cloudstack

import (
	"bytes"
	"fmt"

	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackPortForward() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackPortForwardCreate,
		Read:   resourceCloudStackPortForwardRead,
		Update: resourceCloudStackPortForwardUpdate,
		Delete: resourceCloudStackPortForwardDelete,

		Schema: map[string]*schema.Schema{
			"ipaddress": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"managed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"forward": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"private_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"public_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"virtual_machine": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceCloudStackPortForwardHash,
			},
		},
	}
}

func resourceCloudStackPortForwardCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Retrieve the ipaddress ID
	ipaddressid, e := retrieveID(cs, "ipaddress", d.Get("ipaddress").(string))
	if e != nil {
		return e.Error()
	}

	// We need to set this upfront in order to be able to save a partial state
	d.SetId(ipaddressid)

	// Create all forwards that are configured
	if rs := d.Get("forward").(*schema.Set); rs.Len() > 0 {

		// Create an empty schema.Set to hold all forwards
		forwards := &schema.Set{
			F: resourceCloudStackPortForwardHash,
		}

		for _, forward := range rs.List() {
			// Create a single forward
			err := resourceCloudStackPortForwardCreateForward(d, meta, forward.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			forwards.Add(forward)
			d.Set("forward", forwards)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackPortForwardRead(d, meta)
}

func resourceCloudStackPortForwardCreateForward(
	d *schema.ResourceData, meta interface{}, forward map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Make sure all required parameters are there
	if err := verifyPortForwardParams(d, forward); err != nil {
		return err
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", forward["virtual_machine"].(string))
	if e != nil {
		return e.Error()
	}

	vm, _, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
	if err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.Firewall.NewCreatePortForwardingRuleParams(d.Id(), forward["private_port"].(int),
		forward["protocol"].(string), forward["public_port"].(int), vm.Id)

	// Set the network ID of the default network, needed when public IP address
	// is not associated with any Guest network yet (VPC case)
	p.SetNetworkid(vm.Nic[0].Networkid)

	// Do not open the firewall automatically in any case
	p.SetOpenfirewall(false)

	r, err := cs.Firewall.CreatePortForwardingRule(p)
	if err != nil {
		return err
	}

	forward["uuid"] = r.Id

	return nil
}

func resourceCloudStackPortForwardRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get all the forwards from the running environment
	p := cs.Firewall.NewListPortForwardingRulesParams()
	p.SetIpaddressid(d.Id())
	p.SetListall(true)

	l, err := cs.Firewall.ListPortForwardingRules(p)
	if err != nil {
		return err
	}

	// Make a map of all the forwards so we can easily find a forward
	forwardMap := make(map[string]*cloudstack.PortForwardingRule, l.Count)
	for _, f := range l.PortForwardingRules {
		forwardMap[f.Id] = f
	}

	// Create an empty schema.Set to hold all forwards
	forwards := &schema.Set{
		F: resourceCloudStackPortForwardHash,
	}

	// Read all forwards that are configured
	if rs := d.Get("forward").(*schema.Set); rs.Len() > 0 {
		for _, forward := range rs.List() {
			forward := forward.(map[string]interface{})

			id, ok := forward["uuid"]
			if !ok || id.(string) == "" {
				continue
			}

			// Get the forward
			f, ok := forwardMap[id.(string)]
			if !ok {
				forward["uuid"] = ""
				continue
			}

			// Delete the known rule so only unknown rules remain in the ruleMap
			delete(forwardMap, id.(string))

			privPort, err := strconv.Atoi(f.Privateport)
			if err != nil {
				return err
			}

			pubPort, err := strconv.Atoi(f.Publicport)
			if err != nil {
				return err
			}

			// Update the values
			forward["protocol"] = f.Protocol
			forward["private_port"] = privPort
			forward["public_port"] = pubPort

			if isID(forward["virtual_machine"].(string)) {
				forward["virtual_machine"] = f.Virtualmachineid
			} else {
				forward["virtual_machine"] = f.Virtualmachinename
			}

			forwards.Add(forward)
		}
	}

	// If this is a managed resource, add all unknown forwards to dummy forwards
	managed := d.Get("managed").(bool)
	if managed && len(forwardMap) > 0 {
		for uuid := range forwardMap {
			// Make a dummy forward to hold the unknown UUID
			forward := map[string]interface{}{
				"protocol":        uuid,
				"private_port":    0,
				"public_port":     0,
				"virtual_machine": uuid,
				"uuid":            uuid,
			}

			// Add the dummy forward to the forwards set
			forwards.Add(forward)
		}
	}

	if forwards.Len() > 0 {
		d.Set("forward", forwards)
	} else if !managed {
		d.SetId("")
	}

	return nil
}

func resourceCloudStackPortForwardUpdate(d *schema.ResourceData, meta interface{}) error {
	// Check if the forward set as a whole has changed
	if d.HasChange("forward") {
		o, n := d.GetChange("forward")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old forwards and delete any obsolete ones
		for _, forward := range ors.List() {
			// Delete the forward as it no longer exists in the config
			err := resourceCloudStackPortForwardDeleteForward(d, meta, forward.(map[string]interface{}))
			if err != nil {
				return err
			}
		}

		// Make sure we save the state of the currently configured forwards
		forwards := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("forward", forwards)

		// Then loop through all the currently configured forwards and create the new ones
		for _, forward := range nrs.List() {
			err := resourceCloudStackPortForwardCreateForward(
				d, meta, forward.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			forwards.Add(forward)
			d.Set("forward", forwards)

			if err != nil {
				return err
			}
		}
	}

	return resourceCloudStackPortForwardRead(d, meta)
}

func resourceCloudStackPortForwardDelete(d *schema.ResourceData, meta interface{}) error {
	// Delete all forwards
	if rs := d.Get("forward").(*schema.Set); rs.Len() > 0 {
		for _, forward := range rs.List() {
			// Delete a single forward
			err := resourceCloudStackPortForwardDeleteForward(d, meta, forward.(map[string]interface{}))

			// We need to update this first to preserve the correct state
			d.Set("forward", rs)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceCloudStackPortForwardDeleteForward(
	d *schema.ResourceData, meta interface{}, forward map[string]interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create the parameter struct
	p := cs.Firewall.NewDeletePortForwardingRuleParams(forward["uuid"].(string))

	// Delete the forward
	if _, err := cs.Firewall.DeletePortForwardingRule(p); err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if !strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", forward["uuid"].(string))) {
			return err
		}
	}

	// Empty the UUID of this rule
	forward["uuid"] = ""

	return nil
}

func resourceCloudStackPortForwardHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf(
		"%s-%d-%d-%s",
		m["protocol"].(string),
		m["private_port"].(int),
		m["public_port"].(int),
		m["virtual_machine"].(string)))

	return hashcode.String(buf.String())
}

func verifyPortForwardParams(d *schema.ResourceData, forward map[string]interface{}) error {
	protocol := forward["protocol"].(string)
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf(
			"%s is not a valid protocol. Valid options are 'tcp' and 'udp'", protocol)
	}
	return nil
}

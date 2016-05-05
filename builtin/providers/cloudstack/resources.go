package cloudstack

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

// Define a regexp for parsing the port
var splitPorts = regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)

type retrieveError struct {
	name  string
	value string
	err   error
}

func (e *retrieveError) Error() error {
	return fmt.Errorf("Error retrieving ID of %s %s: %s", e.name, e.value, e.err)
}

func setValueOrID(d *schema.ResourceData, key string, value string, id string) {
	if cloudstack.IsID(d.Get(key).(string)) {
		// If the given id is an empty string, check if the configured value matches
		// the UnlimitedResourceID in which case we set id to UnlimitedResourceID
		if id == "" && d.Get(key).(string) == cloudstack.UnlimitedResourceID {
			id = cloudstack.UnlimitedResourceID
		}

		d.Set(key, id)
	} else {
		d.Set(key, value)
	}
}

func retrieveID(
	cs *cloudstack.CloudStackClient,
	name string,
	value string,
	opts ...cloudstack.OptionFunc) (id string, e *retrieveError) {
	// If the supplied value isn't a ID, try to retrieve the ID ourselves
	if cloudstack.IsID(value) {
		return value, nil
	}

	log.Printf("[DEBUG] Retrieving ID of %s: %s", name, value)

	var err error
	switch name {
	case "disk_offering":
		id, err = cs.DiskOffering.GetDiskOfferingID(value)
	case "virtual_machine":
		id, err = cs.VirtualMachine.GetVirtualMachineID(value, opts...)
	case "service_offering":
		id, err = cs.ServiceOffering.GetServiceOfferingID(value)
	case "network_offering":
		id, err = cs.NetworkOffering.GetNetworkOfferingID(value)
	case "project":
		id, err = cs.Project.GetProjectID(value)
	case "vpc_offering":
		id, err = cs.VPC.GetVPCOfferingID(value)
	case "vpc":
		id, err = cs.VPC.GetVPCID(value, opts...)
	case "network":
		id, err = cs.Network.GetNetworkID(value, opts...)
	case "zone":
		id, err = cs.Zone.GetZoneID(value)
	case "ip_address":
		p := cs.Address.NewListPublicIpAddressesParams()
		p.SetIpaddress(value)
		for _, fn := range opts {
			if e := fn(cs, p); e != nil {
				err = e
				break
			}
		}
		l, e := cs.Address.ListPublicIpAddresses(p)
		if e != nil {
			err = e
			break
		}
		if l.Count == 1 {
			id = l.PublicIpAddresses[0].Id
			break
		}
		err = fmt.Errorf("Could not find ID of IP address: %s", value)
	case "os_type":
		p := cs.GuestOS.NewListOsTypesParams()
		p.SetDescription(value)
		l, e := cs.GuestOS.ListOsTypes(p)
		if e != nil {
			err = e
			break
		}
		if l.Count == 1 {
			id = l.OsTypes[0].Id
			break
		}
		err = fmt.Errorf("Could not find ID of OS Type: %s", value)
	default:
		return id, &retrieveError{name: name, value: value,
			err: fmt.Errorf("Unknown request: %s", name)}
	}

	if err != nil {
		return id, &retrieveError{name: name, value: value, err: err}
	}

	return id, nil
}

func retrieveTemplateID(cs *cloudstack.CloudStackClient, zoneid, value string) (id string, e *retrieveError) {
	// If the supplied value isn't a ID, try to retrieve the ID ourselves
	if cloudstack.IsID(value) {
		return value, nil
	}

	log.Printf("[DEBUG] Retrieving ID of template: %s", value)

	id, err := cs.Template.GetTemplateID(value, "executable", zoneid)
	if err != nil {
		return id, &retrieveError{name: "template", value: value, err: err}
	}

	return id, nil
}

// RetryFunc is the function retried n times
type RetryFunc func() (interface{}, error)

// Retry is a wrapper around a RetryFunc that will retry a function
// n times or until it succeeds.
func Retry(n int, f RetryFunc) (interface{}, error) {
	var lastErr error

	for i := 0; i < n; i++ {
		r, err := f()
		if err == nil || err == cloudstack.AsyncTimeoutErr {
			return r, err
		}

		lastErr = err
		time.Sleep(30 * time.Second)
	}

	return nil, lastErr
}

// This is a temporary helper function to support both the new
// cidr_list and the deprecated source_cidr parameter
func retrieveCidrList(rule map[string]interface{}) []string {
	sourceCidr := rule["source_cidr"].(string)
	if sourceCidr != "" {
		return []string{sourceCidr}
	}

	var cidrList []string
	for _, cidr := range rule["cidr_list"].(*schema.Set).List() {
		cidrList = append(cidrList, cidr.(string))
	}

	return cidrList
}

// This is a temporary helper function to support both the new
// cidr_list and the deprecated source_cidr parameter
func setCidrList(rule map[string]interface{}, cidrList string) {
	sourceCidr := rule["source_cidr"].(string)
	if sourceCidr != "" {
		rule["source_cidr"] = cidrList
		return
	}

	cidrs := &schema.Set{F: schema.HashString}
	for _, cidr := range strings.Split(cidrList, ",") {
		cidrs.Add(cidr)
	}

	rule["cidr_list"] = cidrs
}

// If there is a project supplied, we retrieve and set the project id
func setProjectid(p cloudstack.ProjectIDSetter, cs *cloudstack.CloudStackClient, d *schema.ResourceData) error {
	if project, ok := d.GetOk("project"); ok {
		projectid, e := retrieveID(cs, "project", project.(string))
		if e != nil {
			return e.Error()
		}
		p.SetProjectid(projectid)
	}

	return nil
}

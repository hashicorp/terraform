package cloudstack

import (
	"fmt"
	"log"
	"regexp"

	"github.com/xanzy/go-cloudstack/cloudstack"
)

type retrieveError struct {
	name  string
	value string
	err   error
}

func (e *retrieveError) Error() error {
	return fmt.Errorf("Error retrieving UUID of %s %s: %s", e.name, e.value, e.err)
}

func retrieveUUID(cs *cloudstack.CloudStackClient, name, value string) (uuid string, e *retrieveError) {
	// If the supplied value isn't a UUID, try to retrieve the UUID ourselves
	if isUUID(value) {
		return value, nil
	}

	log.Printf("[DEBUG] Retrieving UUID of %s: %s", name, value)

	var err error
	switch name {
	case "disk_offering":
		uuid, err = cs.DiskOffering.GetDiskOfferingID(value)
	case "virtual_machine":
		uuid, err = cs.VirtualMachine.GetVirtualMachineID(value)
	case "service_offering":
		uuid, err = cs.ServiceOffering.GetServiceOfferingID(value)
	case "network_offering":
		uuid, err = cs.NetworkOffering.GetNetworkOfferingID(value)
	case "vpc_offering":
		uuid, err = cs.VPC.GetVPCOfferingID(value)
	case "vpc":
		uuid, err = cs.VPC.GetVPCID(value)
	case "network":
		uuid, err = cs.Network.GetNetworkID(value)
	case "zone":
		uuid, err = cs.Zone.GetZoneID(value)
	case "ipaddress":
		p := cs.Address.NewListPublicIpAddressesParams()
		p.SetIpaddress(value)
		l, e := cs.Address.ListPublicIpAddresses(p)
		if e != nil {
			err = e
			break
		}
		if l.Count == 1 {
			uuid = l.PublicIpAddresses[0].Id
			break
		}
		err = fmt.Errorf("Could not find UUID of IP address: %s", value)
	case "os_type":
		p := cs.GuestOS.NewListOsTypesParams()
		p.SetDescription(value)
		l, e := cs.GuestOS.ListOsTypes(p)
		if e != nil {
			err = e
			break
		}
		if l.Count == 1 {
			uuid = l.OsTypes[0].Id
			break
		}
		err = fmt.Errorf("Could not find UUID of OS Type: %s", value)
	default:
		return uuid, &retrieveError{name: name, value: value,
			err: fmt.Errorf("Unknown request: %s", name)}
	}

	if err != nil {
		return uuid, &retrieveError{name: name, value: value, err: err}
	}

	return uuid, nil
}

func retrieveTemplateUUID(cs *cloudstack.CloudStackClient, zoneid, value string) (uuid string, e *retrieveError) {
	// If the supplied value isn't a UUID, try to retrieve the UUID ourselves
	if isUUID(value) {
		return value, nil
	}

	log.Printf("[DEBUG] Retrieving UUID of template: %s", value)

	uuid, err := cs.Template.GetTemplateID(value, "executable", zoneid)
	if err != nil {
		return uuid, &retrieveError{name: "template", value: value, err: err}
	}

	return uuid, nil
}

func isUUID(s string) bool {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return re.MatchString(s)
}

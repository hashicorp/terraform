/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	types "github.com/ukcloud/govcloudair/types/v56"
)

type VM struct {
	VM *types.VM
	c  *Client
}

func NewVM(c *Client) *VM {
	return &VM{
		VM: new(types.VM),
		c:  c,
	}
}

func (v *VM) GetStatus() (string, error) {
	err := v.Refresh()
	if err != nil {
		return "", fmt.Errorf("error refreshing VM: %v", err)
	}
	return types.VAppStatuses[v.VM.Status], nil
}

func (v *VM) Refresh() error {

	if v.VM.HREF == "" {
		return fmt.Errorf("cannot refresh VM, Object is empty")
	}

	u, _ := url.ParseRequestURI(v.VM.HREF)

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retrieving task: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	v.VM = &types.VM{}

	if err = decodeBody(resp, v.VM); err != nil {
		return fmt.Errorf("error decoding task response VM: %s", err)
	}

	// The request was successful
	return nil
}

func (c *VCDClient) FindVMByHREF(vmhref string) (VM, error) {

	u, err := url.ParseRequestURI(vmhref)

	if err != nil {
		return VM{}, fmt.Errorf("error decoding vm HREF: %s", err)
	}

	// Querying the VApp
	req := c.Client.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return VM{}, fmt.Errorf("error retrieving VM: %s", err)
	}

	newvm := NewVM(&c.Client)

	if err = decodeBody(resp, newvm.VM); err != nil {
		return VM{}, fmt.Errorf("error decoding VM response: %s", err)
	}

	return *newvm, nil

}

func (v *VM) PowerOn() (Task, error) {

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/power/action/powerOn"

	req := v.c.NewRequest(map[string]string{}, "POST", *s, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error powering on VM: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (v *VM) PowerOff() (Task, error) {

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/power/action/powerOff"

	req := v.c.NewRequest(map[string]string{}, "POST", *s, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error powering off VM: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (v *VM) ChangeCPUcount(size int) (Task, error) {

	err := v.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("error refreshing VM before running customization: %v", err)
	}

	newcpu := &types.OVFItem{
		XmlnsRasd:       "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData",
		XmlnsVCloud:     "http://www.vmware.com/vcloud/v1.5",
		XmlnsXsi:        "http://www.w3.org/2001/XMLSchema-instance",
		VCloudHREF:      v.VM.HREF + "/virtualHardwareSection/cpu",
		VCloudType:      "application/vnd.vmware.vcloud.rasdItem+xml",
		AllocationUnits: "hertz * 10^6",
		Description:     "Number of Virtual CPUs",
		ElementName:     strconv.Itoa(size) + " virtual CPU(s)",
		InstanceID:      4,
		Reservation:     0,
		ResourceType:    3,
		VirtualQuantity: size,
		Weight:          0,
		Link: &types.Link{
			HREF: v.VM.HREF + "/virtualHardwareSection/cpu",
			Rel:  "edit",
			Type: "application/vnd.vmware.vcloud.rasdItem+xml",
		},
	}

	output, err := xml.MarshalIndent(newcpu, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/virtualHardwareSection/cpu"

	req := v.c.NewRequest(map[string]string{}, "PUT", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.rasdItem+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error customizing VM: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (v *VM) ChangeMemorySize(size int) (Task, error) {

	err := v.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("error refreshing VM before running customization: %v", err)
	}

	newmem := &types.OVFItem{
		XmlnsRasd:       "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData",
		XmlnsVCloud:     "http://www.vmware.com/vcloud/v1.5",
		XmlnsXsi:        "http://www.w3.org/2001/XMLSchema-instance",
		VCloudHREF:      v.VM.HREF + "/virtualHardwareSection/memory",
		VCloudType:      "application/vnd.vmware.vcloud.rasdItem+xml",
		AllocationUnits: "byte * 2^20",
		Description:     "Memory Size",
		ElementName:     strconv.Itoa(size) + " MB of memory",
		InstanceID:      5,
		Reservation:     0,
		ResourceType:    4,
		VirtualQuantity: size,
		Weight:          0,
		Link: &types.Link{
			HREF: v.VM.HREF + "/virtualHardwareSection/memory",
			Rel:  "edit",
			Type: "application/vnd.vmware.vcloud.rasdItem+xml",
		},
	}

	output, err := xml.MarshalIndent(newmem, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/virtualHardwareSection/memory"

	req := v.c.NewRequest(map[string]string{}, "PUT", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.rasdItem+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error customizing VM: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (v *VM) ChangeNetworkConfig(network, ip string) (Task, error) {
	err := v.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("error refreshing VM before running customization: %v", err)
	}

	// Determine what type of address is requested for the vApp
	ipAllocationMode := "NONE"
	ipAddress := "Any"

	// TODO: Review current behaviour of using DHCP when left blank
	if ip == "" || ip == "dhcp" {
		ipAllocationMode = "DHCP"
	} else if ip == "allocated" {
		ipAllocationMode = "POOL"
	} else if ip == "none" {
		ipAllocationMode = "NONE"
	} else if ip != "" {
		ipAllocationMode = "MANUAL"
		// TODO: Check a valid IP has been given
		ipAddress = ip
	}

	networkConnection := &types.NetworkConnection{
		Network:                 network,
		NeedsCustomization:      true,
		NetworkConnectionIndex:  0,
		IPAddress:               ipAddress,
		IsConnected:             true,
		IPAddressAllocationMode: ipAllocationMode,
	}

	newnetwork := &types.NetworkConnectionSection{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		Ovf:   "http://schemas.dmtf.org/ovf/envelope/1",
		Info:  "Specifies the available VM network connections",
		PrimaryNetworkConnectionIndex: 0,
		NetworkConnection:             networkConnection,
	}

	output, err := xml.MarshalIndent(newnetwork, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	log.Printf("[DEBUG] NetworkXML: %s", output)

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/networkConnectionSection/"

	req := v.c.NewRequest(map[string]string{}, "PUT", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.networkConnectionSection+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error customizing VM Network: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

func (v *VM) RunCustomizationScript(computername, script string) (Task, error) {
	return v.Customize(computername, script, false)
}

func (v *VM) Customize(computername, script string, changeSid bool) (Task, error) {
	err := v.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("error refreshing VM before running customization: %v", err)
	}

	vu := &types.GuestCustomizationSection{
		Ovf:   "http://schemas.dmtf.org/ovf/envelope/1",
		Xsi:   "http://www.w3.org/2001/XMLSchema-instance",
		Xmlns: "http://www.vmware.com/vcloud/v1.5",

		HREF:                v.VM.HREF,
		Type:                "application/vnd.vmware.vcloud.guestCustomizationSection+xml",
		Info:                "Specifies Guest OS Customization Settings",
		Enabled:             true,
		ComputerName:        computername,
		CustomizationScript: script,
		ChangeSid:           false,
	}

	output, err := xml.MarshalIndent(vu, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	log.Printf("[DEBUG] VCD Client configuration: %s", output)

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/guestCustomizationSection/"

	req := v.c.NewRequest(map[string]string{}, "PUT", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.guestCustomizationSection+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error customizing VM: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

func (v *VM) Undeploy() (Task, error) {

	vu := &types.UndeployVAppParams{
		Xmlns:               "http://www.vmware.com/vcloud/v1.5",
		UndeployPowerAction: "powerOff",
	}

	output, err := xml.MarshalIndent(vu, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(v.VM.HREF)
	s.Path += "/action/undeploy"

	req := v.c.NewRequest(map[string]string{}, "POST", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.undeployVAppParams+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error undeploy vApp: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

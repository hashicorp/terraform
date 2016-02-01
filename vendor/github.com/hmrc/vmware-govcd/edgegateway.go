/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	types "github.com/hmrc/vmware-govcd/types/v56"
)

type EdgeGateway struct {
	EdgeGateway *types.EdgeGateway
	c           *Client
}

func NewEdgeGateway(c *Client) *EdgeGateway {
	return &EdgeGateway{
		EdgeGateway: new(types.EdgeGateway),
		c:           c,
	}
}

func (e *EdgeGateway) AddDhcpPool(network *types.OrgVDCNetwork, dhcppool []interface{}) (Task, error) {
	newedgeconfig := e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration
	log.Printf("[DEBUG] EDGE GATEWAY: %#v", newedgeconfig)
	log.Printf("[DEBUG] EDGE GATEWAY SERVICE: %#v", newedgeconfig.GatewayDhcpService)
	newdchpservice := &types.GatewayDhcpService{}
	if newedgeconfig.GatewayDhcpService == nil {
		newdchpservice.IsEnabled = true
	} else {
		newdchpservice.IsEnabled = newedgeconfig.GatewayDhcpService.IsEnabled

		for _, v := range newedgeconfig.GatewayDhcpService.Pool {

			// Kludgy IF to avoid deleting DNAT rules not created by us.
			// If matches, let's skip it and continue the loop
			if v.Network.HREF == network.HREF {
				continue
			}

			newdchpservice.Pool = append(newdchpservice.Pool, v)
		}
	}

	for _, v := range dhcppool {
		data := v.(map[string]interface{})

		dhcprule := &types.DhcpPoolService{
			IsEnabled: true,
			Network: &types.Reference{
				HREF: network.HREF,
				Name: network.Name,
			},
			DefaultLeaseTime: 3600,
			MaxLeaseTime:     7200,
			LowIPAddress:     data["start_address"].(string),
			HighIPAddress:    data["end_address"].(string),
		}
		newdchpservice.Pool = append(newdchpservice.Pool, dhcprule)
	}

	newRules := &types.EdgeGatewayServiceConfiguration{
		Xmlns:              "http://www.vmware.com/vcloud/v1.5",
		GatewayDhcpService: newdchpservice,
	}

	output, err := xml.MarshalIndent(newRules, "  ", "    ")
	if err != nil {
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	var resp *http.Response
	for {
		b := bytes.NewBufferString(xml.Header + string(output))

		s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
		s.Path += "/action/configureServices"

		req := e.c.NewRequest(map[string]string{}, "POST", *s, b)
		log.Printf("[DEBUG] POSTING TO URL: %s", s.Path)
		log.Printf("[DEBUG] XML TO SEND:\n%s", b)

		req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

		resp, err = checkResp(e.c.Http.Do(req))
		if err != nil {
			if v, _ := regexp.MatchString("is busy completing an operation.$", err.Error()); v {
				time.Sleep(3 * time.Second)
				continue
			}
			return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
		}
		break
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (e *EdgeGateway) RemoveNATMapping(nattype, externalIP, internalIP, port string) (Task, error) {
	// Find uplink interface
	var uplink types.Reference
	for _, gi := range e.EdgeGateway.Configuration.GatewayInterfaces.GatewayInterface {
		if gi.InterfaceType != "uplink" {
			continue
		}
		uplink = *gi.Network
	}

	newedgeconfig := e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration

	// Take care of the NAT service
	newnatservice := &types.NatService{}

	newnatservice.IsEnabled = newedgeconfig.NatService.IsEnabled
	newnatservice.NatType = newedgeconfig.NatService.NatType
	newnatservice.Policy = newedgeconfig.NatService.Policy
	newnatservice.ExternalIP = newedgeconfig.NatService.ExternalIP

	for _, v := range newedgeconfig.NatService.NatRule {

		// Kludgy IF to avoid deleting DNAT rules not created by us.
		// If matches, let's skip it and continue the loop
		if v.RuleType == nattype &&
			v.GatewayNatRule.OriginalIP == externalIP &&
			v.GatewayNatRule.OriginalPort == port &&
			v.GatewayNatRule.Interface.HREF == uplink.HREF {
			log.Printf("[DEBUG] REMOVING %s Rule: %#v", v.RuleType, v.GatewayNatRule)
			continue
		}
		log.Printf("[DEBUG] KEEPING %s Rule: %#v", v.RuleType, v.GatewayNatRule)
		newnatservice.NatRule = append(newnatservice.NatRule, v)
	}

	newedgeconfig.NatService = newnatservice

	newRules := &types.EdgeGatewayServiceConfiguration{
		Xmlns:      "http://www.vmware.com/vcloud/v1.5",
		NatService: newnatservice,
	}

	output, err := xml.MarshalIndent(newRules, "  ", "    ")
	if err != nil {
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
	s.Path += "/action/configureServices"

	req := e.c.NewRequest(map[string]string{}, "POST", *s, b)
	log.Printf("[DEBUG] POSTING TO URL: %s", s.Path)
	log.Printf("[DEBUG] XML TO SEND:\n%s", b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

	resp, err := checkResp(e.c.Http.Do(req))
	if err != nil {
		log.Printf("[DEBUG] Error is: %#v", err)
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (e *EdgeGateway) AddNATMapping(nattype, externalIP, internalIP, port string) (Task, error) {
	// Find uplink interface
	var uplink types.Reference
	for _, gi := range e.EdgeGateway.Configuration.GatewayInterfaces.GatewayInterface {
		if gi.InterfaceType != "uplink" {
			continue
		}
		uplink = *gi.Network
	}

	newedgeconfig := e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration

	// Take care of the NAT service
	newnatservice := &types.NatService{}

	if newedgeconfig.NatService == nil {
		newnatservice.IsEnabled = true
	} else {
		newnatservice.IsEnabled = newedgeconfig.NatService.IsEnabled
		newnatservice.NatType = newedgeconfig.NatService.NatType
		newnatservice.Policy = newedgeconfig.NatService.Policy
		newnatservice.ExternalIP = newedgeconfig.NatService.ExternalIP

		for _, v := range newedgeconfig.NatService.NatRule {

			// Kludgy IF to avoid deleting DNAT rules not created by us.
			// If matches, let's skip it and continue the loop
			if v.RuleType == nattype &&
				v.GatewayNatRule.OriginalIP == externalIP &&
				v.GatewayNatRule.OriginalPort == port &&
				v.GatewayNatRule.TranslatedIP == internalIP &&
				v.GatewayNatRule.Interface.HREF == uplink.HREF {
				continue
			}

			newnatservice.NatRule = append(newnatservice.NatRule, v)
		}
	}

	//add rule
	natRule := &types.NatRule{
		RuleType:  nattype,
		IsEnabled: true,
		GatewayNatRule: &types.GatewayNatRule{
			Interface: &types.Reference{
				HREF: uplink.HREF,
			},
			OriginalIP:     externalIP,
			OriginalPort:   port,
			TranslatedIP:   internalIP,
			TranslatedPort: port,
			Protocol:       "tcp",
		},
	}
	newnatservice.NatRule = append(newnatservice.NatRule, natRule)

	newedgeconfig.NatService = newnatservice

	newRules := &types.EdgeGatewayServiceConfiguration{
		Xmlns:      "http://www.vmware.com/vcloud/v1.5",
		NatService: newnatservice,
	}

	output, err := xml.MarshalIndent(newRules, "  ", "    ")
	if err != nil {
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
	s.Path += "/action/configureServices"

	req := e.c.NewRequest(map[string]string{}, "POST", *s, b)
	log.Printf("[DEBUG] POSTING TO URL: %s", s.Path)
	log.Printf("[DEBUG] XML TO SEND:\n%s", b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

	resp, err := checkResp(e.c.Http.Do(req))
	if err != nil {
		log.Printf("[DEBUG] Error is: %#v", err)
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (e *EdgeGateway) CreateFirewallRules(defaultAction string, rules []*types.FirewallRule) (Task, error) {
	err := e.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("error: %v\n", err)
	}

	newRules := &types.EdgeGatewayServiceConfiguration{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		FirewallService: &types.FirewallService{
			IsEnabled:        true,
			DefaultAction:    defaultAction,
			LogDefaultAction: true,
			FirewallRule:     rules,
		},
	}

	output, err := xml.MarshalIndent(newRules, "  ", "    ")
	if err != nil {
		return Task{}, fmt.Errorf("error: %v\n", err)
	}

	var resp *http.Response
	for {
		b := bytes.NewBufferString(xml.Header + string(output))

		s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
		s.Path += "/action/configureServices"

		req := e.c.NewRequest(map[string]string{}, "POST", *s, b)
		log.Printf("[DEBUG] POSTING TO URL: %s", s.Path)
		log.Printf("[DEBUG] XML TO SEND:\n%s", b)

		req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

		resp, err = checkResp(e.c.Http.Do(req))
		if err != nil {
			if v, _ := regexp.MatchString("is busy completing an operation.$", err.Error()); v {
				time.Sleep(3 * time.Second)
				continue
			}
			return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
		}
		break
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

func (e *EdgeGateway) Refresh() error {

	if e.EdgeGateway == nil {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(e.EdgeGateway.HREF)

	req := e.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(e.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retreiving Edge Gateway: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	e.EdgeGateway = &types.EdgeGateway{}

	if err = decodeBody(resp, e.EdgeGateway); err != nil {
		return fmt.Errorf("error decoding Edge Gateway response: %s", err)
	}

	// The request was successful
	return nil
}

func (e *EdgeGateway) Remove1to1Mapping(internal, external string) (Task, error) {

	// Refresh EdgeGateway rules
	err := e.Refresh()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	var uplinkif string
	for _, gifs := range e.EdgeGateway.Configuration.GatewayInterfaces.GatewayInterface {
		if gifs.InterfaceType == "uplink" {
			uplinkif = gifs.Network.HREF
		}
	}

	newedgeconfig := e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration

	// Take care of the NAT service
	newnatservice := &types.NatService{}

	// Copy over the NAT configuration
	newnatservice.IsEnabled = newedgeconfig.NatService.IsEnabled
	newnatservice.NatType = newedgeconfig.NatService.NatType
	newnatservice.Policy = newedgeconfig.NatService.Policy
	newnatservice.ExternalIP = newedgeconfig.NatService.ExternalIP

	for k, v := range newedgeconfig.NatService.NatRule {

		// Kludgy IF to avoid deleting DNAT rules not created by us.
		// If matches, let's skip it and continue the loop
		if v.RuleType == "DNAT" &&
			v.GatewayNatRule.OriginalIP == external &&
			v.GatewayNatRule.TranslatedIP == internal &&
			v.GatewayNatRule.OriginalPort == "any" &&
			v.GatewayNatRule.TranslatedPort == "any" &&
			v.GatewayNatRule.Protocol == "any" &&
			v.GatewayNatRule.Interface.HREF == uplinkif {
			continue
		}

		// Kludgy IF to avoid deleting SNAT rules not created by us.
		// If matches, let's skip it and continue the loop
		if v.RuleType == "SNAT" &&
			v.GatewayNatRule.OriginalIP == internal &&
			v.GatewayNatRule.TranslatedIP == external &&
			v.GatewayNatRule.Interface.HREF == uplinkif {
			continue
		}

		// If doesn't match the above IFs, it's something we need to preserve,
		// let's add it to the new NatService struct
		newnatservice.NatRule = append(newnatservice.NatRule, newedgeconfig.NatService.NatRule[k])

	}

	// Fill the new NatService Section
	newedgeconfig.NatService = newnatservice

	// Take care of the Firewall service
	newfwservice := &types.FirewallService{}

	// Copy over the firewall configuration
	newfwservice.IsEnabled = newedgeconfig.FirewallService.IsEnabled
	newfwservice.DefaultAction = newedgeconfig.FirewallService.DefaultAction
	newfwservice.LogDefaultAction = newedgeconfig.FirewallService.LogDefaultAction

	for k, v := range newedgeconfig.FirewallService.FirewallRule {

		// Kludgy IF to avoid deleting inbound FW rules not created by us.
		// If matches, let's skip it and continue the loop
		if v.Policy == "allow" &&
			v.Protocols.Any == true &&
			v.DestinationPortRange == "Any" &&
			v.SourcePortRange == "Any" &&
			v.SourceIP == "Any" &&
			v.DestinationIP == external {
			continue
		}

		// Kludgy IF to avoid deleting outbound FW rules not created by us.
		// If matches, let's skip it and continue the loop
		if v.Policy == "allow" &&
			v.Protocols.Any == true &&
			v.DestinationPortRange == "Any" &&
			v.SourcePortRange == "Any" &&
			v.SourceIP == internal &&
			v.DestinationIP == "Any" {
			continue
		}

		// If doesn't match the above IFs, it's something we need to preserve,
		// let's add it to the new FirewallService struct
		newfwservice.FirewallRule = append(newfwservice.FirewallRule, newedgeconfig.FirewallService.FirewallRule[k])

	}

	// Fill the new FirewallService Section
	newedgeconfig.FirewallService = newfwservice

	// Fix
	newedgeconfig.NatService.IsEnabled = true

	output, err := xml.MarshalIndent(newedgeconfig, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
	s.Path += "/action/configureServices"

	req := e.c.NewRequest(map[string]string{}, "POST", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

	resp, err := checkResp(e.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

func (e *EdgeGateway) Create1to1Mapping(internal, external, description string) (Task, error) {

	// Refresh EdgeGateway rules
	err := e.Refresh()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	var uplinkif string
	for _, gifs := range e.EdgeGateway.Configuration.GatewayInterfaces.GatewayInterface {
		if gifs.InterfaceType == "uplink" {
			uplinkif = gifs.Network.HREF
		}
	}

	newedgeconfig := e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration

	snat := &types.NatRule{
		Description: description,
		RuleType:    "SNAT",
		IsEnabled:   true,
		GatewayNatRule: &types.GatewayNatRule{
			Interface: &types.Reference{
				HREF: uplinkif,
			},
			OriginalIP:   internal,
			TranslatedIP: external,
			Protocol:     "any",
		},
	}

	newedgeconfig.NatService.NatRule = append(newedgeconfig.NatService.NatRule, snat)

	dnat := &types.NatRule{
		Description: description,
		RuleType:    "DNAT",
		IsEnabled:   true,
		GatewayNatRule: &types.GatewayNatRule{
			Interface: &types.Reference{
				HREF: uplinkif,
			},
			OriginalIP:     external,
			OriginalPort:   "any",
			TranslatedIP:   internal,
			TranslatedPort: "any",
			Protocol:       "any",
		},
	}

	newedgeconfig.NatService.NatRule = append(newedgeconfig.NatService.NatRule, dnat)

	fwin := &types.FirewallRule{
		Description: description,
		IsEnabled:   true,
		Policy:      "allow",
		Protocols: &types.FirewallRuleProtocols{
			Any: true,
		},
		DestinationPortRange: "Any",
		DestinationIP:        external,
		SourcePortRange:      "Any",
		SourceIP:             "Any",
		EnableLogging:        false,
	}

	newedgeconfig.FirewallService.FirewallRule = append(newedgeconfig.FirewallService.FirewallRule, fwin)

	fwout := &types.FirewallRule{
		Description: description,
		IsEnabled:   true,
		Policy:      "allow",
		Protocols: &types.FirewallRuleProtocols{
			Any: true,
		},
		DestinationPortRange: "Any",
		DestinationIP:        "Any",
		SourcePortRange:      "Any",
		SourceIP:             internal,
		EnableLogging:        false,
	}

	newedgeconfig.FirewallService.FirewallRule = append(newedgeconfig.FirewallService.FirewallRule, fwout)

	output, err := xml.MarshalIndent(newedgeconfig, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s, _ := url.ParseRequestURI(e.EdgeGateway.HREF)
	s.Path += "/action/configureServices"

	req := e.c.NewRequest(map[string]string{}, "POST", *s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml")

	resp, err := checkResp(e.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error reconfiguring Edge Gateway: %s", err)
	}

	task := NewTask(e.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil

}

/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/url"
	"strings"

	types "github.com/ukcloud/govcloudair/types/v56"
)

type Vdc struct {
	Vdc *types.Vdc
	c   *Client
}

func NewVdc(c *Client) *Vdc {
	return &Vdc{
		Vdc: new(types.Vdc),
		c:   c,
	}
}

func (c *Client) retrieveVDC() (Vdc, error) {

	req := c.NewRequest(map[string]string{}, "GET", c.VCDVDCHREF, nil)

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return Vdc{}, fmt.Errorf("error retreiving vdc: %s", err)
	}

	vdc := NewVdc(c)

	if err = decodeBody(resp, vdc.Vdc); err != nil {
		return Vdc{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// The request was successful
	return *vdc, nil
}

func (v *Vdc) Refresh() error {

	if v.Vdc.HREF == "" {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(v.Vdc.HREF)

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retreiving Edge Gateway: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	unmarshalledVdc := &types.Vdc{}

	if err = decodeBody(resp, unmarshalledVdc); err != nil {
		return fmt.Errorf("error decoding vdc response: %s", err)
	}

	v.Vdc = unmarshalledVdc

	// The request was successful
	return nil
}

func (v *Vdc) FindVDCNetwork(network string) (OrgVDCNetwork, error) {

	for _, an := range v.Vdc.AvailableNetworks {
		for _, n := range an.Network {
			if n.Name == network {
				u, err := url.ParseRequestURI(n.HREF)
				if err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error retreiving orgvdcnetwork: %s", err)
				}

				orgnet := NewOrgVDCNetwork(v.c)

				if err = decodeBody(resp, orgnet.OrgVDCNetwork); err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
				}

				// The request was successful
				return *orgnet, nil

			}
		}
	}

	return OrgVDCNetwork{}, fmt.Errorf("can't find VDC Network: %s", network)
}

func (v *Vdc) FindStorageProfile(storage_profile string) (types.Reference, error) {

	for _, an := range v.Vdc.VdcStorageProfiles {
		for _, n := range an.VdcStorageProfile {
			if n.Name == storage_profile {
				return *n, nil
			}
		}
		return types.Reference{}, fmt.Errorf("can't find VDC Storage_profile: %s", storage_profile)
	}
	return types.Reference{}, fmt.Errorf("can't find any VDC Storage_profiles")
}

func (v *Vdc) GetDefaultStorageProfile(storage_profiles *types.QueryResultRecordsType) (types.Reference, error) {

	for _, n := range storage_profiles.OrgVdcStorageProfileRecord {
		if n.IsDefaultStorageProfile {
			storage_profile_reference := types.Reference{HREF: n.HREF, Name: n.Name}
			return storage_profile_reference, nil
		}
	}
	return types.Reference{}, fmt.Errorf("can't find Default VDC Storage_profile")
}

// Doesn't work with vCloud API 5.5, only vCloud Air
func (v *Vdc) GetVDCOrg() (Org, error) {

	for _, av := range v.Vdc.Link {
		if av.Rel == "up" && av.Type == "application/vnd.vmware.vcloud.org+xml" {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return Org{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return Org{}, fmt.Errorf("error retreiving org: %s", err)
			}

			org := NewOrg(v.c)

			if err = decodeBody(resp, org.Org); err != nil {
				return Org{}, fmt.Errorf("error decoding org response: %s", err)
			}

			// The request was successful
			return *org, nil

		}
	}
	return Org{}, fmt.Errorf("can't find VDC Org")
}

func (v *Vdc) FindEdgeGateway(edgegateway string) (EdgeGateway, error) {

	for _, av := range v.Vdc.Link {
		if av.Rel == "edgeGateways" && av.Type == "application/vnd.vmware.vcloud.query.records+xml" {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			// Querying the Result list
			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway records: %s", err)
			}

			query := new(types.QueryResultEdgeGatewayRecordsType)

			if err = decodeBody(resp, query); err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
			}

			u, err = url.ParseRequestURI(query.EdgeGatewayRecord.HREF)
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
			}

			// Querying the Result list
			req = v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err = checkResp(v.c.Http.Do(req))
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway: %s", err)
			}

			edge := NewEdgeGateway(v.c)

			if err = decodeBody(resp, edge.EdgeGateway); err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway response: %s", err)
			}

			return *edge, nil

		}
	}
	return EdgeGateway{}, fmt.Errorf("can't find Edge Gateway")

}

func (v *Vdc) FindVAppByName(vapp string) (VApp, error) {

	err := v.Refresh()
	if err != nil {
		return VApp{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	for _, resents := range v.Vdc.ResourceEntities {
		for _, resent := range resents.ResourceEntity {

			if resent.Name == vapp && resent.Type == "application/vnd.vmware.vcloud.vApp+xml" {

				u, err := url.ParseRequestURI(resent.HREF)

				if err != nil {
					return VApp{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				// Querying the VApp
				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return VApp{}, fmt.Errorf("error retrieving vApp: %s", err)
				}

				newvapp := NewVApp(v.c)

				if err = decodeBody(resp, newvapp.VApp); err != nil {
					return VApp{}, fmt.Errorf("error decoding vApp response: %s", err.Error())
				}

				return *newvapp, nil

			}
		}
	}
	return VApp{}, fmt.Errorf("can't find vApp: %s", vapp)
}

func (v *Vdc) FindVMByName(vapp VApp, vm string) (VM, error) {

	err := v.Refresh()
	if err != nil {
		return VM{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	for _, child := range vapp.VApp.Children.VM {

		if child.Name == vm {

			u, err := url.ParseRequestURI(child.HREF)

			if err != nil {
				return VM{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			// Querying the VApp
			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return VM{}, fmt.Errorf("error retrieving vm: %s", err)
			}

			newvm := NewVM(v.c)

			//body, err := ioutil.ReadAll(resp.Body)
			//fmt.Println(string(body))

			if err = decodeBody(resp, newvm.VM); err != nil {
				return VM{}, fmt.Errorf("error decoding vm response: %s", err.Error())
			}

			return *newvm, nil

		}

	}
	return VM{}, fmt.Errorf("can't find vm: %s", vm)
}

func (v *Vdc) FindVAppByID(vappid string) (VApp, error) {

	// Horrible hack to fetch a vapp with its id.
	// urn:vcloud:vapp:00000000-0000-0000-0000-000000000000

	err := v.Refresh()
	if err != nil {
		return VApp{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	urnslice := strings.SplitAfter(vappid, ":")
	urnid := urnslice[len(urnslice) - 1]

	for _, resents := range v.Vdc.ResourceEntities {
		for _, resent := range resents.ResourceEntity {

			hrefslice := strings.SplitAfter(resent.HREF, "/")
			hrefslice = strings.SplitAfter(hrefslice[len(hrefslice) - 1], "-")
			res := strings.Join(hrefslice[1:], "")

			if res == urnid && resent.Type == "application/vnd.vmware.vcloud.vApp+xml" {

				u, err := url.ParseRequestURI(resent.HREF)

				if err != nil {
					return VApp{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				// Querying the VApp
				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return VApp{}, fmt.Errorf("error retrieving vApp: %s", err)
				}

				newvapp := NewVApp(v.c)

				if err = decodeBody(resp, newvapp.VApp); err != nil {
					return VApp{}, fmt.Errorf("error decoding vApp response: %s", err)
				}

				return *newvapp, nil

			}
		}
	}
	return VApp{}, fmt.Errorf("can't find vApp")

}

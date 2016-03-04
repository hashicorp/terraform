//
// Copyright 2014, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cloudstack

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type AddBigSwitchVnsDeviceParams struct {
	p map[string]interface{}
}

func (p *AddBigSwitchVnsDeviceParams) toURLValues() url.Values {
	u := url.Values{}
	if p.p == nil {
		return u
	}
	if v, found := p.p["hostname"]; found {
		u.Set("hostname", v.(string))
	}
	if v, found := p.p["physicalnetworkid"]; found {
		u.Set("physicalnetworkid", v.(string))
	}
	return u
}

func (p *AddBigSwitchVnsDeviceParams) SetHostname(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["hostname"] = v
	return
}

func (p *AddBigSwitchVnsDeviceParams) SetPhysicalnetworkid(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["physicalnetworkid"] = v
	return
}

// You should always use this function to get a new AddBigSwitchVnsDeviceParams instance,
// as then you are sure you have configured all required params
func (s *BigSwitchVNSService) NewAddBigSwitchVnsDeviceParams(hostname string, physicalnetworkid string) *AddBigSwitchVnsDeviceParams {
	p := &AddBigSwitchVnsDeviceParams{}
	p.p = make(map[string]interface{})
	p.p["hostname"] = hostname
	p.p["physicalnetworkid"] = physicalnetworkid
	return p
}

// Adds a BigSwitch VNS device
func (s *BigSwitchVNSService) AddBigSwitchVnsDevice(p *AddBigSwitchVnsDeviceParams) (*AddBigSwitchVnsDeviceResponse, error) {
	resp, err := s.cs.newRequest("addBigSwitchVnsDevice", p.toURLValues())
	if err != nil {
		return nil, err
	}

	var r AddBigSwitchVnsDeviceResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}

	// If we have a async client, we need to wait for the async result
	if s.cs.async {
		b, err := s.cs.GetAsyncJobResult(r.JobID, s.cs.timeout)
		if err != nil {
			if err == AsyncTimeoutErr {
				return &r, err
			}
			return nil, err
		}

		b, err = getRawValue(b)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

type AddBigSwitchVnsDeviceResponse struct {
	JobID               string `json:"jobid,omitempty"`
	Bigswitchdevicename string `json:"bigswitchdevicename,omitempty"`
	Hostname            string `json:"hostname,omitempty"`
	Physicalnetworkid   string `json:"physicalnetworkid,omitempty"`
	Provider            string `json:"provider,omitempty"`
	Vnsdeviceid         string `json:"vnsdeviceid,omitempty"`
}

type DeleteBigSwitchVnsDeviceParams struct {
	p map[string]interface{}
}

func (p *DeleteBigSwitchVnsDeviceParams) toURLValues() url.Values {
	u := url.Values{}
	if p.p == nil {
		return u
	}
	if v, found := p.p["vnsdeviceid"]; found {
		u.Set("vnsdeviceid", v.(string))
	}
	return u
}

func (p *DeleteBigSwitchVnsDeviceParams) SetVnsdeviceid(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["vnsdeviceid"] = v
	return
}

// You should always use this function to get a new DeleteBigSwitchVnsDeviceParams instance,
// as then you are sure you have configured all required params
func (s *BigSwitchVNSService) NewDeleteBigSwitchVnsDeviceParams(vnsdeviceid string) *DeleteBigSwitchVnsDeviceParams {
	p := &DeleteBigSwitchVnsDeviceParams{}
	p.p = make(map[string]interface{})
	p.p["vnsdeviceid"] = vnsdeviceid
	return p
}

//  delete a bigswitch vns device
func (s *BigSwitchVNSService) DeleteBigSwitchVnsDevice(p *DeleteBigSwitchVnsDeviceParams) (*DeleteBigSwitchVnsDeviceResponse, error) {
	resp, err := s.cs.newRequest("deleteBigSwitchVnsDevice", p.toURLValues())
	if err != nil {
		return nil, err
	}

	var r DeleteBigSwitchVnsDeviceResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}

	// If we have a async client, we need to wait for the async result
	if s.cs.async {
		b, err := s.cs.GetAsyncJobResult(r.JobID, s.cs.timeout)
		if err != nil {
			if err == AsyncTimeoutErr {
				return &r, err
			}
			return nil, err
		}

		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

type DeleteBigSwitchVnsDeviceResponse struct {
	JobID       string `json:"jobid,omitempty"`
	Displaytext string `json:"displaytext,omitempty"`
	Success     bool   `json:"success,omitempty"`
}

type ListBigSwitchVnsDevicesParams struct {
	p map[string]interface{}
}

func (p *ListBigSwitchVnsDevicesParams) toURLValues() url.Values {
	u := url.Values{}
	if p.p == nil {
		return u
	}
	if v, found := p.p["keyword"]; found {
		u.Set("keyword", v.(string))
	}
	if v, found := p.p["page"]; found {
		vv := strconv.Itoa(v.(int))
		u.Set("page", vv)
	}
	if v, found := p.p["pagesize"]; found {
		vv := strconv.Itoa(v.(int))
		u.Set("pagesize", vv)
	}
	if v, found := p.p["physicalnetworkid"]; found {
		u.Set("physicalnetworkid", v.(string))
	}
	if v, found := p.p["vnsdeviceid"]; found {
		u.Set("vnsdeviceid", v.(string))
	}
	return u
}

func (p *ListBigSwitchVnsDevicesParams) SetKeyword(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["keyword"] = v
	return
}

func (p *ListBigSwitchVnsDevicesParams) SetPage(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["page"] = v
	return
}

func (p *ListBigSwitchVnsDevicesParams) SetPagesize(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["pagesize"] = v
	return
}

func (p *ListBigSwitchVnsDevicesParams) SetPhysicalnetworkid(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["physicalnetworkid"] = v
	return
}

func (p *ListBigSwitchVnsDevicesParams) SetVnsdeviceid(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["vnsdeviceid"] = v
	return
}

// You should always use this function to get a new ListBigSwitchVnsDevicesParams instance,
// as then you are sure you have configured all required params
func (s *BigSwitchVNSService) NewListBigSwitchVnsDevicesParams() *ListBigSwitchVnsDevicesParams {
	p := &ListBigSwitchVnsDevicesParams{}
	p.p = make(map[string]interface{})
	return p
}

// Lists BigSwitch Vns devices
func (s *BigSwitchVNSService) ListBigSwitchVnsDevices(p *ListBigSwitchVnsDevicesParams) (*ListBigSwitchVnsDevicesResponse, error) {
	resp, err := s.cs.newRequest("listBigSwitchVnsDevices", p.toURLValues())
	if err != nil {
		return nil, err
	}

	var r ListBigSwitchVnsDevicesResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

type ListBigSwitchVnsDevicesResponse struct {
	Count               int                   `json:"count"`
	BigSwitchVnsDevices []*BigSwitchVnsDevice `json:"bigswitchvnsdevice"`
}

type BigSwitchVnsDevice struct {
	Bigswitchdevicename string `json:"bigswitchdevicename,omitempty"`
	Hostname            string `json:"hostname,omitempty"`
	Physicalnetworkid   string `json:"physicalnetworkid,omitempty"`
	Provider            string `json:"provider,omitempty"`
	Vnsdeviceid         string `json:"vnsdeviceid,omitempty"`
}

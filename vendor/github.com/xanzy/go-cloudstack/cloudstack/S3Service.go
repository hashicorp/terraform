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
	"fmt"
	"net/url"
	"strconv"
)

type AddS3Params struct {
	p map[string]interface{}
}

func (p *AddS3Params) toURLValues() url.Values {
	u := url.Values{}
	if p.p == nil {
		return u
	}
	if v, found := p.p["accesskey"]; found {
		u.Set("accesskey", v.(string))
	}
	if v, found := p.p["bucket"]; found {
		u.Set("bucket", v.(string))
	}
	if v, found := p.p["connectiontimeout"]; found {
		vv := strconv.Itoa(v.(int))
		u.Set("connectiontimeout", vv)
	}
	if v, found := p.p["endpoint"]; found {
		u.Set("endpoint", v.(string))
	}
	if v, found := p.p["maxerrorretry"]; found {
		vv := strconv.Itoa(v.(int))
		u.Set("maxerrorretry", vv)
	}
	if v, found := p.p["secretkey"]; found {
		u.Set("secretkey", v.(string))
	}
	if v, found := p.p["sockettimeout"]; found {
		vv := strconv.Itoa(v.(int))
		u.Set("sockettimeout", vv)
	}
	if v, found := p.p["usehttps"]; found {
		vv := strconv.FormatBool(v.(bool))
		u.Set("usehttps", vv)
	}
	return u
}

func (p *AddS3Params) SetAccesskey(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["accesskey"] = v
	return
}

func (p *AddS3Params) SetBucket(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["bucket"] = v
	return
}

func (p *AddS3Params) SetConnectiontimeout(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["connectiontimeout"] = v
	return
}

func (p *AddS3Params) SetEndpoint(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["endpoint"] = v
	return
}

func (p *AddS3Params) SetMaxerrorretry(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["maxerrorretry"] = v
	return
}

func (p *AddS3Params) SetSecretkey(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["secretkey"] = v
	return
}

func (p *AddS3Params) SetSockettimeout(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["sockettimeout"] = v
	return
}

func (p *AddS3Params) SetUsehttps(v bool) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["usehttps"] = v
	return
}

// You should always use this function to get a new AddS3Params instance,
// as then you are sure you have configured all required params
func (s *S3Service) NewAddS3Params(accesskey string, bucket string, secretkey string) *AddS3Params {
	p := &AddS3Params{}
	p.p = make(map[string]interface{})
	p.p["accesskey"] = accesskey
	p.p["bucket"] = bucket
	p.p["secretkey"] = secretkey
	return p
}

// Adds S3
func (s *S3Service) AddS3(p *AddS3Params) (*AddS3Response, error) {
	resp, err := s.cs.newRequest("addS3", p.toURLValues())
	if err != nil {
		return nil, err
	}

	var r AddS3Response
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

type AddS3Response struct {
	Details      []string `json:"details,omitempty"`
	Id           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Protocol     string   `json:"protocol,omitempty"`
	Providername string   `json:"providername,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Url          string   `json:"url,omitempty"`
	Zoneid       string   `json:"zoneid,omitempty"`
	Zonename     string   `json:"zonename,omitempty"`
}

type ListS3sParams struct {
	p map[string]interface{}
}

func (p *ListS3sParams) toURLValues() url.Values {
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
	return u
}

func (p *ListS3sParams) SetKeyword(v string) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["keyword"] = v
	return
}

func (p *ListS3sParams) SetPage(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["page"] = v
	return
}

func (p *ListS3sParams) SetPagesize(v int) {
	if p.p == nil {
		p.p = make(map[string]interface{})
	}
	p.p["pagesize"] = v
	return
}

// You should always use this function to get a new ListS3sParams instance,
// as then you are sure you have configured all required params
func (s *S3Service) NewListS3sParams() *ListS3sParams {
	p := &ListS3sParams{}
	p.p = make(map[string]interface{})
	return p
}

// This is a courtesy helper function, which in some cases may not work as expected!
func (s *S3Service) GetS3ID(keyword string) (string, error) {
	p := &ListS3sParams{}
	p.p = make(map[string]interface{})

	p.p["keyword"] = keyword

	l, err := s.ListS3s(p)
	if err != nil {
		return "", err
	}

	if l.Count == 0 {
		return "", fmt.Errorf("No match found for %s: %+v", keyword, l)
	}

	if l.Count == 1 {
		return l.S3s[0].Id, nil
	}

	if l.Count > 1 {
		for _, v := range l.S3s {
			if v.Name == keyword {
				return v.Id, nil
			}
		}
	}
	return "", fmt.Errorf("Could not find an exact match for %s: %+v", keyword, l)
}

// Lists S3s
func (s *S3Service) ListS3s(p *ListS3sParams) (*ListS3sResponse, error) {
	resp, err := s.cs.newRequest("listS3s", p.toURLValues())
	if err != nil {
		return nil, err
	}

	var r ListS3sResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

type ListS3sResponse struct {
	Count int   `json:"count"`
	S3s   []*S3 `json:"s3"`
}

type S3 struct {
	Details      []string `json:"details,omitempty"`
	Id           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Protocol     string   `json:"protocol,omitempty"`
	Providername string   `json:"providername,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Url          string   `json:"url,omitempty"`
	Zoneid       string   `json:"zoneid,omitempty"`
	Zonename     string   `json:"zonename,omitempty"`
}

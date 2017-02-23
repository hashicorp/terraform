/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package services

import (
	"fmt"
	"strings"

	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

// no documentation yet
type Utility_Network struct {
	Session *session.Session
	Options sl.Options
}

// GetUtilityNetworkService returns an instance of the Utility_Network SoftLayer service
func GetUtilityNetworkService(sess *session.Session) Utility_Network {
	return Utility_Network{Session: sess}
}

func (r Utility_Network) Id(id int) Utility_Network {
	r.Options.Id = &id
	return r
}

func (r Utility_Network) Mask(mask string) Utility_Network {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Utility_Network) Filter(filter string) Utility_Network {
	r.Options.Filter = filter
	return r
}

func (r Utility_Network) Limit(limit int) Utility_Network {
	r.Options.Limit = &limit
	return r
}

func (r Utility_Network) Offset(offset int) Utility_Network {
	r.Options.Offset = &offset
	return r
}

// A method used to return the nameserver information for a given address
func (r Utility_Network) NsLookup(address *string, typ *string) (resp string, err error) {
	params := []interface{}{
		address,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Utility_Network", "nsLookup", params, &r.Options, &resp)
	return
}

// Perform a WHOIS lookup from SoftLayer's application servers on the given IP address or hostname and return the raw results of that command. The returned result is similar to the result received from running the command `whois` from a UNIX command shell. A WHOIS lookup queries a host's registrar to retrieve domain registrant information including registration date, expiry date, and the administrative, technical, billing, and abuse contacts responsible for a domain. WHOIS lookups are useful for determining a physical contact responsible for a particular domain. WHOIS lookups are also useful for determining domain availability. Running a WHOIS lookup on an IP address queries ARIN for that IP block's ownership, and is helpful for determining a physical entity responsible for a certain IP address.
func (r Utility_Network) Whois(address *string) (resp string, err error) {
	params := []interface{}{
		address,
	}
	err = r.Session.DoRequest("SoftLayer_Utility_Network", "whois", params, &r.Options, &resp)
	return
}

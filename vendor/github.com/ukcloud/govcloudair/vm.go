/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	// "bytes"
	// "encoding/xml"
	"fmt"
	// "log"
	"net/url"
	// "os"
	// "strconv"

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

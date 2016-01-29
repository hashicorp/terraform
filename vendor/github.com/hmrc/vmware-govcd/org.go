/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcd

import (
	"fmt"
	"net/url"

	types "github.com/hmrc/vmware-govcd/types/v56"
)

type Org struct {
	Org *types.Org
	c   *Client
}

func NewOrg(c *Client) *Org {
	return &Org{
		Org: new(types.Org),
		c:   c,
	}
}

func (o *Org) FindCatalog(catalog string) (Catalog, error) {

	for _, av := range o.Org.Link {
		if av.Rel == "down" && av.Type == "application/vnd.vmware.vcloud.catalog+xml" && av.Name == catalog {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return Catalog{}, fmt.Errorf("error decoding org response: %s", err)
			}

			req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(o.c.Http.Do(req))
			if err != nil {
				return Catalog{}, fmt.Errorf("error retreiving catalog: %s", err)
			}

			cat := NewCatalog(o.c)

			if err = decodeBody(resp, cat.Catalog); err != nil {
				return Catalog{}, fmt.Errorf("error decoding catalog response: %s", err)
			}

			// The request was successful
			return *cat, nil

		}
	}

	return Catalog{}, fmt.Errorf("can't find catalog: %s", catalog)
}

/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/url"

	types "github.com/ukcloud/govcloudair/types/v56"
)

type Catalog struct {
	Catalog *types.Catalog
	c       *Client
}

func NewCatalog(c *Client) *Catalog {
	return &Catalog{
		Catalog: new(types.Catalog),
		c:       c,
	}
}

func (c *Catalog) FindCatalogItem(catalogitem string) (CatalogItem, error) {

	for _, cis := range c.Catalog.CatalogItems {
		for _, ci := range cis.CatalogItem {
			if ci.Name == catalogitem && ci.Type == "application/vnd.vmware.vcloud.catalogItem+xml" {
				u, err := url.ParseRequestURI(ci.HREF)

				if err != nil {
					return CatalogItem{}, fmt.Errorf("error decoding catalog response: %s", err)
				}

				req := c.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(c.c.Http.Do(req))
				if err != nil {
					return CatalogItem{}, fmt.Errorf("error retreiving catalog: %s", err)
				}

				cat := NewCatalogItem(c.c)

				if err = decodeBody(resp, cat.CatalogItem); err != nil {
					return CatalogItem{}, fmt.Errorf("error decoding catalog response: %s", err)
				}

				// The request was successful
				return *cat, nil
			}
		}
	}

	return CatalogItem{}, fmt.Errorf("can't find catalog item: %s", catalogitem)
}

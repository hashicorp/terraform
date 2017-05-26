/*
 * Copyright 2016 Skyscape Cloud Services.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"

	types "github.com/ukcloud/govcloudair/types/v56"
)

type Results struct {
	Results *types.QueryResultRecordsType
	c       *Client
}

func NewResults(c *Client) *Results {
	return &Results{
		Results: new(types.QueryResultRecordsType),
		c:       c,
	}
}

func (c *VCDClient) Query(params map[string]string) (Results, error) {

	req := c.Client.NewRequest(params, "GET", c.QueryHREF, nil)
	req.Header.Add("Accept", "vnd.vmware.vcloud.org+xml;version=5.5")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Results{}, fmt.Errorf("error retreiving query: %s", err)
	}

	results := NewResults(&c.Client)

	if err = decodeBody(resp, results.Results); err != nil {
		return Results{}, fmt.Errorf("error decoding query results: %s", err)
	}

	return *results, nil
}

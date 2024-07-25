// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/zclconf/go-cty/cty"
)

var _ auth.CredentialsSource = &credentialsSource{}

type credentialsSource struct {
	configured map[svchost.Hostname]cty.Value
}

func newCredentialsSource() *credentialsSource {
	return &credentialsSource{
		configured: map[svchost.Hostname]cty.Value{},
	}
}

func (c *credentialsSource) ForHost(host svchost.Hostname) (auth.HostCredentials, error) {
	v, ok := c.configured[host]
	if ok {
		return auth.HostCredentialsFromObject(v), nil
	}
	return nil, nil
}

func (c *credentialsSource) StoreForHost(host svchost.Hostname, credentials auth.HostCredentialsWritable) error {
	c.configured[host] = credentials.ToStore()
	return nil
}

func (c *credentialsSource) ForgetForHost(host svchost.Hostname) error {
	delete(c.configured, host)
	return nil
}

/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"github.com/vmware/govcloudair/testutil"

	. "gopkg.in/check.v1"
)

func (s *S) Test_NetRefresh(c *C) {

	// Get the Org populated
	testServer.ResponseMap(1, testutil.ResponseMap{
		"/api/network/44444444-4444-4444-4444-4444444444444": testutil.Response{200, nil, orgvdcnetExample},
	})

	network, err := s.vdc.FindVDCNetwork("networkName")
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(network.OrgVDCNetwork.Name, Equals, "networkName")

	testServer.Response(200, nil, orgvdcnetExample)
	err = network.Refresh()
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(network.OrgVDCNetwork.Name, Equals, "networkName")

}

var orgvdcnetExample = `
<?xml version="1.0" encoding="UTF-8"?>
<OrgVdcNetwork xmlns="http://www.vmware.com/vcloud/v1.5" status="1" name="networkName" id="urn:vcloud:network:cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1" href="http://localhost:4444/api/network/cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1" type="application/vnd.vmware.vcloud.orgVdcNetwork+xml" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
    <Link rel="up" href="http://localhost:4444/api/vdc/214cd6b2-3f7a-4ee5-9b0a-52b4001a4a84" type="application/vnd.vmware.vcloud.vdc+xml"/>
    <Link rel="down" href="http://localhost:4444/api/network/cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1/metadata" type="application/vnd.vmware.vcloud.metadata+xml"/>
    <Link rel="down" href="http://localhost:4444/api/network/cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1/allocatedAddresses/" type="application/vnd.vmware.vcloud.allocatedNetworkAddress+xml"/>
    <Description>This routed network was created with Create VDC.</Description>
    <Configuration>
        <IpScopes>
            <IpScope>
                <IsInherited>false</IsInherited>
                <Gateway>192.168.109.1</Gateway>
                <Netmask>255.255.255.0</Netmask>
                <IsEnabled>true</IsEnabled>
                <IpRanges>
                    <IpRange>
                        <StartAddress>192.168.109.2</StartAddress>
                        <EndAddress>192.168.109.100</EndAddress>
                    </IpRange>
                </IpRanges>
            </IpScope>
        </IpScopes>
        <FenceMode>natRouted</FenceMode>
        <RetainNetInfoAcrossDeployments>false</RetainNetInfoAcrossDeployments>
    </Configuration>
    <IsShared>false</IsShared>
</OrgVdcNetwork>
	`

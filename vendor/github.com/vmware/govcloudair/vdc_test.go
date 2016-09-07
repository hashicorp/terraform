/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"github.com/vmware/govcloudair/testutil"

	. "gopkg.in/check.v1"
)

func (s *S) Test_FindVDCNetwork(c *C) {

	testServer.Response(200, nil, orgvdcnetExample)

	net, err := s.vdc.FindVDCNetwork("networkName")

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(net, NotNil)
	c.Assert(net.OrgVDCNetwork.HREF, Equals, "http://localhost:4444/api/network/cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1")

	// find Invalid Network
	net, err = s.vdc.FindVDCNetwork("INVALID")
	c.Assert(err, NotNil)
}

func (s *S) Test_GetVDCOrg(c *C) {

	testServer.Response(200, nil, orgExample)

	org, err := s.vdc.GetVDCOrg()

	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(org, NotNil)
	c.Assert(org.Org.HREF, Equals, "http://localhost:4444/api/org/23bd2339-c55f-403c-baf3-13109e8c8d57")
}

func (s *S) Test_NewVdc(c *C) {

	testServer.Response(200, nil, vdcExample)
	err := s.vdc.Refresh()
	_ = testServer.WaitRequest()
	c.Assert(err, IsNil)

	c.Assert(s.vdc.Vdc.Link[0].Rel, Equals, "up")
	c.Assert(s.vdc.Vdc.Link[0].Type, Equals, "application/vnd.vmware.vcloud.org+xml")
	c.Assert(s.vdc.Vdc.Link[0].HREF, Equals, "http://localhost:4444/api/org/11111111-1111-1111-1111-111111111111")

	c.Assert(s.vdc.Vdc.AllocationModel, Equals, "AllocationPool")

	for _, v := range s.vdc.Vdc.ComputeCapacity {
		c.Assert(v.CPU.Units, Equals, "MHz")
		c.Assert(v.CPU.Allocated, Equals, int64(30000))
		c.Assert(v.CPU.Limit, Equals, int64(30000))
		c.Assert(v.CPU.Reserved, Equals, int64(15000))
		c.Assert(v.CPU.Used, Equals, int64(0))
		c.Assert(v.CPU.Overhead, Equals, int64(0))
		c.Assert(v.Memory.Units, Equals, "MB")
		c.Assert(v.Memory.Allocated, Equals, int64(61440))
		c.Assert(v.Memory.Limit, Equals, int64(61440))
		c.Assert(v.Memory.Reserved, Equals, int64(61440))
		c.Assert(v.Memory.Used, Equals, int64(6144))
		c.Assert(v.Memory.Overhead, Equals, int64(95))
	}

	c.Assert(s.vdc.Vdc.ResourceEntities[0].ResourceEntity[0].Name, Equals, "vAppTemplate")
	c.Assert(s.vdc.Vdc.ResourceEntities[0].ResourceEntity[0].Type, Equals, "application/vnd.vmware.vcloud.vAppTemplate+xml")
	c.Assert(s.vdc.Vdc.ResourceEntities[0].ResourceEntity[0].HREF, Equals, "http://localhost:4444/api/vAppTemplate/vappTemplate-22222222-2222-2222-2222-222222222222")

	for _, v := range s.vdc.Vdc.AvailableNetworks {
		for _, v2 := range v.Network {
			c.Assert(v2.Name, Equals, "networkName")
			c.Assert(v2.Type, Equals, "application/vnd.vmware.vcloud.network+xml")
			c.Assert(v2.HREF, Equals, "http://localhost:4444/api/network/44444444-4444-4444-4444-4444444444444")
		}
	}

	c.Assert(s.vdc.Vdc.NicQuota, Equals, 0)
	c.Assert(s.vdc.Vdc.NetworkQuota, Equals, 20)
	c.Assert(s.vdc.Vdc.UsedNetworkCount, Equals, 0)
	c.Assert(s.vdc.Vdc.VMQuota, Equals, 0)
	c.Assert(s.vdc.Vdc.IsEnabled, Equals, true)

	for _, v := range s.vdc.Vdc.VdcStorageProfiles {
		for _, v2 := range v.VdcStorageProfile {
			c.Assert(v2.Name, Equals, "storageProfile")
			c.Assert(v2.Type, Equals, "application/vnd.vmware.vcloud.vdcStorageProfile+xml")
			c.Assert(v2.HREF, Equals, "http://localhost:4444/api/vdcStorageProfile/88888888-8888-8888-8888-888888888888")
		}
	}

}

func (s *S) Test_FindVApp(c *C) {

	// testServer.Response(200, nil, vappExample)

	// vapp, err := s.vdc.FindVAppByID("")

	// _ = testServer.WaitRequest()
	// testServer.Flush()
	// c.Assert(err, IsNil)

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/vdc/00000000-0000-0000-0000-000000000000":       testutil.Response{200, nil, vdcExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000": testutil.Response{200, nil, vappExample},
	})

	_, err := s.vdc.FindVAppByName("myVApp")

	_ = testServer.WaitRequests(2)

	c.Assert(err, IsNil)

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/vdc/00000000-0000-0000-0000-000000000000":       testutil.Response{200, nil, vdcExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000": testutil.Response{200, nil, vappExample},
	})

	_, err = s.vdc.FindVAppByID("urn:vcloud:vapp:00000000-0000-0000-0000-000000000000")

	_ = testServer.WaitRequests(2)

	c.Assert(err, IsNil)

}

var vdcExample = `
	<?xml version="1.0" ?>
	<Vdc href="http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" id="urn:vcloud:vdc:00000000-0000-0000-0000-000000000000" name="M916272752-5793" status="1" type="application/vnd.vmware.vcloud.vdc+xml" xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:xsi="http://www.w3.org/2001/XMLSchema-in stance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
	  <Link href="http://localhost:4444/api/org/11111111-1111-1111-1111-111111111111" rel="up" type="application/vnd.vmware.vcloud.org+xml"/>
	  <Link href="http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000/edgeGateways" rel="edgeGateways" type="application/vnd.vmware.vcloud.query.records+xml"/>
	  <AllocationModel>AllocationPool</AllocationModel>
	  <ComputeCapacity>
	    <Cpu>
	      <Units>MHz</Units>
	      <Allocated>30000</Allocated>
	      <Limit>30000</Limit>
	      <Reserved>15000</Reserved>
	      <Used>0</Used>
	      <Overhead>0</Overhead>
	    </Cpu>
	    <Memory>
	      <Units>MB</Units>
	      <Allocated>61440</Allocated>
	      <Limit>61440</Limit>
	      <Reserved>61440</Reserved>
	      <Used>6144</Used>
	      <Overhead>95</Overhead>
	    </Memory>
	  </ComputeCapacity>
	  <ResourceEntities>
	    <ResourceEntity href="http://localhost:4444/api/vAppTemplate/vappTemplate-22222222-2222-2222-2222-222222222222" name="vAppTemplate" type="application/vnd.vmware.vcloud.vAppTemplate+xml"/>
      <ResourceEntity href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" name="myVApp" type="application/vnd.vmware.vcloud.vApp+xml"/>
	  </ResourceEntities>
	  <AvailableNetworks>
	    <Network href="http://localhost:4444/api/network/44444444-4444-4444-4444-4444444444444" name="networkName" type="application/vnd.vmware.vcloud.network+xml"/>
	  </AvailableNetworks>
	  <Capabilities>
	    <SupportedHardwareVersions>
	      <SupportedHardwareVersion>vmx-10</SupportedHardwareVersion>
	    </SupportedHardwareVersions>
	  </Capabilities>
	  <NicQuota>0</NicQuota>
	  <NetworkQuota>20</NetworkQuota>
	  <UsedNetworkCount>0</UsedNetworkCount>
	  <VmQuota>0</VmQuota>
	  <IsEnabled>true</IsEnabled>
	  <VdcStorageProfiles>
	    <VdcStorageProfile href="http://localhost:4444/api/vdcStorageProfile/88888888-8888-8888-8888-888888888888" name="storageProfile" type="application/vnd.vmware.vcloud.vdcStorageProfile+xml"/>
	  </VdcStorageProfiles>
	</Vdc>
	`

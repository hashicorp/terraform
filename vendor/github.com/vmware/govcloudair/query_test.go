/*
 * Copyright 2016 Skyscape Cloud Services.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	// "fmt"
	"github.com/vmware/govcloudair/testutil"
	. "gopkg.in/check.v1"
)

func (s *K) Test_Query(c *C) {

	// Get the Org populated
	testServer.ResponseMap(1, testutil.ResponseMap{
		"/api/query?type=vm": testutil.Response{200, nil, queryVmExample},
	})

	results, err := s.client.Query(map[string]string{"type": "vm"})
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(results.Results.Total, Equals, 4)
	c.Assert(len(results.Results.VMRecord), Equals, 4)
}

var queryVmExample = `<?xml version="1.0" encoding="UTF-8"?>
<QueryResultRecords xmlns="http://www.vmware.com/vcloud/v1.5" name="vm" page="1" pageSize="25" total="4" href="http://localhost:4444/api/query?type=vm&amp;page=1&amp;pageSize=25&amp;format=records" type="application/vnd.vmware.vcloud.query.records+xml" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.10.6.15/api/v1.5/schema/master.xsd">
    <Link rel="alternate" href="http://localhost:4444/api/query?type=vm&amp;page=1&amp;pageSize=25&amp;format=references" type="application/vnd.vmware.vcloud.query.references+xml"/>
    <Link rel="alternate" href="http://localhost:4444/api/query?type=vm&amp;page=1&amp;pageSize=25&amp;format=idrecords" type="application/vnd.vmware.vcloud.query.idrecords+xml"/>
    <VMRecord container="http://localhost:4444/api/vApp/vapp-11111111-1111-1111-1111-111111111111" containerName="jenkins01" guestOs="CentOS 4/5/6/7 (64-bit)" hardwareVersion="9" isBusy="false" isDeleted="false" isDeployed="true" isInMaintenanceMode="false" isPublished="false" isVAppTemplate="false" memoryMB="2048" name="jenkins01" numberOfCpus="2" status="POWERED_ON" storageProfileName="BASIC-Any" vdc="http://localhost:4444/api/vdc/55555555-5555-5555-5555-555555555555" href="http://localhost:4444/api/vApp/vm-66666666-6666-6666-6666-666666666666" isVdcEnabled="true" pvdcHighestSupportedHardwareVersion="9" taskStatus="success" task="http://localhost:4444/api/task/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" taskDetails=" " vmToolsVersion="2147483647" networkName="Development Network" taskStatusName="vappDeploy"/>
    <VMRecord container="http://localhost:4444/api/vApp/vapp-22222222-2222-2222-2222-222222222222" containerName="Hipchat" guestOs="Ubuntu Linux (64-bit)" hardwareVersion="9" isBusy="false" isDeleted="false" isDeployed="true" isInMaintenanceMode="false" isPublished="false" isVAppTemplate="false" memoryMB="4096" name="hipchat01" numberOfCpus="4" status="POWERED_ON" storageProfileName="BASIC-Any" vdc="http://localhost:4444/api/vdc/55555555-5555-5555-5555-555555555555" href="http://localhost:4444/api/vApp/vm-77777777-7777-7777-7777-777777777777" isVdcEnabled="true" pvdcHighestSupportedHardwareVersion="9" taskStatus="success" task="http://localhost:4444/api/task/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" taskDetails=" " vmToolsVersion="2147483647" networkName="Development Network" taskStatusName="metadataUpdate"/>
    <VMRecord container="http://localhost:4444/api/vApp/vapp-33333333-3333-3333-3333-333333333333" containerName="TestVMXNET3" guestOs="CentOS 4/5/6/7 (64-bit)" hardwareVersion="9" isBusy="false" isDeleted="false" isDeployed="false" isInMaintenanceMode="false" isPublished="false" isVAppTemplate="false" memoryMB="512" name="testvmxnet" numberOfCpus="1" status="POWERED_OFF" storageProfileName="BASIC-Any" vdc="http://localhost:4444/api/vdc/55555555-5555-5555-5555-555555555555" href="http://localhost:4444/api/vApp/vm-88888888-8888-8888-8888-888888888888" isVdcEnabled="true" pvdcHighestSupportedHardwareVersion="9" taskStatus="success" task="http://localhost:4444/api/task/cccccccc-cccc-cccc-cccc-cccccccccccc" taskDetails=" " vmToolsVersion="2147483647" networkName="Development Network" taskStatusName="metadataUpdate"/>
    <VMRecord catalogName="DevOps" container="http://localhost:4444/api/vAppTemplate/vappTemplate-44444444-4444-4444-4444-444444444444" containerName="centos71" guestOs="CentOS 4/5/6/7 (64-bit)" hardwareVersion="9" isBusy="false" isDeleted="false" isDeployed="false" isInMaintenanceMode="false" isPublished="false" isVAppTemplate="true" memoryMB="512" name="centos71" numberOfCpus="1" status="POWERED_OFF" storageProfileName="BASIC-Any" vdc="http://localhost:4444/api/vdc/55555555-5555-5555-5555-555555555555" href="http://localhost:4444/api/vAppTemplate/vm-99999999-9999-9999-9999-999999999999" isVdcEnabled="true" pvdcHighestSupportedHardwareVersion="9" vmToolsVersion="2147483647"/>
</QueryResultRecords>
	`

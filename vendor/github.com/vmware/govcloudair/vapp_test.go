/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"github.com/vmware/govcloudair/testutil"

	. "gopkg.in/check.v1"
)

func (s *S) Test_ComposeVApp(c *C) {

	testServer.ResponseMap(7, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                       testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                  testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                   testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":               testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5": testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":    testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                 testutil.Response{200, nil, vappExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	status, err := s.vapp.GetStatus()

	c.Assert(err, IsNil)
	c.Assert(status, Equals, "POWERED_OFF")

	_ = testServer.WaitRequests(7)

}

func (s *S) Test_PowerOn(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.PowerOn()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_PowerOff(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.PowerOff()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Reboot(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Reboot()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_SetOvf(c *C) {
	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                       testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                  testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                   testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":               testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5": testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":    testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                 testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000/productSections":   testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	var test map[string]string
	test = make(map[string]string)
	test["guestinfo.hostname"] = "testhostname"
	task, err = s.vapp.SetOvf(test)

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

func (s *S) Test_AddMetadata(c *C) {
	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                       testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                  testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                   testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":               testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5": testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":    testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                 testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000/metadata/key":      testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	task, err = s.vapp.AddMetadata("key", "value")

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

func (s *S) Test_ChangeVMName(c *C) {

	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                       testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                  testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                   testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":               testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5": testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":    testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                 testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000":                   testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	task, err = s.vapp.ChangeVMName("My-vm")

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

func (s *S) Test_Reset(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Reset()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Suspend(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Suspend()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Shutdown(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Shutdown()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Undeploy(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Undeploy()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Deploy(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Deploy()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_Delete(c *C) {

	testServer.Response(200, nil, taskExample)
	task, err := s.vapp.Delete()
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

}

func (s *S) Test_RunCustomizationScript(c *C) {

	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                                testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                           testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                            testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":                        testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5":          testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":             testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000/guestCustomizationSection/": testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	task, err = s.vapp.RunCustomizationScript("computername", "this is my script")

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

func (s *S) Test_ChangeCPUcount(c *C) {

	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                                testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                           testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                            testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":                        testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5":          testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":             testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/cpu": testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	task, err = s.vapp.ChangeCPUcount(2)

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

func (s *S) Test_ChangeMemorySize(c *C) {

	testServer.ResponseMap(8, testutil.ResponseMap{
		"/api/org/11111111-1111-1111-1111-111111111111":                                   testutil.Response{200, nil, orgExample},
		"/api/network/44444444-4444-4444-4444-4444444444444":                              testutil.Response{200, nil, orgvdcnetExample},
		"/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854":                               testutil.Response{200, nil, catalogExample},
		"/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae":                           testutil.Response{200, nil, catalogitemExample},
		"/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5":             testutil.Response{200, nil, vapptemplateExample},
		"/api/vdc/00000000-0000-0000-0000-000000000000/action/composeVApp":                testutil.Response{200, nil, instantiatedvappExample},
		"/api/vApp/vapp-00000000-0000-0000-0000-000000000000":                             testutil.Response{200, nil, vappExample},
		"/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/memory": testutil.Response{200, nil, taskExample},
	})

	// Get the Org populated
	org, err := s.vdc.GetVDCOrg()
	c.Assert(err, IsNil)

	// Populate OrgVDCNetwork
	net, err := s.vdc.FindVDCNetwork("networkName")
	c.Assert(err, IsNil)

	// Populate Catalog
	cat, err := org.FindCatalog("Public Catalog")
	c.Assert(err, IsNil)

	// Populate Catalog Item
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	c.Assert(err, IsNil)

	// Get VAppTemplate
	vapptemplate, err := catitem.GetVAppTemplate()
	c.Assert(err, IsNil)

	// Compose VApp
	task, err := s.vapp.ComposeVApp(net, vapptemplate, "name", "description")
	c.Assert(err, IsNil)
	c.Assert(task.Task.OperationName, Equals, "vdcInstantiateVapp")
	c.Assert(s.vapp.VApp.HREF, Equals, "http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000")

	task, err = s.vapp.ChangeMemorySize(4096)

	c.Assert(err, IsNil)
	c.Assert(task.Task.Status, Equals, "success")

	_ = testServer.WaitRequests(8)

}

var instantiatedvappExample = `
	<?xml version="1.0" ?>
	<VApp deployed="false" href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" id="urn:vcloud:vapp:00000000-0000-0000-0000-000000000000" name="myVApp" ovfDescriptorUploaded="true" status="0" type="application/vnd.vmware.vcloud.vApp+xml" xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
	  <Link href="http://localhost:4444/api/network/f869430c-7490-4d32-bf34-4208b6059c21" name="M916272752-5793-default-routed" rel="down" type="application/vnd.vmware.vcloud.vAppNetwork+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/controlAccess/" rel="down" type="application/vnd.vmware.vcloud.controlAccess+xml"/>
	  <Link href="http://localhost:4444/api/vdc/214cd6b2-3f7a-4ee5-9b0a-52b4001a4a84" rel="up" type="application/vnd.vmware.vcloud.vdc+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/owner" rel="down" type="application/vnd.vmware.vcloud.owner+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/metadata" rel="down" type="application/vnd.vmware.vcloud.metadata+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/ovf" rel="ovf" type="text/xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/productSections/" rel="down" type="application/vnd.vmware.vcloud.productSections+xml"/>
	  <Link href="http://localhost:4444/api/vApp/00000000-0000-0000-0000-000000000000/backups" rel="add" type="application/vnd.emc.vcp.adhocBackupParams+xml"/>
	  <Link href="http://localhost:4444/api/vApp/00000000-0000-0000-0000-000000000000/backups" rel="add" type="application/vnd.emc.vcp.adhocBackupParams+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/metrics/historic" rel="advancedmetrics" type="application/vnd.vmware.vcloud.metrics.historicUsageList+xml"/>
	  <Description>My vApp to be deployed</Description>
	  <Tasks>
	    <Task cancelRequested="false" expiryTime="2015-01-22T15:26:59.824Z" href="http://localhost:4444/api/task/b3ff4b8c-9292-41a5-8dc8-22aba49bb02d" id="urn:vcloud:task:b3ff4b8c-9292-41a5-8dc8-22aba49bb02d" name="task" operation="Creating Virtual Application My vApp 2(00000000-0000-0000-0000-000000000000)" operationName="vdcInstantiateVapp" serviceNamespace="com.vmware.vcloud" startTime="2014-10-24T15:26:59.824Z" status="running" type="application/vnd.vmware.vcloud.task+xml">
	      <Link href="http://localhost:4444/api/task/b3ff4b8c-9292-41a5-8dc8-22aba49bb02d/action/cancel" rel="task:cancel"/>
	      <Owner href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" name="My vApp 2" type="application/vnd.vmware.vcloud.vApp+xml"/>
	      <User href="http://localhost:4444/api/admin/user/d8ac278a-5b49-4c85-9a81-468838e89eb9" name="frapposelli1@gts-vchs.com" type="application/vnd.vmware.admin.user+xml"/>
	      <Organization href="http://localhost:4444/api/org/23bd2339-c55f-403c-baf3-13109e8c8d57" name="M916272752-5793" type="application/vnd.vmware.vcloud.org+xml"/>
	      <Progress>1</Progress>
	      <Details/>
	    </Task>
	  </Tasks>
	  <DateCreated>2014-10-24T15:26:59.067Z</DateCreated>
	  <Owner type="application/vnd.vmware.vcloud.owner+xml">
	    <User href="http://localhost:4444/api/admin/user/d8ac278a-5b49-4c85-9a81-468838e89eb9" name="frapposelli1@gts-vchs.com" type="application/vnd.vmware.admin.user+xml"/>
	  </Owner>
	  <InMaintenanceMode>false</InMaintenanceMode>
	</VApp>
	`

var vappExample = `
	<?xml version="1.0" ?>
	<VApp deployed="false" href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" id="urn:vcloud:vapp:00000000-0000-0000-0000-000000000000" name="Test API GO4" ovfDescriptorUploaded="true" status="8" type="application/vnd.vmware.vcloud.vApp+xml" xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:ovf="http://schemas.dmtf.org/ovf/envelope/1" xmlns:rasd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData" xmlns:vmw="http://www.vmware.com/schema/ovf" xmlns:vssd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_VirtualSystemSettingData.xsd http://www.vmware.com/schema/ovf http://www.vmware.com/schema/ovf http://schemas.dmtf.org/ovf/envelope/1http://schemas.dmtf.org/ovf/envelope/1/dsp8023_1.1.0.xsd http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_ResourceAllocationSettingData.xsd">
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/power/action/powerOn" rel="power:powerOn"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/deploy" rel="deploy" type="application/vnd.vmware.vcloud.deployVAppParams+xml"/>
	  <Link href="http://localhost:4444/api/network/e68434e9-a9ae-47d8-b809-743e70307085" name="M916272752-5793-default-isolated" rel="down" type="application/vnd.vmware.vcloud.vAppNetwork+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/controlAccess/" rel="down" type="application/vnd.vmware.vcloud.controlAccess+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/controlAccess" rel="controlAccess" type="application/vnd.vmware.vcloud.controlAccess+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/recomposeVApp" rel="recompose" type="application/vnd.vmware.vcloud.recomposeVAppParams+xml"/>
	  <Link href="http://localhost:4444/api/vdc/214cd6b2-3f7a-4ee5-9b0a-52b4001a4a84" rel="up" type="application/vnd.vmware.vcloud.vdc+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" rel="edit" type="application/vnd.vmware.vcloud.vApp+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" rel="remove"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/enableDownload" rel="enable"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/disableDownload" rel="disable"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/owner" rel="down" type="application/vnd.vmware.vcloud.owner+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/metadata" rel="down" type="application/vnd.vmware.vcloud.metadata+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/ovf" rel="ovf" type="text/xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/productSections/" rel="down" type="application/vnd.vmware.vcloud.productSections+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/action/createSnapshot" rel="snapshot:create" type="application/vnd.vmware.vcloud.createSnapshotParams+xml"/>
	  <Link href="http://localhost:4444/api/vApp/00000000-0000-0000-0000-000000000000/backups" rel="add" type="application/vnd.emc.vcp.adhocBackupParams+xml"/>
	  <Link href="http://localhost:4444/api/vApp/00000000-0000-0000-0000-000000000000/backups" rel="add" type="application/vnd.emc.vcp.adhocBackupParams+xml"/>
	  <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/metrics/historic" rel="advancedmetrics" type="application/vnd.vmware.vcloud.metrics.historicUsageList+xml"/>
	  <Description>Test API GO4444!</Description>
	  <LeaseSettingsSection href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/leaseSettingsSection/" ovf:required="false" type="application/vnd.vmware.vcloud.leaseSettingsSection+xml">
	    <ovf:Info>Lease settings section</ovf:Info>
	    <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/leaseSettingsSection/" rel="edit" type="application/vnd.vmware.vcloud.leaseSettingsSection+xml"/>
	    <DeploymentLeaseInSeconds>0</DeploymentLeaseInSeconds>
	    <StorageLeaseInSeconds>0</StorageLeaseInSeconds>
	  </LeaseSettingsSection>
	  <ovf:StartupSection vcloud:href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/startupSection/" vcloud:type="application/vnd.vmware.vcloud.startupSection+xml" xmlns:vcloud="http://www.vmware.com/vcloud/v1.5">
	    <ovf:Info>VApp startup section</ovf:Info>
	    <ovf:Item ovf:id="CentOS64-32bit" ovf:order="0" ovf:startAction="powerOn" ovf:startDelay="0" ovf:stopAction="powerOff" ovf:stopDelay="0"/>
	    <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/startupSection/" rel="edit" type="application/vnd.vmware.vcloud.startupSection+xml"/>
	  </ovf:StartupSection>
	  <ovf:NetworkSection vcloud:href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/networkSection/" vcloud:type="application/vnd.vmware.vcloud.networkSection+xml" xmlns:vcloud="http://www.vmware.com/vcloud/v1.5">
	    <ovf:Info>The list of logical networks</ovf:Info>
	    <ovf:Network ovf:name="M916272752-5793-default-isolated">
	      <ovf:Description>This isolated network was created with Create VDC.</ovf:Description>
	    </ovf:Network>
	    <ovf:Network ovf:name="none">
	      <ovf:Description>This is a special place-holder used for disconnected network interfaces.</ovf:Description>
	    </ovf:Network>
	  </ovf:NetworkSection>
	  <NetworkConfigSection href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/networkConfigSection/" ovf:required="false" type="application/vnd.vmware.vcloud.networkConfigSection+xml">
	    <ovf:Info>The configuration parameters for logical networks</ovf:Info>
	    <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/networkConfigSection/" rel="edit" type="application/vnd.vmware.vcloud.networkConfigSection+xml"/>
	    <NetworkConfig networkName="M916272752-5793-default-isolated">
	      <Link href="http://localhost:4444/api/admin/network/e68434e9-a9ae-47d8-b809-743e70307085/action/reset" rel="repair"/>
	      <Description>This isolated network was created with Create VDC.</Description>
	      <Configuration>
	        <IpScopes>
	          <IpScope>
	            <IsInherited>true</IsInherited>
	            <Gateway>192.168.99.1</Gateway>
	            <Netmask>255.255.255.0</Netmask>
	            <IsEnabled>true</IsEnabled>
	            <IpRanges>
	              <IpRange>
	                <StartAddress>192.168.99.2</StartAddress>
	                <EndAddress>192.168.99.100</EndAddress>
	              </IpRange>
	            </IpRanges>
	          </IpScope>
	        </IpScopes>
	        <ParentNetwork href="http://localhost:4444/api/admin/network/8d0cbfe2-25b3-4a1f-b608-5ffeabc7a53d" id="8d0cbfe2-25b3-4a1f-b608-5ffeabc7a53d" name="M916272752-5793-default-isolated"/>
	        <FenceMode>bridged</FenceMode>
	        <RetainNetInfoAcrossDeployments>false</RetainNetInfoAcrossDeployments>
	      </Configuration>
	      <IsDeployed>false</IsDeployed>
	    </NetworkConfig>
	    <NetworkConfig networkName="none">
	      <Description>This is a special place-holder used for disconnected network interfaces.</Description>
	      <Configuration>
	        <IpScopes>
	          <IpScope>
	            <IsInherited>false</IsInherited>
	            <Gateway>196.254.254.254</Gateway>
	            <Netmask>255.255.0.0</Netmask>
	            <Dns1>196.254.254.254</Dns1>
	          </IpScope>
	        </IpScopes>
	        <FenceMode>isolated</FenceMode>
	      </Configuration>
	      <IsDeployed>false</IsDeployed>
	    </NetworkConfig>
	  </NetworkConfigSection>
	  <SnapshotSection href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000/snapshotSection" ovf:required="false" type="application/vnd.vmware.vcloud.snapshotSection+xml">
	    <ovf:Info>Snapshot information section</ovf:Info>
	  </SnapshotSection>
	  <DateCreated>2014-11-06T22:24:43.913Z</DateCreated>
	  <Owner type="application/vnd.vmware.vcloud.owner+xml">
	    <User href="http://localhost:4444/api/admin/user/d8ac278a-5b49-4c85-9a81-468838e89eb9" name="frapposelli1@gts-vchs.com" type="application/vnd.vmware.admin.user+xml"/>
	  </Owner>
	  <InMaintenanceMode>false</InMaintenanceMode>
	  <Children>
	    <Vm deployed="false" href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000" id="urn:vcloud:vm:00000000-0000-0000-0000-000000000000" name="CentOS64-32bit" needsCustomization="true" nestedHypervisorEnabled="false" status="8" type="application/vnd.vmware.vcloud.vm+xml">
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/power/action/powerOn" rel="power:powerOn"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/deploy" rel="deploy" type="application/vnd.vmware.vcloud.deployVAppParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000" rel="edit" type="application/vnd.vmware.vcloud.vm+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000" rel="remove"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/metadata" rel="down" type="application/vnd.vmware.vcloud.metadata+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/productSections/" rel="down" type="application/vnd.vmware.vcloud.productSections+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/metrics/historic" rel="down" type="application/vnd.vmware.vcloud.metrics.historicUsageSpec+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/metrics/historic" rel="metrics" type="application/vnd.vmware.vcloud.metrics.historicUsageSpec+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/screen" rel="screen:thumbnail"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/media/action/insertMedia" rel="media:insertMedia" type="application/vnd.vmware.vcloud.mediaInsertOrEjectParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/media/action/ejectMedia" rel="media:ejectMedia" type="application/vnd.vmware.vcloud.mediaInsertOrEjectParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/disk/action/attach" rel="disk:attach" type="application/vnd.vmware.vcloud.diskAttachOrDetachParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/disk/action/detach" rel="disk:detach" type="application/vnd.vmware.vcloud.diskAttachOrDetachParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/upgradeHardwareVersion" rel="upgrade"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/enableNestedHypervisor" rel="enable"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/customizeAtNextPowerOn" rel="customizeAtNextPowerOn"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/createSnapshot" rel="snapshot:create" type="application/vnd.vmware.vcloud.createSnapshotParams+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/action/reconfigureVm" name="CentOS64-32bit" rel="reconfigureVm" type="application/vnd.vmware.vcloud.vm+xml"/>
	      <Link href="http://localhost:4444/api/vApp/vapp-00000000-0000-0000-0000-000000000000" rel="up" type="application/vnd.vmware.vcloud.vApp+xml"/>
	      <Description>id: cts-6.4-32bit</Description>
	      <ovf:VirtualHardwareSection ovf:transport="" vcloud:href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/" vcloud:type="application/vnd.vmware.vcloud.virtualHardwareSection+xml" xmlns:vcloud="http://www.vmware.com/vcloud/v1.5">
	        <ovf:Info>Virtual hardware requirements</ovf:Info>
	        <ovf:System>
	          <vssd:ElementName>Virtual Hardware Family</vssd:ElementName>
	          <vssd:InstanceID>0</vssd:InstanceID>
	          <vssd:VirtualSystemIdentifier>CentOS64-32bit</vssd:VirtualSystemIdentifier>
	          <vssd:VirtualSystemType>vmx-09</vssd:VirtualSystemType>
	        </ovf:System>
	        <ovf:Item>
	          <rasd:Address>00:50:56:02:0b:36</rasd:Address>
	          <rasd:AddressOnParent>0</rasd:AddressOnParent>
	          <rasd:AutomaticAllocation>false</rasd:AutomaticAllocation>
	          <rasd:Connection vcloud:ipAddressingMode="NONE" vcloud:primaryNetworkConnection="true">none</rasd:Connection>
	          <rasd:Description>E1000 ethernet adapter on &quot;none&quot;</rasd:Description>
	          <rasd:ElementName>Network adapter 0</rasd:ElementName>
	          <rasd:InstanceID>1</rasd:InstanceID>
	          <rasd:ResourceSubType>E1000</rasd:ResourceSubType>
	          <rasd:ResourceType>10</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item>
	          <rasd:Address>0</rasd:Address>
	          <rasd:Description>SCSI Controller</rasd:Description>
	          <rasd:ElementName>SCSI Controller 0</rasd:ElementName>
	          <rasd:InstanceID>2</rasd:InstanceID>
	          <rasd:ResourceSubType>lsilogic</rasd:ResourceSubType>
	          <rasd:ResourceType>6</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item>
	          <rasd:AddressOnParent>0</rasd:AddressOnParent>
	          <rasd:Description>Hard disk</rasd:Description>
	          <rasd:ElementName>Hard disk 1</rasd:ElementName>
	          <rasd:HostResource vcloud:busSubType="lsilogic" vcloud:busType="6" vcloud:capacity="20480" vcloud:storageProfileHref="http://localhost:4444/api/vdcStorageProfile/816409e1-6207-4a1f-bd45-947cd03d6452" vcloud:storageProfileOverrideVmDefault="false"/>
	          <rasd:InstanceID>2000</rasd:InstanceID>
	          <rasd:Parent>2</rasd:Parent>
	          <rasd:ResourceType>17</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item>
	          <rasd:Address>1</rasd:Address>
	          <rasd:Description>IDE Controller</rasd:Description>
	          <rasd:ElementName>IDE Controller 1</rasd:ElementName>
	          <rasd:InstanceID>3</rasd:InstanceID>
	          <rasd:ResourceType>5</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item>
	          <rasd:AddressOnParent>0</rasd:AddressOnParent>
	          <rasd:AutomaticAllocation>false</rasd:AutomaticAllocation>
	          <rasd:Description>CD/DVD Drive</rasd:Description>
	          <rasd:ElementName>CD/DVD Drive 1</rasd:ElementName>
	          <rasd:HostResource/>
	          <rasd:InstanceID>3002</rasd:InstanceID>
	          <rasd:Parent>3</rasd:Parent>
	          <rasd:ResourceType>15</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item>
	          <rasd:AddressOnParent>0</rasd:AddressOnParent>
	          <rasd:AutomaticAllocation>false</rasd:AutomaticAllocation>
	          <rasd:Description>Floppy Drive</rasd:Description>
	          <rasd:ElementName>Floppy Drive 1</rasd:ElementName>
	          <rasd:HostResource/>
	          <rasd:InstanceID>8000</rasd:InstanceID>
	          <rasd:ResourceType>14</rasd:ResourceType>
	        </ovf:Item>
	        <ovf:Item vcloud:href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/cpu" vcloud:type="application/vnd.vmware.vcloud.rasdItem+xml">
	          <rasd:AllocationUnits>hertz * 10^6</rasd:AllocationUnits>
	          <rasd:Description>Number of Virtual CPUs</rasd:Description>
	          <rasd:ElementName>1 virtual CPU(s)</rasd:ElementName>
	          <rasd:InstanceID>4</rasd:InstanceID>
	          <rasd:Reservation>0</rasd:Reservation>
	          <rasd:ResourceType>3</rasd:ResourceType>
	          <rasd:VirtualQuantity>1</rasd:VirtualQuantity>
	          <rasd:Weight>0</rasd:Weight>
	          <vmw:CoresPerSocket ovf:required="false">1</vmw:CoresPerSocket>
	          <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/cpu" rel="edit" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        </ovf:Item>
	        <ovf:Item vcloud:href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/memory" vcloud:type="application/vnd.vmware.vcloud.rasdItem+xml">
	          <rasd:AllocationUnits>byte * 2^20</rasd:AllocationUnits>
	          <rasd:Description>Memory Size</rasd:Description>
	          <rasd:ElementName>1024 MB of memory</rasd:ElementName>
	          <rasd:InstanceID>5</rasd:InstanceID>
	          <rasd:Reservation>0</rasd:Reservation>
	          <rasd:ResourceType>4</rasd:ResourceType>
	          <rasd:VirtualQuantity>1024</rasd:VirtualQuantity>
	          <rasd:Weight>0</rasd:Weight>
	          <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/memory" rel="edit" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        </ovf:Item>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/" rel="edit" type="application/vnd.vmware.vcloud.virtualHardwareSection+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/cpu" rel="down" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/cpu" rel="edit" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/memory" rel="down" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/memory" rel="edit" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/disks" rel="down" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/disks" rel="edit" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/media" rel="down" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/networkCards" rel="down" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/networkCards" rel="edit" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/serialPorts" rel="down" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/virtualHardwareSection/serialPorts" rel="edit" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
	      </ovf:VirtualHardwareSection>
	      <ovf:OperatingSystemSection ovf:id="36" vcloud:href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/operatingSystemSection/" vcloud:type="application/vnd.vmware.vcloud.operatingSystemSection+xml" vmw:osType="centosGuest" xmlns:vcloud="http://www.vmware.com/vcloud/v1.5">
	        <ovf:Info>Specifies the operating system installed</ovf:Info>
	        <ovf:Description>CentOS 4/5/6 (32-bit)</ovf:Description>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/operatingSystemSection/" rel="edit" type="application/vnd.vmware.vcloud.operatingSystemSection+xml"/>
	      </ovf:OperatingSystemSection>
	      <NetworkConnectionSection href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/networkConnectionSection/" ovf:required="false" type="application/vnd.vmware.vcloud.networkConnectionSection+xml">
	        <ovf:Info>Specifies the available VM network connections</ovf:Info>
	        <PrimaryNetworkConnectionIndex>0</PrimaryNetworkConnectionIndex>
	        <NetworkConnection needsCustomization="true" network="none">
	          <NetworkConnectionIndex>0</NetworkConnectionIndex>
	          <IsConnected>false</IsConnected>
	          <MACAddress>00:50:56:02:0b:36</MACAddress>
	          <IpAddressAllocationMode>NONE</IpAddressAllocationMode>
	        </NetworkConnection>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/networkConnectionSection/" rel="edit" type="application/vnd.vmware.vcloud.networkConnectionSection+xml"/>
	      </NetworkConnectionSection>
	      <GuestCustomizationSection href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/guestCustomizationSection/" ovf:required="false" type="application/vnd.vmware.vcloud.guestCustomizationSection+xml">
	        <ovf:Info>Specifies Guest OS Customization Settings</ovf:Info>
	        <Enabled>true</Enabled>
	        <ChangeSid>false</ChangeSid>
	        <VirtualMachineId>00000000-0000-0000-0000-000000000000</VirtualMachineId>
	        <JoinDomainEnabled>false</JoinDomainEnabled>
	        <UseOrgSettings>false</UseOrgSettings>
	        <AdminPasswordEnabled>true</AdminPasswordEnabled>
	        <AdminPasswordAuto>true</AdminPasswordAuto>
	        <AdminAutoLogonEnabled>false</AdminAutoLogonEnabled>
	        <AdminAutoLogonCount>0</AdminAutoLogonCount>
	        <ResetPasswordRequired>true</ResetPasswordRequired>
	        <ComputerName>cts-6.4-32bit</ComputerName>
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/guestCustomizationSection/" rel="edit" type="application/vnd.vmware.vcloud.guestCustomizationSection+xml"/>
	      </GuestCustomizationSection>
	      <ovf:ProductSection ovf:class="" ovf:instance="" ovf:required="true">
	        <ovf:Property ovf:key="guestinfo.hostname" ovf:password="false" ovf:type="string" ovf:userConfigurable="true" ovf:value="">
	          <ovf:Label>Hostname</ovf:Label>
	          <ovf:Description>Hostname</ovf:Description>
	          <ovf:Value ovf:value="coreos01"/>
	        </ovf:Property>
	      </ovf:ProductSection>
	      <RuntimeInfoSection vcloud:href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/runtimeInfoSection" vcloud:type="application/vnd.vmware.vcloud.virtualHardwareSection+xml" xmlns:vcloud="http://www.vmware.com/vcloud/v1.5">
	        <ovf:Info>Specifies Runtime info</ovf:Info>
	        <VMWareTools version="9283"/>
	      </RuntimeInfoSection>
	      <SnapshotSection href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/snapshotSection" ovf:required="false" type="application/vnd.vmware.vcloud.snapshotSection+xml">
	        <ovf:Info>Snapshot information section</ovf:Info>
	      </SnapshotSection>
	      <VAppScopedLocalId>CentOS64-32bit</VAppScopedLocalId>
	      <VmCapabilities href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/vmCapabilities/" type="application/vnd.vmware.vcloud.vmCapabilitiesSection+xml">
	        <Link href="http://localhost:4444/api/vApp/vm-00000000-0000-0000-0000-000000000000/vmCapabilities/" rel="edit" type="application/vnd.vmware.vcloud.vmCapabilitiesSection+xml"/>
	        <MemoryHotAddEnabled>false</MemoryHotAddEnabled>
	        <CpuHotAddEnabled>false</CpuHotAddEnabled>
	      </VmCapabilities>
	      <StorageProfile href="http://localhost:4444/api/vdcStorageProfile/816409e1-6207-4a1f-bd45-947cd03d6452" name="SSD-Accelerated" type="application/vnd.vmware.vcloud.vdcStorageProfile+xml"/>
	    </Vm>
	  </Children>
	</VApp>
	`

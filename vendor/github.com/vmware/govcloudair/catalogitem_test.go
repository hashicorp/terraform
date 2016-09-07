/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	. "gopkg.in/check.v1"
)

func (s *S) Test_GetVAppTemplate(c *C) {

	// Get the Org populated
	testServer.Response(200, nil, orgExample)
	org, err := s.vdc.GetVDCOrg()
	_ = testServer.WaitRequest()
	testServer.Flush()
	c.Assert(err, IsNil)

	// Populate Catalog
	testServer.Response(200, nil, catalogExample)
	cat, err := org.FindCatalog("Public Catalog")
	_ = testServer.WaitRequest()
	testServer.Flush()

	// Populate Catalog Item
	testServer.Response(200, nil, catalogitemExample)
	catitem, err := cat.FindCatalogItem("CentOS64-32bit")
	_ = testServer.WaitRequest()
	testServer.Flush()

	// Get VAppTemplate
	testServer.Response(200, nil, vapptemplateExample)
	vapptemplate, err := catitem.GetVAppTemplate()
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(vapptemplate.VAppTemplate.HREF, Equals, "http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5")
	c.Assert(vapptemplate.VAppTemplate.Name, Equals, "CentOS64-32bit")
	c.Assert(vapptemplate.VAppTemplate.Description, Equals, "id: cts-6.4-32bit")

}

var catalogitemExample = `
	<?xml version="1.0" ?>
	<CatalogItem href="http://localhost:4444/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae" id="urn:vcloud:catalogitem:1176e485-8858-4e15-94e5-ae4face605ae" name="CentOS64-32bit" size="0" type="application/vnd.vmware.vcloud.catalogItem+xml" xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
		<Link href="http://localhost:4444/api/catalog/e8a20fdf-8a78-440c-ac71-0420db59f854" rel="up" type="application/vnd.vmware.vcloud.catalog+xml"/>
		<Link href="http://localhost:4444/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae/metadata" rel="down" type="application/vnd.vmware.vcloud.metadata+xml"/>
		<Description>id: cts-6.4-32bit</Description>
		<Entity href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5" name="CentOS64-32bit" type="application/vnd.vmware.vcloud.vAppTemplate+xml"/>
		<DateCreated>2014-06-04T21:06:43.750Z</DateCreated>
		<VersionNumber>4</VersionNumber>
	</CatalogItem>
`

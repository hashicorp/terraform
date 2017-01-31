/*
 * Copyright 2016 Skyscape Cloud Services.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	// "fmt"
	"github.com/vmware/govcloudair/testutil"
	. "gopkg.in/check.v1"
)

func (s *K) Test_FindVMByHREF(c *C) {

	// Get the Org populated
	testServer.ResponseMap(1, testutil.ResponseMap{
		"/api/vApp/vm-11111111-1111-1111-1111-111111111111": testutil.Response{200, nil, vmExample},
	})

	vm_href := vcdu_api.String() + "/vApp/vm-11111111-1111-1111-1111-111111111111"
	vm, err := s.client.FindVMByHREF(vm_href)
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(vm.VM.Name, Equals, "testvmxnet")
	c.Assert(vm.VM.VirtualHardwareSection.Item, NotNil)
}

var vmExample = `<?xml version="1.0" encoding="UTF-8"?>
<Vm xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:ovf="http://schemas.dmtf.org/ovf/envelope/1" xmlns:vssd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData" xmlns:rasd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData" xmlns:vmw="http://www.vmware.com/schema/ovf" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" needsCustomization="true" nestedHypervisorEnabled="false" deployed="false" status="8" name="testvmxnet" id="urn:vcloud:vm:11111111-1111-1111-1111-111111111111" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111" type="application/vnd.vmware.vcloud.vm+xml" xsi:schemaLocation="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_VirtualSystemSettingData.xsd http://www.vmware.com/schema/ovf http://www.vmware.com/schema/ovf http://schemas.dmtf.org/ovf/envelope/1 http://schemas.dmtf.org/ovf/envelope/1/dsp8023_1.1.0.xsd http://www.vmware.com/vcloud/v1.5 http://10.10.6.11/api/v1.5/schema/master.xsd http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_ResourceAllocationSettingData.xsd">
    <Link rel="power:powerOn" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/power/action/powerOn"/>
    <Link rel="deploy" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/action/deploy" type="application/vnd.vmware.vcloud.deployVAppParams+xml"/>
    <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111" type="application/vnd.vmware.vcloud.vm+xml"/>
    <Link rel="remove" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111"/>
    <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/metadata" type="application/vnd.vmware.vcloud.metadata+xml"/>
    <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/productSections/" type="application/vnd.vmware.vcloud.productSections+xml"/>
    <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/metrics/historic" type="application/vnd.vmware.vcloud.metrics.historicUsageSpec+xml"/>
    <Link rel="metrics" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/metrics/historic" type="application/vnd.vmware.vcloud.metrics.historicUsageSpec+xml"/>
    <Link rel="screen:thumbnail" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/screen"/>
    <Link rel="media:insertMedia" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/media/action/insertMedia" type="application/vnd.vmware.vcloud.mediaInsertOrEjectParams+xml"/>
    <Link rel="media:ejectMedia" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/media/action/ejectMedia" type="application/vnd.vmware.vcloud.mediaInsertOrEjectParams+xml"/>
    <Link rel="disk:attach" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/disk/action/attach" type="application/vnd.vmware.vcloud.diskAttachOrDetachParams+xml"/>
    <Link rel="disk:detach" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/disk/action/detach" type="application/vnd.vmware.vcloud.diskAttachOrDetachParams+xml"/>
    <Link rel="enable" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/action/enableNestedHypervisor"/>
    <Link rel="customizeAtNextPowerOn" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/action/customizeAtNextPowerOn"/>
    <Link rel="snapshot:create" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/action/createSnapshot" type="application/vnd.vmware.vcloud.createSnapshotParams+xml"/>
    <Link rel="reconfigureVm" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/action/reconfigureVm" name="testvmxnet" type="application/vnd.vmware.vcloud.vm+xml"/>
    <Link rel="up" href="http://localhost:4444/api/vApp/vapp-22222222-2222-2222-2222-222222222222" type="application/vnd.vmware.vcloud.vApp+xml"/>
    <Description/>
    <ovf:VirtualHardwareSection xmlns:vcloud="http://www.vmware.com/vcloud/v1.5" ovf:transport="" vcloud:href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/" vcloud:type="application/vnd.vmware.vcloud.virtualHardwareSection+xml">
        <ovf:Info>Virtual hardware requirements</ovf:Info>
        <ovf:System>
            <vssd:ElementName>Virtual Hardware Family</vssd:ElementName>
            <vssd:InstanceID>0</vssd:InstanceID>
            <vssd:VirtualSystemIdentifier>testvmxnet</vssd:VirtualSystemIdentifier>
            <vssd:VirtualSystemType>vmx-09</vssd:VirtualSystemType>
        
        </ovf:System>
        <ovf:Item>
            <rasd:Address>00:50:56:01:35:88</rasd:Address>
            <rasd:AddressOnParent>1</rasd:AddressOnParent>
            <rasd:AutomaticAllocation>true</rasd:AutomaticAllocation>
            <rasd:Connection vcloud:ipAddress="10.40.0.11" vcloud:primaryNetworkConnection="true" vcloud:ipAddressingMode="POOL">Development Network</rasd:Connection>
            <rasd:Description>Vmxnet3 ethernet adapter on "Development Network"</rasd:Description>
            <rasd:ElementName>Network adapter 1</rasd:ElementName>
            <rasd:InstanceID>1</rasd:InstanceID>
            <rasd:ResourceSubType>VMXNET3</rasd:ResourceSubType>
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
            <rasd:HostResource vcloud:capacity="65536" vcloud:storageProfileOverrideVmDefault="false" vcloud:busSubType="lsilogic" vcloud:storageProfileHref="http://localhost:4444/api/vdcStorageProfile/33333333-3333-3333-3333-333333333333" vcloud:busType="6"/>
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
        <ovf:Item vcloud:href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/cpu" vcloud:type="application/vnd.vmware.vcloud.rasdItem+xml">
            <rasd:AllocationUnits>hertz * 10^6</rasd:AllocationUnits>
            <rasd:Description>Number of Virtual CPUs</rasd:Description>
            <rasd:ElementName>1 virtual CPU(s)</rasd:ElementName>
            <rasd:InstanceID>4</rasd:InstanceID>
            <rasd:Reservation>0</rasd:Reservation>
            <rasd:ResourceType>3</rasd:ResourceType>
            <rasd:VirtualQuantity>1</rasd:VirtualQuantity>
            <rasd:Weight>0</rasd:Weight>
            <vmw:CoresPerSocket ovf:required="false">1</vmw:CoresPerSocket>
            <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/cpu" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        
        </ovf:Item>
        <ovf:Item vcloud:href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/memory" vcloud:type="application/vnd.vmware.vcloud.rasdItem+xml">
            <rasd:AllocationUnits>byte * 2^20</rasd:AllocationUnits>
            <rasd:Description>Memory Size</rasd:Description>
            <rasd:ElementName>512 MB of memory</rasd:ElementName>
            <rasd:InstanceID>5</rasd:InstanceID>
            <rasd:Reservation>0</rasd:Reservation>
            <rasd:ResourceType>4</rasd:ResourceType>
            <rasd:VirtualQuantity>512</rasd:VirtualQuantity>
            <rasd:Weight>0</rasd:Weight>
            <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/memory" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        
        </ovf:Item>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/" type="application/vnd.vmware.vcloud.virtualHardwareSection+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/cpu" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/cpu" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/memory" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/memory" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/disks" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/disks" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/media" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/networkCards" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/networkCards" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="down" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/serialPorts" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/virtualHardwareSection/serialPorts" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
    
    </ovf:VirtualHardwareSection>
    <ovf:OperatingSystemSection xmlns:vcloud="http://www.vmware.com/vcloud/v1.5" ovf:id="101" vcloud:href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/operatingSystemSection/" vcloud:type="application/vnd.vmware.vcloud.operatingSystemSection+xml" vmw:osType="centos64Guest">
        <ovf:Info>Specifies the operating system installed</ovf:Info>
        <ovf:Description>CentOS 4/5/6/7 (64-bit)</ovf:Description>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/operatingSystemSection/" type="application/vnd.vmware.vcloud.operatingSystemSection+xml"/>
    
    </ovf:OperatingSystemSection>
    <NetworkConnectionSection href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/networkConnectionSection/" type="application/vnd.vmware.vcloud.networkConnectionSection+xml" ovf:required="false">
        <ovf:Info>Specifies the available VM network connections</ovf:Info>
        <PrimaryNetworkConnectionIndex>1</PrimaryNetworkConnectionIndex>
        <NetworkConnection needsCustomization="true" network="Development Network">
            <NetworkConnectionIndex>1</NetworkConnectionIndex>
            <IpAddress>10.40.0.11</IpAddress>
            <IsConnected>true</IsConnected>
            <MACAddress>00:50:56:01:35:88</MACAddress>
            <IpAddressAllocationMode>POOL</IpAddressAllocationMode>
        
        </NetworkConnection>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/networkConnectionSection/" type="application/vnd.vmware.vcloud.networkConnectionSection+xml"/>
    
    </NetworkConnectionSection>
    <GuestCustomizationSection href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/guestCustomizationSection/" type="application/vnd.vmware.vcloud.guestCustomizationSection+xml" ovf:required="false">
        <ovf:Info>Specifies Guest OS Customization Settings</ovf:Info>
        <Enabled>true</Enabled>
        <ChangeSid>false</ChangeSid>
        <VirtualMachineId>11111111-1111-1111-1111-111111111111</VirtualMachineId>
        <JoinDomainEnabled>false</JoinDomainEnabled>
        <UseOrgSettings>false</UseOrgSettings>
        <AdminPasswordEnabled>true</AdminPasswordEnabled>
        <AdminPasswordAuto>true</AdminPasswordAuto>
        <AdminAutoLogonEnabled>false</AdminAutoLogonEnabled>
        <AdminAutoLogonCount>0</AdminAutoLogonCount>
        <ResetPasswordRequired>false</ResetPasswordRequired>
        <ComputerName>testvmxnet3</ComputerName>
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/guestCustomizationSection/" type="application/vnd.vmware.vcloud.guestCustomizationSection+xml"/>
    
    </GuestCustomizationSection>
    <RuntimeInfoSection xmlns:vcloud="http://www.vmware.com/vcloud/v1.5" vcloud:href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/runtimeInfoSection" vcloud:type="application/vnd.vmware.vcloud.virtualHardwareSection+xml">
        <ovf:Info>Specifies Runtime info</ovf:Info>
        <VMWareTools version="2147483647"/>
    
    </RuntimeInfoSection>
    <SnapshotSection href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/snapshotSection" type="application/vnd.vmware.vcloud.snapshotSection+xml" ovf:required="false">
        <ovf:Info>Snapshot information section</ovf:Info>
    
    </SnapshotSection>
    <VAppScopedLocalId>vm</VAppScopedLocalId>
    <VmCapabilities href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/vmCapabilities/" type="application/vnd.vmware.vcloud.vmCapabilitiesSection+xml">
        <Link rel="edit" href="http://localhost:4444/api/vApp/vm-11111111-1111-1111-1111-111111111111/vmCapabilities/" type="application/vnd.vmware.vcloud.vmCapabilitiesSection+xml"/>
        <MemoryHotAddEnabled>false</MemoryHotAddEnabled>
        <CpuHotAddEnabled>false</CpuHotAddEnabled>
    
    </VmCapabilities>
    <StorageProfile href="http://localhost:4444/api/vdcStorageProfile/33333333-3333-3333-3333-333333333333" name="BASIC-Any" type="application/vnd.vmware.vcloud.vdcStorageProfile+xml"/>
</Vm>
	`

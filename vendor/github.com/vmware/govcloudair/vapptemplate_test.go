/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

var vapptemplateExample = `
<?xml version="1.0" encoding="UTF-8"?>
<VAppTemplate xmlns="http://www.vmware.com/vcloud/v1.5" xmlns:ovf="http://schemas.dmtf.org/ovf/envelope/1" xmlns:vssd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData" xmlns:rasd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData" xmlns:vmw="http://www.vmware.com/schema/ovf" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" goldMaster="false" ovfDescriptorUploaded="true" status="8" name="CentOS64-32bit" id="urn:vcloud:vapptemplate:40cb9721-5f1a-44f9-b5c3-98c5f518c4f5" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5" type="application/vnd.vmware.vcloud.vAppTemplate+xml" xsi:schemaLocation="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_VirtualSystemSettingData.xsd http://www.vmware.com/schema/ovf http://www.vmware.com/schema/ovf http://schemas.dmtf.org/ovf/envelope/1 http://schemas.dmtf.org/ovf/envelope/1/dsp8023_1.1.0.xsd http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.22.0/CIM_ResourceAllocationSettingData.xsd">
    <Link rel="catalogItem" href="http://localhost:4444/api/catalogItem/1176e485-8858-4e15-94e5-ae4face605ae" type="application/vnd.vmware.vcloud.catalogItem+xml"/>
    <Link rel="enable" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/action/enableDownload"/>
    <Link rel="disable" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/action/disableDownload"/>
    <Link rel="ovf" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/ovf" type="text/xml"/>
    <Link rel="storageProfile" href="http://localhost:4444/api/vdcStorageProfile/9ae2c9f2-d711-4de9-a108-e4c064c33b60" name="SSD-Accelerated" type="application/vnd.vmware.vcloud.vdcStorageProfile+xml"/>
    <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/owner" type="application/vnd.vmware.vcloud.owner+xml"/>
    <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/metadata" type="application/vnd.vmware.vcloud.metadata+xml"/>
    <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/productSections/" type="application/vnd.vmware.vcloud.productSections+xml"/>
    <Description>id: cts-6.4-32bit</Description>
    <Owner type="application/vnd.vmware.vcloud.owner+xml">
        <User href="http://localhost:4444/api/admin/user/db5e3e0c-61ac-4e1e-93f3-d6096dde2517" name="system" type="application/vnd.vmware.admin.user+xml"/>
    </Owner>
    <Children>
        <Vm goldMaster="false" status="8" name="CentOS64-32bit" id="urn:vcloud:vm:3a3934d2-1f3f-4782-a911-f143da27e88c" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c" type="application/vnd.vmware.vcloud.vm+xml">
            <Link rel="up" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5" type="application/vnd.vmware.vcloud.vAppTemplate+xml"/>
            <Link rel="storageProfile" href="http://localhost:4444/api/vdcStorageProfile/9ae2c9f2-d711-4de9-a108-e4c064c33b60" type="application/vnd.vmware.vcloud.vdcStorageProfile+xml"/>
            <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/metadata" type="application/vnd.vmware.vcloud.metadata+xml"/>
            <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/productSections/" type="application/vnd.vmware.vcloud.productSections+xml"/>
            <Description>id: cts-6.4-32bit</Description>
            <NetworkConnectionSection href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/networkConnectionSection/" type="application/vnd.vmware.vcloud.networkConnectionSection+xml" ovf:required="false">
                <ovf:Info>Specifies the available VM network connections</ovf:Info>
                <PrimaryNetworkConnectionIndex>0</PrimaryNetworkConnectionIndex>
                <NetworkConnection needsCustomization="true" network="none">
                    <NetworkConnectionIndex>0</NetworkConnectionIndex>
                    <IsConnected>false</IsConnected>
                    <MACAddress>00:50:56:02:00:39</MACAddress>
                    <IpAddressAllocationMode>NONE</IpAddressAllocationMode>
                </NetworkConnection>
            </NetworkConnectionSection>
            <GuestCustomizationSection href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/guestCustomizationSection/" type="application/vnd.vmware.vcloud.guestCustomizationSection+xml" ovf:required="false">
                <ovf:Info>Specifies Guest OS Customization Settings</ovf:Info>
                <Enabled>true</Enabled>
                <ChangeSid>false</ChangeSid>
                <VirtualMachineId>3a3934d2-1f3f-4782-a911-f143da27e88c</VirtualMachineId>
                <JoinDomainEnabled>false</JoinDomainEnabled>
                <UseOrgSettings>false</UseOrgSettings>
                <AdminPasswordEnabled>true</AdminPasswordEnabled>
                <AdminPasswordAuto>true</AdminPasswordAuto>
                <AdminAutoLogonEnabled>false</AdminAutoLogonEnabled>
                <AdminAutoLogonCount>0</AdminAutoLogonCount>
                <ResetPasswordRequired>true</ResetPasswordRequired>
                <ComputerName>cts-6.4-32bit</ComputerName>
            </GuestCustomizationSection>
            <ovf:VirtualHardwareSection xmlns:vcloud="http://www.vmware.com/vcloud/v1.5" ovf:transport="" vcloud:href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/" vcloud:type="application/vnd.vmware.vcloud.virtualHardwareSection+xml">
                <ovf:Info>Virtual hardware requirements</ovf:Info>
                <ovf:System>
                    <vssd:ElementName>Virtual Hardware Family</vssd:ElementName>
                    <vssd:InstanceID>0</vssd:InstanceID>
                    <vssd:VirtualSystemIdentifier>CentOS64-32bit</vssd:VirtualSystemIdentifier>
                    <vssd:VirtualSystemType>vmx-09</vssd:VirtualSystemType>
                </ovf:System>
                <ovf:Item>
                    <rasd:Address>00:50:56:02:00:39</rasd:Address>
                    <rasd:AddressOnParent>0</rasd:AddressOnParent>
                    <rasd:AutomaticAllocation>false</rasd:AutomaticAllocation>
                    <rasd:Connection vcloud:primaryNetworkConnection="true" vcloud:ipAddressingMode="NONE">none</rasd:Connection>
                    <rasd:Description>E1000 ethernet adapter on "none"</rasd:Description>
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
                    <rasd:HostResource vcloud:capacity="20480" vcloud:storageProfileOverrideVmDefault="false" vcloud:busSubType="lsilogic" vcloud:storageProfileHref="http://localhost:4444/api/vdcStorageProfile/9ae2c9f2-d711-4de9-a108-e4c064c33b60" vcloud:busType="6"/>
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
                <ovf:Item>
                    <rasd:AllocationUnits>hertz * 10^6</rasd:AllocationUnits>
                    <rasd:Description>Number of Virtual CPUs</rasd:Description>
                    <rasd:ElementName>1 virtual CPU(s)</rasd:ElementName>
                    <rasd:InstanceID>4</rasd:InstanceID>
                    <rasd:Reservation>0</rasd:Reservation>
                    <rasd:ResourceType>3</rasd:ResourceType>
                    <rasd:VirtualQuantity>1</rasd:VirtualQuantity>
                    <rasd:Weight>0</rasd:Weight>
                    <vmw:CoresPerSocket ovf:required="false">1</vmw:CoresPerSocket>
                </ovf:Item>
                <ovf:Item>
                    <rasd:AllocationUnits>byte * 2^20</rasd:AllocationUnits>
                    <rasd:Description>Memory Size</rasd:Description>
                    <rasd:ElementName>1024 MB of memory</rasd:ElementName>
                    <rasd:InstanceID>5</rasd:InstanceID>
                    <rasd:Reservation>0</rasd:Reservation>
                    <rasd:ResourceType>4</rasd:ResourceType>
                    <rasd:VirtualQuantity>1024</rasd:VirtualQuantity>
                    <rasd:Weight>0</rasd:Weight>
                </ovf:Item>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/cpu" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/memory" type="application/vnd.vmware.vcloud.rasdItem+xml"/>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/disks" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/media" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/networkCards" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
                <Link rel="down" href="http://localhost:4444/api/vAppTemplate/vm-3a3934d2-1f3f-4782-a911-f143da27e88c/virtualHardwareSection/serialPorts" type="application/vnd.vmware.vcloud.rasdItemsList+xml"/>
            </ovf:VirtualHardwareSection>
            <VAppScopedLocalId>CentOS64-32bit</VAppScopedLocalId>
            <DateCreated>2014-06-04T21:06:43.547Z</DateCreated>
        </Vm>
    </Children>
    <ovf:NetworkSection xmlns:vcloud="http://www.vmware.com/vcloud/v1.5" vcloud:href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/networkSection/" vcloud:type="application/vnd.vmware.vcloud.networkSection+xml">
        <ovf:Info>The list of logical networks</ovf:Info>
        <ovf:Network ovf:name="none">
            <ovf:Description>This is a special place-holder used for disconnected network interfaces.</ovf:Description>
        </ovf:Network>
    </ovf:NetworkSection>
    <NetworkConfigSection href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/networkConfigSection/" type="application/vnd.vmware.vcloud.networkConfigSection+xml" ovf:required="false">
        <ovf:Info>The configuration parameters for logical networks</ovf:Info>
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
    <LeaseSettingsSection href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/leaseSettingsSection/" type="application/vnd.vmware.vcloud.leaseSettingsSection+xml" ovf:required="false">
        <ovf:Info>Lease settings section</ovf:Info>
        <Link rel="edit" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/leaseSettingsSection/" type="application/vnd.vmware.vcloud.leaseSettingsSection+xml"/>
        <StorageLeaseInSeconds>0</StorageLeaseInSeconds>
    </LeaseSettingsSection>
    <CustomizationSection goldMaster="false" href="http://localhost:4444/api/vAppTemplate/vappTemplate-40cb9721-5f1a-44f9-b5c3-98c5f518c4f5/customizationSection/" type="application/vnd.vmware.vcloud.customizationSection+xml" ovf:required="false">
        <ovf:Info>VApp template customization section</ovf:Info>
        <CustomizeOnInstantiate>true</CustomizeOnInstantiate>
    </CustomizationSection>
    <DateCreated>2014-06-04T21:06:43.547Z</DateCreated>
</VAppTemplate>

	`

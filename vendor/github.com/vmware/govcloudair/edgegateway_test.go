/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"github.com/vmware/govcloudair/testutil"

	. "gopkg.in/check.v1"
)

func (s *S) Test_Refresh(c *C) {

	// Get the Org populated
	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/vdc/00000000-0000-0000-0000-000000000000/edgeGateways":  testutil.Response{200, nil, edgegatewayqueryresultsExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000": testutil.Response{200, nil, edgegatewayExample},
	})

	edge, err := s.vdc.FindEdgeGateway("M916272752-5793")
	_ = testServer.WaitRequests(2)
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(edge.EdgeGateway.Name, Equals, "M916272752-5793")

	testServer.Response(200, nil, edgegatewayExample)
	err = edge.Refresh()
	_ = testServer.WaitRequest()
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(edge.EdgeGateway.Name, Equals, "M916272752-5793")

}

func (s *S) Test_NATMapping(c *C) {
	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/vdc/00000000-0000-0000-0000-000000000000/edgeGateways":  testutil.Response{200, nil, edgegatewayqueryresultsExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000": testutil.Response{200, nil, edgegatewayExample},
	})

	edge, err := s.vdc.FindEdgeGateway("M916272752-5793")
	_ = testServer.WaitRequests(2)
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(edge.EdgeGateway.Name, Equals, "M916272752-5793")

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, edgegatewayExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/configureServices": testutil.Response{200, nil, taskExample},
	})

	_, err = edge.AddNATMapping("DNAT", "10.0.0.1", "20.0.0.2", "77")
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, edgegatewayExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/configureServices": testutil.Response{200, nil, taskExample},
	})

	_, err = edge.RemoveNATMapping("DNAT", "10.0.0.1", "20.0.0.2", "77")
	_ = testServer.WaitRequest()

	c.Assert(err, IsNil)

}

func (s *S) Test_1to1Mappings(c *C) {

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/vdc/00000000-0000-0000-0000-000000000000/edgeGateways":  testutil.Response{200, nil, edgegatewayqueryresultsExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000": testutil.Response{200, nil, edgegatewayExample},
	})

	edge, err := s.vdc.FindEdgeGateway("M916272752-5793")
	_ = testServer.WaitRequests(2)
	testServer.Flush()

	c.Assert(err, IsNil)
	c.Assert(edge.EdgeGateway.Name, Equals, "M916272752-5793")

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, edgegatewayExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/configureServices": testutil.Response{200, nil, taskExample},
	})

	_, err = edge.Create1to1Mapping("10.0.0.1", "20.0.0.2", "description")
	_ = testServer.WaitRequests(2)

	c.Assert(err, IsNil)

	testServer.ResponseMap(2, testutil.ResponseMap{
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000":                          testutil.Response{200, nil, edgegatewayExample},
		"/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/configureServices": testutil.Response{200, nil, taskExample},
	})

	_, err = edge.Remove1to1Mapping("10.0.0.1", "20.0.0.2")
	_ = testServer.WaitRequests(2)

	c.Assert(err, IsNil)

}

var edgegatewayqueryresultsExample = `
<QueryResultRecords xmlns="http://www.vmware.com/vcloud/v1.5" name="edgeGateway" page="1" pageSize="25" total="1" href="http://localhost:4444/api/admin/vdc/00000000-0000-0000-0000-000000000000/edgeGateways?page=1&amp;pageSize=25&amp;format=records" type="application/vnd.vmware.vcloud.query.records+xml" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
    <Link rel="alternate" href="http://localhost:4444/api/admin/vdc/00000000-0000-0000-0000-000000000000/edgeGateways?page=1&amp;pageSize=25&amp;format=references" type="application/vnd.vmware.vcloud.query.references+xml"/>
    <Link rel="alternate" href="http://localhost:4444/api/admin/vdc/00000000-0000-0000-0000-000000000000/edgeGateways?page=1&amp;pageSize=25&amp;format=idrecords" type="application/vnd.vmware.vcloud.query.idrecords+xml"/>
    <EdgeGatewayRecord gatewayStatus="READY" haStatus="UP" isBusy="false" name="M916272752-5793" numberOfExtNetworks="1" numberOfOrgNetworks="2" vdc="http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000" isSyslogServerSettingInSync="true" taskStatus="success" taskOperation="networkConfigureEdgeGatewayServices" task="http://localhost:4444/api/task/eca0d70f-36e0-428a-93c2-311b792f3326" taskDetails=" "/>
</QueryResultRecords>
`

var edgegatewayExample = `
<EdgeGateway xmlns="http://www.vmware.com/vcloud/v1.5" status="1" name="M916272752-5793" id="urn:vcloud:gateway:00000000-0000-0000-0000-000000000000" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000" type="application/vnd.vmware.admin.edgeGateway+xml" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.vmware.com/vcloud/v1.5 http://10.6.32.3/api/v1.5/schema/master.xsd">
    <Link rel="up" href="http://localhost:4444/api/vdc/214cd6b2-3f7a-4ee5-9b0a-52b4001a4a84" type="application/vnd.vmware.vcloud.vdc+xml"/>
    <Link rel="edgeGateway:redeploy" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/redeploy"/>
    <Link rel="edgeGateway:configureServices" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/configureServices" type="application/vnd.vmware.admin.edgeGatewayServiceConfiguration+xml"/>
    <Link rel="edgeGateway:reapplyServices" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/reapplyServices"/>
    <Link rel="edgeGateway:syncSyslogSettings" href="http://localhost:4444/api/admin/edgeGateway/00000000-0000-0000-0000-000000000000/action/syncSyslogServerSettings"/>
    <Configuration>
        <GatewayBackingConfig>compact</GatewayBackingConfig>
        <GatewayInterfaces>
            <GatewayInterface>
                <Name>JGray Network</Name>
                <DisplayName>JGray Network</DisplayName>
                <Network href="http://localhost:4444/api/admin/network/32e903a4-c726-497f-9ec7-e4a43b58bdbc" name="JGray Network" type="application/vnd.vmware.admin.network+xml"/>
                <InterfaceType>internal</InterfaceType>
                <SubnetParticipation>
                    <Gateway>192.168.108.1</Gateway>
                    <Netmask>255.255.255.0</Netmask>
                    <IpAddress>192.168.108.1</IpAddress>
                </SubnetParticipation>
                <ApplyRateLimit>false</ApplyRateLimit>
                <UseForDefaultRoute>false</UseForDefaultRoute>
            </GatewayInterface>
            <GatewayInterface>
                <Name>M916272752-5793-default-routed</Name>
                <DisplayName>M916272752-5793-default-routed</DisplayName>
                <Network href="http://localhost:4444/api/admin/network/cb0f4c9e-1a46-49d4-9fcb-d228000a6bc1" name="M916272752-5793-default-routed" type="application/vnd.vmware.admin.network+xml"/>
                <InterfaceType>internal</InterfaceType>
                <SubnetParticipation>
                    <Gateway>192.168.109.1</Gateway>
                    <Netmask>255.255.255.0</Netmask>
                    <IpAddress>192.168.109.1</IpAddress>
                </SubnetParticipation>
                <ApplyRateLimit>false</ApplyRateLimit>
                <UseForDefaultRoute>false</UseForDefaultRoute>
            </GatewayInterface>
            <GatewayInterface>
                <Name>d2p3-ext</Name>
                <DisplayName>d2p3-ext</DisplayName>
                <Network href="http://localhost:4444/api/admin/network/6254f107-9876-4d03-986f-8bec7a4bcb3f" name="d2p3-ext" type="application/vnd.vmware.admin.network+xml"/>
                <InterfaceType>uplink</InterfaceType>
                <SubnetParticipation>
                    <Gateway>23.92.224.1</Gateway>
                    <Netmask>255.255.254.0</Netmask>
                    <IpAddress>23.92.225.51</IpAddress>
                    <IpRanges>
                        <IpRange>
                            <StartAddress>23.92.225.73</StartAddress>
                            <EndAddress>23.92.225.75</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.27</StartAddress>
                            <EndAddress>23.92.225.27</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.11</StartAddress>
                            <EndAddress>23.92.225.11</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.54</StartAddress>
                            <EndAddress>23.92.225.54</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.224.34</StartAddress>
                            <EndAddress>23.92.224.34</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.5</StartAddress>
                            <EndAddress>23.92.225.5</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.34</StartAddress>
                            <EndAddress>23.92.225.34</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.224.255</StartAddress>
                            <EndAddress>23.92.224.255</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.49</StartAddress>
                            <EndAddress>23.92.225.51</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.62</StartAddress>
                            <EndAddress>23.92.225.62</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.43</StartAddress>
                            <EndAddress>23.92.225.43</EndAddress>
                        </IpRange>
                        <IpRange>
                            <StartAddress>23.92.225.30</StartAddress>
                            <EndAddress>23.92.225.30</EndAddress>
                        </IpRange>
                    </IpRanges>
                </SubnetParticipation>
                <ApplyRateLimit>true</ApplyRateLimit>
                <InRateLimit>1024.0</InRateLimit>
                <OutRateLimit>1024.0</OutRateLimit>
                <UseForDefaultRoute>true</UseForDefaultRoute>
            </GatewayInterface>
        </GatewayInterfaces>
        <EdgeGatewayServiceConfiguration>
            <FirewallService>
                <IsEnabled>true</IsEnabled>
                <DefaultAction>drop</DefaultAction>
                <LogDefaultAction>false</LogDefaultAction>
                <FirewallRule>
                    <Id>1</Id>
                    <IsEnabled>true</IsEnabled>
                    <MatchOnTranslate>false</MatchOnTranslate>
                    <Description>ssh prd-001</Description>
                    <Policy>allow</Policy>
                    <Protocols>
                        <Tcp>true</Tcp>
                    </Protocols>
                    <Port>999</Port>
                    <DestinationPortRange>999</DestinationPortRange>
                    <DestinationIp>23.92.225.51</DestinationIp>
                    <SourcePort>-1</SourcePort>
                    <SourcePortRange>Any</SourcePortRange>
                    <SourceIp>60.241.110.111</SourceIp>
                    <EnableLogging>false</EnableLogging>
                </FirewallRule>
                <FirewallRule>
                    <Id>2</Id>
                    <IsEnabled>true</IsEnabled>
                    <MatchOnTranslate>false</MatchOnTranslate>
                    <Description>inet prd-001</Description>
                    <Policy>allow</Policy>
                    <Protocols>
                        <Any>true</Any>
                    </Protocols>
                    <Port>-1</Port>
                    <DestinationPortRange>Any</DestinationPortRange>
                    <DestinationIp>Any</DestinationIp>
                    <SourcePort>-1</SourcePort>
                    <SourcePortRange>Any</SourcePortRange>
                    <SourceIp>192.168.109.8</SourceIp>
                    <EnableLogging>false</EnableLogging>
                </FirewallRule>
                <FirewallRule>
                    <Id>3</Id>
                    <IsEnabled>true</IsEnabled>
                    <MatchOnTranslate>false</MatchOnTranslate>
                    <Description>suppah megah rulez creati0nz</Description>
                    <Policy>allow</Policy>
                    <Protocols>
                        <Any>true</Any>
                    </Protocols>
                    <Port>-1</Port>
                    <DestinationPortRange>Any</DestinationPortRange>
                    <DestinationIp>23.92.224.255</DestinationIp>
                    <SourcePort>-1</SourcePort>
                    <SourcePortRange>Any</SourcePortRange>
                    <SourceIp>Any</SourceIp>
                    <EnableLogging>false</EnableLogging>
                </FirewallRule>
                <FirewallRule>
                    <Id>4</Id>
                    <IsEnabled>true</IsEnabled>
                    <MatchOnTranslate>false</MatchOnTranslate>
                    <Description>suppah megah rulez creati0nz</Description>
                    <Policy>allow</Policy>
                    <Protocols>
                        <Any>true</Any>
                    </Protocols>
                    <Port>-1</Port>
                    <DestinationPortRange>Any</DestinationPortRange>
                    <DestinationIp>Any</DestinationIp>
                    <SourcePort>-1</SourcePort>
                    <SourcePortRange>Any</SourcePortRange>
                    <SourceIp>192.168.109.3</SourceIp>
                    <EnableLogging>false</EnableLogging>
                </FirewallRule>
            </FirewallService>
            <NatService>
                <IsEnabled>false</IsEnabled>
                <NatRule>
                    <RuleType>DNAT</RuleType>
                    <IsEnabled>true</IsEnabled>
                    <Id>65537</Id>
                    <GatewayNatRule>
                        <Interface href="http://localhost:4444/api/admin/network/6254f107-9876-4d03-986f-8bec7a4bcb3f" name="d2p3-ext" type="application/vnd.vmware.admin.network+xml"/>
                        <OriginalIp>23.92.225.51</OriginalIp>
                        <OriginalPort>999</OriginalPort>
                        <TranslatedIp>192.168.109.8</TranslatedIp>
                        <TranslatedPort>22</TranslatedPort>
                        <Protocol>Tcp</Protocol>
                    </GatewayNatRule>
                </NatRule>
                <NatRule>
                    <RuleType>SNAT</RuleType>
                    <IsEnabled>true</IsEnabled>
                    <Id>65538</Id>
                    <GatewayNatRule>
                        <Interface href="http://localhost:4444/api/admin/network/6254f107-9876-4d03-986f-8bec7a4bcb3f" name="d2p3-ext" type="application/vnd.vmware.admin.network+xml"/>
                        <OriginalIp>192.168.109.8</OriginalIp>
                        <TranslatedIp>23.92.225.51</TranslatedIp>
                    </GatewayNatRule>
                </NatRule>
                <NatRule>
                    <RuleType>SNAT</RuleType>
                    <IsEnabled>true</IsEnabled>
                    <Id>65539</Id>
                    <GatewayNatRule>
                        <Interface href="http://localhost:4444/api/admin/network/6254f107-9876-4d03-986f-8bec7a4bcb3f" name="d2p3-ext" type="application/vnd.vmware.admin.network+xml"/>
                        <OriginalIp>192.168.109.3</OriginalIp>
                        <TranslatedIp>23.92.224.255</TranslatedIp>
                    </GatewayNatRule>
                </NatRule>
                <NatRule>
                    <RuleType>DNAT</RuleType>
                    <IsEnabled>true</IsEnabled>
                    <Id>65540</Id>
                    <GatewayNatRule>
                        <Interface href="http://localhost:4444/api/admin/network/6254f107-9876-4d03-986f-8bec7a4bcb3f" name="d2p3-ext" type="application/vnd.vmware.admin.network+xml"/>
                        <OriginalIp>23.92.224.255</OriginalIp>
                        <OriginalPort>any</OriginalPort>
                        <TranslatedIp>192.168.109.3</TranslatedIp>
                        <TranslatedPort>any</TranslatedPort>
                        <Protocol>any</Protocol>
                    </GatewayNatRule>
                </NatRule>
            </NatService>
            <GatewayIpsecVpnService>
                <IsEnabled>true</IsEnabled>
                <Tunnel>
                    <Name>Test VPN Alpha</Name>
                    <Description>For OC Pod Testing</Description>
                    <IpsecVpnThirdPartyPeer>
                        <PeerId>192.168.110.99</PeerId>
                    </IpsecVpnThirdPartyPeer>
                    <PeerIpAddress>64.184.133.62</PeerIpAddress>
                    <PeerId>192.168.110.99</PeerId>
                    <LocalIpAddress>23.92.225.51</LocalIpAddress>
                    <LocalId>23.92.225.51</LocalId>
                    <LocalSubnet>
                        <Name>M916272752-5793-default-routed</Name>
                        <Gateway>192.168.109.1</Gateway>
                        <Netmask>255.255.255.0</Netmask>
                    </LocalSubnet>
                    <PeerSubnet>
                        <Name>192.168.120.0/24</Name>
                        <Gateway>192.168.120.0</Gateway>
                        <Netmask>255.255.255.0</Netmask>
                    </PeerSubnet>
                    <SharedSecret>Blah1Blah2Blah3Blah1Blah2Blah3Blah1Blah2Blah3</SharedSecret>
                    <SharedSecretEncrypted>false</SharedSecretEncrypted>
                    <EncryptionProtocol>AES256</EncryptionProtocol>
                    <Mtu>1500</Mtu>
                    <IsEnabled>true</IsEnabled>
                    <IsOperational>false</IsOperational>
                </Tunnel>
                <Tunnel>
                    <Name>JGray VPN</Name>
                    <IpsecVpnThirdPartyPeer>
                        <PeerId>67.172.148.74</PeerId>
                    </IpsecVpnThirdPartyPeer>
                    <PeerIpAddress>67.172.148.74</PeerIpAddress>
                    <PeerId>67.172.148.74</PeerId>
                    <LocalIpAddress>23.92.225.51</LocalIpAddress>
                    <LocalId>23.92.225.51</LocalId>
                    <LocalSubnet>
                        <Name>JGray Network</Name>
                        <Gateway>192.168.108.1</Gateway>
                        <Netmask>255.255.255.0</Netmask>
                    </LocalSubnet>
                    <PeerSubnet>
                        <Name>192.168.1.0/24</Name>
                        <Gateway>192.168.1.0</Gateway>
                        <Netmask>255.255.255.0</Netmask>
                    </PeerSubnet>
                    <SharedSecret>fM7puHkGbg6tqpd4jKNhBo45mbs8fw3yapgt3H7f97a2jrkLyYjP9eSH4oGwps7u</SharedSecret>
                    <SharedSecretEncrypted>false</SharedSecretEncrypted>
                    <EncryptionProtocol>AES256</EncryptionProtocol>
                    <Mtu>1500</Mtu>
                    <IsEnabled>false</IsEnabled>
                    <IsOperational>false</IsOperational>
                </Tunnel>
            </GatewayIpsecVpnService>
            <StaticRoutingService>
                <IsEnabled>false</IsEnabled>
            </StaticRoutingService>
            <LoadBalancerService>
                <IsEnabled>false</IsEnabled>
            </LoadBalancerService>
        </EdgeGatewayServiceConfiguration>
        <HaEnabled>true</HaEnabled>
        <UseDefaultRouteForDnsRelay>true</UseDefaultRouteForDnsRelay>
    </Configuration>
</EdgeGateway>
	`

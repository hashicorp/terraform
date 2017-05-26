# 1&amp;1  Cloudserver Go SDK

The 1&amp;1  Go SDK is a Go library designed for interaction with the 1&amp;1  cloud platform over the REST API.

This guide contains instructions on getting started with the library and automating various management tasks available through the 1&amp;1  Cloud Panel UI.

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Authentication](#authentication)
- [Operations](#operations)
  - [Servers](#servers)
  - [Images](#images)
  - [Shared Storages](#shared-storages)
  - [Firewall Policies](#firewall-policies)
  - [Load Balancers](#load-balancers)
  - [Public IPs](#public-ips)
  - [Private Networks](#private-networks)
  - [VPNs](#vpns)
  - [Monitoring Center](#monitoring-center)
  - [Monitoring Policies](#monitoring-policies)
  - [Logs](#logs)
  - [Users](#users)
  - [Roles](#roles)
  - [Usages](#usages)
  - [Server Appliances](#server-appliances)
  - [DVD ISO](#dvd-iso)
  - [Ping](#ping)
  - [Pricing](#pricing)
  - [Data Centers](#data-centers)
- [Examples](#examples)
- [Index](#index)

## Overview

This SDK is a wrapper for the 1&amp;1  REST API written in Go(lang). All operations against the API are performed over SSL and authenticated using your 1&amp;1  token key. The Go library facilitates the access to the REST API either within an instance running on 1&amp;1  platform or directly across the Internet from any HTTPS-enabled application.

For more information on the 1&1 Cloud Server SDK for Go, visit the [Community Portal](https://www.1and1.com/cloud-community/).

## Getting Started

Before you begin you will need to have signed up for a 1&amp;1  account. The credentials you create during sign-up will be used to authenticate against the API.

Install the Go language tools. Find the install package and instructions on the official <a href='https://golang.org/doc/install'>Go website</a>. Make sure that you have set up the `GOPATH` environment variable properly, as indicated in the instructions.

### Installation

The official Go library is available from the 1&amp;1  GitHub account found <a href='https://github.com/1and1/oneandone-cloudserver-sdk-go'>here</a>.

Use the following Go command to download oneandone-cloudserver-sdk-go to your configured GOPATH:

`go get github.com/1and1/oneandone-cloudserver-sdk-go`

Import the library in your Go code:

`import "github.com/1and1/oneandone-cloudserver-sdk-go"`

### Authentication

Set the authentication token and create the API client:

```
token := oneandone.SetToken("82ee732b8d47e451be5c6ad5b7b56c81")
api := oneandone.New(token, oneandone.BaseUrl)
```

Refer to the [Examples](#examples) and [Operations](#operations) sections for additional information.

## Operations

### Servers

**List all servers:**

`servers, err := api.ListServers()`

Alternatively, use the method with query parameters.

`servers, err := api.ListServers(page, per_page, sort, query, fields)`

To paginate the list of servers received in the response use `page` and `per_page` parameters. Set `per_page` to the number of servers that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of servers sorted in expected order pass a server property (e.g. `"name"`) in `sort` parameter.

Use `query` parameter to search for a string in the response and return only the server instances that contain it.

To retrieve a collection of servers containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,description,hardware.ram"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single server:**

`server, err := api.GetServer(server_id)`

**List fixed-size server templates:**

`fiss, err := api.ListFixedInstanceSizes()`

**Retrieve information about a fixed-size server template:**

`fis, err := api.GetFixedInstanceSize(fis_id)`

**Retrieve information about a server's hardware:**

`hardware, err := api.GetServerHardware(server_id)`

**List a server's HDDs:**

`hdds, err := api.ListServerHdds(server_id)`

**Retrieve a single server HDD:**

`hdd, err := api.GetServerHdd(server_id, hdd_id)`

**Retrieve information about a server's image:**

`image, err := api.GetServerImage(server_id)`

**List a server's IPs:**

`ips, err := api.ListServerIps(server_id)`

**Retrieve information about a single server IP:**

`ip, err := api.GetServerIp(server_id, ip_id)`

**Retrieve information about a server's firewall policy:**

`firewall, err := api.GetServerIpFirewallPolicy(server_id, ip_id)`

**List all load balancers assigned to a server IP:**

`lbs, err := api.ListServerIpLoadBalancers(server_id, ip_id)`

**Retrieve information about a server's status:**

`status, err := api.GetServerStatus(server_id)`

**Retrieve information about the DVD loaded into the virtual DVD unit of a server:**

`dvd, err := api.GetServerDvd(server_id)`

**List a server's private networks:**

`pns, err := api.ListServerPrivateNetworks(server_id)`

**Retrieve information about a server's private network:**

`pn, err := api.GetServerPrivateNetwork(server_id, pn_id)`

**Retrieve information about a server's snapshot:**

`snapshot, err := api.GetServerSnapshot(server_id)`

**Create a server:**

```
req := oneandone.ServerRequest {
    Name:        "Server Name",
    Description: "Server description.",
    ApplianceId: server_appliance_id,
    PowerOn:     true,
    Hardware:    oneandone.Hardware {
      Vcores:            1,
      CoresPerProcessor: 1,
      Ram:               2,
      Hdds: []oneandone.Hdd {
        oneandone.Hdd {
            Size:   100,
            IsMain: true,
        },
      },
    },
  }

server_id, server, err := api.CreateServer(&req)
```

**Create a fixed-size server and return back the server's IP address and first password:**

```
req := oneandone.ServerRequest {
    Name:        server_name,
    ApplianceId: server_appliance_id,
    PowerOn:     true_or_false,
    Hardware:    oneandone.Hardware {
          FixedInsSizeId: fixed_instance_size_id,
      },
  }

ip_address, password, err := api.CreateServerEx(&req, timeout)
```

**Update a server:**

`server, err := api.RenameServer(server_id, new_name, new_desc)`

**Delete a server:**

`server, err := api.DeleteServer(server_id, keep_ips)`

Set `keep_ips` parameter to `true` for keeping server IPs after deleting a server.

**Update a server's hardware:**

```
hardware := oneandone.Hardware {
		Vcores: 2,
		CoresPerProcessor: 1,
		Ram: 2,
	}

server, err := api.UpdateServerHardware(server_id, &hardware)
```

**Add new hard disk(s) to a server:**

```
hdds := oneandone.ServerHdds {
    Hdds: []oneandone.Hdd {
        {
          Size: 50,
          IsMain: false,
      },
    },
  }

server, err := api.AddServerHdds(server_id, &hdds)
```

**Resize a server's hard disk:**

`server, err := api.ResizeServerHdd(server_id, hdd_id, new_size)`

**Remove a server's hard disk:**

`server, err := api.DeleteServerHdd(server_id, hdd_id)`

**Load a DVD into the virtual DVD unit of a server:**

`server, err := api.LoadServerDvd(server_id, dvd_id)`

**Unload a DVD from the virtual DVD unit of a server:**

`server, err := api.EjectServerDvd(server_id)`

**Reinstall a new image into a server:**

`server, err := api.ReinstallServerImage(server_id, image_id, password, fp_id)`

**Assign a new IP to a server:**

`server, err := api.AssignServerIp(server_id, ip_type)`

**Release an IP and optionally remove it from a server:**

`server, err := api.DeleteServerIp(server_id, ip_id, keep_ip)`

Set `keep_ip` to true for releasing the IP without removing it.

**Assign a new firewall policy to a server's IP:**

`server, err := api.AssignServerIpFirewallPolicy(server_id, ip_id, fp_id)`

**Remove a firewall policy from a server's IP:**

`server, err := api.UnassignServerIpFirewallPolicy(server_id, ip_id)`

**Assign a new load balancer to a server's IP:**

`server, err := api.AssignServerIpLoadBalancer(server_id, ip_id, lb_id)`

**Remove a load balancer from a server's IP:**

`server, err := api.UnassignServerIpLoadBalancer(server_id, ip_id, lb_id)`

**Start a server:**

`server, err := api.StartServer(server_id)`

**Reboot a server:**

`server, err := api.RebootServer(server_id, is_hardware)`

Set `is_hardware` to true for HARDWARE method of rebooting.

Set `is_hardware` to false for SOFTWARE method of rebooting.

**Shutdown a server:**

`server, err := api.ShutdownServer(server_id, is_hardware)`

Set `is_hardware` to true for HARDWARE method of powering off.

Set `is_hardware` to false for SOFTWARE method of powering off.

**Assign a private network to a server:**

`server, err := api.AssignServerPrivateNetwork(server_id, pn_id)`

**Remove a server's private network:**

`server, err := api.RemoveServerPrivateNetwork(server_id, pn_id)`

**Create a new server's snapshot:**

`server, err := api.CreateServerSnapshot(server_id)`

**Restore a server's snapshot:**

`server, err := api.RestoreServerSnapshot(server_id, snapshot_id)`

**Remove a server's snapshot:**

`server, err := api.DeleteServerSnapshot(server_id, snapshot_id);`

**Clone a server:**

`server, err := api.CloneServer(server_id, new_name)`


### Images

**List all images:**

`images, err = api.ListImages()`

Alternatively, use the method with query parameters.

`images, err = api.ListImages(page, per_page, sort, query, fields)`

To paginate the list of images received in the response use `page` and `per_page` parameters. set `per_page` to the number of images that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of images sorted in expected order pass an image property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the elements that contain it.

To retrieve a collection of images containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single image:**

`image, err = api.GetImage(image_id)`


**Create an image:**

```
request := oneandone.ImageConfig {
    Name: image_name,
    Description: image_description,
    ServerId: server_id, 
    Frequency: image_frequenct,
    NumImages: number_of_images,
  }

image_id, image, err = api.CreateImage(&request)
```
All fields except `Description` are required. `Frequency` may be set to `"ONCE"`, `"DAILY"` or `"WEEKLY"`.

**Update an image:**


`image, err = api.UpdateImage(image_id, new_name, new_description, new_frequenct)`

If any of the parameters `new_name`, `new_description` or `new_frequenct` is set to an empty string, it is ignored in the request. `Frequency` may be set to `"ONCE"`, `"DAILY"` or `"WEEKLY"`.

**Delete an image:**

`image, err = api.DeleteImage(image_id)`

### Shared Storages

`ss, err := api.ListSharedStorages()`

Alternatively, use the method with query parameters.

`ss, err := api.ListSharedStorages(page, per_page, sort, query, fields)`

To paginate the list of shared storages received in the response use `page` and `per_page` parameters. Set `per_page` to the number of volumes that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of shared storages sorted in expected order pass a volume property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the volume instances that contain it.

To retrieve a collection of shared storages containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,size,size_used"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a shared storage:**

`ss, err := api.GetSharedStorage(ss_id)`


**Create a shared storage:**

```
request := oneandone.SharedStorageRequest {
    Name: test_ss_name, 
    Description: test_ss_desc,
    Size: oneandone.Int2Pointer(size),
  }
  
ss_id, ss, err := api.CreateSharedStorage(&request)

```
`Description` is optional parameter.


**Update a shared storage:**

```
request := oneandone.SharedStorageRequest {
    Name: new_name, 
    Description: new_desc,
    Size: oneandone.Int2Pointer(new_size),
  }
  
ss, err := api.UpdateSharedStorage(ss_id, &request)
```
All request's parameters are optional.


**Remove a shared storage:**

`ss, err := api.DeleteSharedStorage(ss_id)`


**List a shared storage servers:**

`ss_servers, err := api.ListSharedStorageServers(ss_id)`


**Retrieve a shared storage server:**

`ss_server, err := api.GetSharedStorageServer(ss_id, server_id)`


**Add servers to a shared storage:**

```
servers := []oneandone.SharedStorageServer {
    {
      Id: server_id,
      Rights: permissions,
    } ,
  }
  
ss, err := api.AddSharedStorageServers(ss_id, servers)
```
`Rights` may be set to `R` or `RW` string.

				
**Remove a server from a shared storage:**

`ss, err := api.DeleteSharedStorageServer(ss_id, server_id)`


**Retrieve the credentials for accessing the shared storages:**

`ss_credentials, err := api.GetSharedStorageCredentials()`


**Change the password for accessing the shared storages:**

`ss_credentials, err := api.UpdateSharedStorageCredentials(new_password)`


### Firewall Policies

**List firewall policies:**

`firewalls, err := api.ListFirewallPolicies()`

Alternatively, use the method with query parameters.

`firewalls, err := api.ListFirewallPolicies(page, per_page, sort, query, fields)`

To paginate the list of firewall policies received in the response use `page` and `per_page` parameters. Set `per_page` to the number of firewall policies that will be shown in each page.  `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of firewall policies sorted in expected order pass a firewall policy property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the firewall policy instances that contain it.

To retrieve a collection of firewall policies containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single firewall policy:**

`firewall, err := api.GetFirewallPolicy(fp_id)`


**Create a firewall policy:**

```
request := oneandone.FirewallPolicyRequest {
    Name: fp_name, 
    Description: fp_desc,
    Rules: []oneandone.FirewallPolicyRule {
      {
        Protocol: protocol,
        PortFrom: oneandone.Int2Pointer(port_from),
        PortTo: oneandone.Int2Pointer(port_to),
        SourceIp: source_ip,
      },
    },
  }
  
firewall_id, firewall, err := api.CreateFirewallPolicy(&request)
```
`SourceIp` and `Description` are optional parameters.

			
**Update a firewall policy:**

`firewall, err := api.UpdateFirewallPolicy(fp_id, fp_new_name, fp_new_description)`

Passing an empty string in `fp_new_name` or `fp_new_description` skips updating the firewall policy name or description respectively.

			
**Delete a firewall policy:**

`firewall, err := api.DeleteFirewallPolicy(fp_id)`


**List servers/IPs attached to a firewall policy:**

`server_ips, err := api.ListFirewallPolicyServerIps(fp_id)`


**Retrieve information about a server/IP assigned to a firewall policy:**

`server_ip, err := api.GetFirewallPolicyServerIp(fp_id, ip_id)`


**Add servers/IPs to a firewall policy:**

`firewall, err := api.AddFirewallPolicyServerIps(fp_id, ip_ids)`

`ip_ids` is a slice of IP ID's.


**Remove a server/IP from a firewall policy:**

`firewall, err := api.DeleteFirewallPolicyServerIp(fp_id, ip_id)`


**List rules of a firewall policy:**

`fp_rules, err := api.ListFirewallPolicyRules(fp_id)`


**Retrieve information about a rule of a firewall policy:**

`fp_rule, err := api.GetFirewallPolicyRule(fp_id, rule_id)`


**Adds new rules to a firewall policy:**

```
fp_rules := []oneandone.FirewallPolicyRule {
    {
      Protocol: protocol1,
      PortFrom: oneandone.Int2Pointer(port_from1),
      PortTo: oneandone.Int2Pointer(port_to1),
      SourceIp: source_ip,
    },
    {
      Protocol: protocol2,
      PortFrom: oneandone.Int2Pointer(port_from2),
      PortTo: oneandone.Int2Pointer(port_to2),
    },
  }

firewall, err := api.AddFirewallPolicyRules(fp_id, fp_rules)
```

**Remove a rule from a firewall policy:**

`firewall, err := api.DeleteFirewallPolicyRule(fp_id, rule_id)`


### Load Balancers

**List load balancers:**

`loadbalancers, err := api.ListLoadBalancers()`

Alternatively, use the method with query parameters.

`loadbalancers, err := api.ListLoadBalancers(page, per_page, sort, query, fields)`

To paginate the list of load balancers received in the response use `page` and `per_page` parameters. Set `per_page` to the number of load balancers that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of load balancers sorted in expected order pass a load balancer property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the load balancer instances that contain it.

To retrieve a collection of load balancers containing only the requested fields pass a list of comma separated properties (e.g. `"ip,name,method"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single load balancer:**

`loadbalancer, err := api.GetLoadBalancer(lb_id)`


**Create a load balancer:**

```
request := oneandone.LoadBalancerRequest {
    Name: lb_name, 
    Description: lb_description,
    Method: lb_method,
    Persistence: oneandone.Bool2Pointer(true_or_false),
    PersistenceTime: oneandone.Int2Pointer(seconds1),
    HealthCheckTest: protocol1,
    HealthCheckInterval: oneandone.Int2Pointer(seconds2),
    HealthCheckPath: health_check_path,
    HealthCheckPathParser: health_check_path_parser,
    Rules: []oneandone.LoadBalancerRule {
        {
          Protocol: protocol1,
          PortBalancer: lb_port,
          PortServer: server_port,
          Source: source_ip,
        },
    },
  }
  
loadbalancer_id, loadbalancer, err := api.CreateLoadBalancer(&request)
```
Optional parameters are `HealthCheckPath`, `HealthCheckPathParser`, `Source` and `Description`. Load balancer `Method` must be set to `"ROUND_ROBIN"` or `"LEAST_CONNECTIONS"`.

**Update a load balancer:**
```
request := oneandone.LoadBalancerRequest {
    Name: new_name,
    Description: new_description,
    Persistence: oneandone.Bool2Pointer(true_or_false),
    PersistenceTime: oneandone.Int2Pointer(new_seconds1),
    HealthCheckTest: new_protocol,
    HealthCheckInterval: oneandone.Int2Pointer(new_seconds2),
    HealthCheckPath: new_path,
    HealthCheckPathParser: new_parser,
    Method: new_lb_method,
  }
  
loadbalancer, err := api.UpdateLoadBalancer(lb_id, &request)
```
All updatable fields are optional.


**Delete a load balancer:**

`loadbalancer, err := api.DeleteLoadBalancer(lb_id)`


**List servers/IPs attached to a load balancer:**

`server_ips, err := api.ListLoadBalancerServerIps(lb_id)`


**Retrieve information about a server/IP assigned to a load balancer:**

`server_ip, err := api.GetLoadBalancerServerIp(lb_id, ip_id)`


**Add servers/IPs to a load balancer:**

`loadbalancer, err := api.AddLoadBalancerServerIps(lb_id, ip_ids)`

`ip_ids` is a slice of IP ID's.


**Remove a server/IP from a load balancer:**

`loadbalancer, err := api.DeleteLoadBalancerServerIp(lb_id, ip_id)`


**List rules of a load balancer:**

`lb_rules, err := api.ListLoadBalancerRules(lb_id)`


**Retrieve information about a rule of a load balancer:**

`lb_rule, err := api.GetLoadBalancerRule(lb_id, rule_id)`


**Adds new rules to a load balancer:**

```
lb_rules := []oneandone.LoadBalancerRule {
    {
      Protocol: protocol1,
      PortBalancer: lb_port1,
      PortServer: server_port1,
      Source: source_ip,
    },
    {
      Protocol: protocol2,
      PortBalancer: lb_port2,
      PortServer: server_port2,
    },
  }

loadbalancer, err := api.AddLoadBalancerRules(lb_id, lb_rules)
```

**Remove a rule from a load balancer:**

`loadbalancer, err := api.DeleteLoadBalancerRule(lb_id, rule_id)`


### Public IPs

**Retrieve a list of your public IPs:**

`public_ips, err := api.ListPublicIps()`

Alternatively, use the method with query parameters.

`public_ips, err := api.ListPublicIps(page, per_page, sort, query, fields)`

To paginate the list of public IPs received in the response use `page` and `per_page` parameters. Set `per_page` to the number of public IPs that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of public IPs sorted in expected order pass a public IP property (e.g. `"ip"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the public IP instances that contain it.

To retrieve a collection of public IPs containing only the requested fields pass a list of comma separated properties (e.g. `"id,ip,reverse_dns"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.


**Retrieve a single public IP:**

`public_ip, err := api.GetPublicIp(ip_id)`


**Create a public IP:**

`ip_id, public_ip, err := api.CreatePublicIp(ip_type, reverse_dns)`

Both parameters are optional and may be left blank. `ip_type` may be set to `"IPV4"` or `"IPV6"`. Presently, only IPV4 is supported.

**Update the reverse DNS of a public IP:**

`public_ip, err := api.UpdatePublicIp(ip_id, reverse_dns)`

If an empty string is passed in `reverse_dns,` it removes previous reverse dns of the public IP.

**Remove a public IP:**

`public_ip, err := api.DeletePublicIp(ip_id)`


### Private Networks

**List all private networks:**

`private_nets, err := api.ListPrivateNetworks()`

Alternatively, use the method with query parameters.

`private_nets, err := api.ListPrivateNetworks(page, per_page, sort, query, fields)`

To paginate the list of private networks received in the response use `page` and `per_page` parameters. Set `per_page` to the number of private networks that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of private networks sorted in expected order pass a private network property (e.g. `"-creation_date"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the private network instances that contain it.

To retrieve a collection of private networks containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is blank, it is ignored in the request.

**Retrieve information about a private network:**

`private_net, err := api.GetPrivateNetwork(pn_id)`

**Create a new private network:**

```
request := oneandone.PrivateNetworkRequest {
    Name: pn_name, 
    Description: pn_description,
    NetworkAddress: network_address,
    SubnetMask: subnet_mask,
  }

pnet_id, private_net, err := api.CreatePrivateNetwork(&request)
```
Private network `Name` is required parameter.


**Modify a private network:**

```
request := oneandone.PrivateNetworkRequest {
    Name: new_pn_name, 
    Description: new_pn_description,
    NetworkAddress: new_network_address,
    SubnetMask: new_subnet_mask,
  }

private_net, err := api.UpdatePrivateNetwork(pn_id, &request)
```
All parameters in the request are optional.


**Delete a private network:**

`private_net, err := api.DeletePrivateNetwork(pn_id)`


**List all servers attached to a private network:**

`servers, err = := api.ListPrivateNetworkServers(pn_id)`


**Retrieve a server attached to a private network:**

`server, err = := api.GetPrivateNetworkServer(pn_id, server_id)`


**Attach servers to a private network:**

`private_net, err := api.AttachPrivateNetworkServers(pn_id, server_ids)`

`server_ids` is a slice of server ID's.

*Note:* Servers cannot be attached to a private network if they currently have a snapshot.


**Remove a server from a private network:**

`private_net, err := api.DetachPrivateNetworkServer(pn_id, server_id)`

*Note:* The server cannot be removed from a private network if it currently has a snapshot or it is powered on.


### VPNs

**List all VPNs:**

`vpns, err := api.ListVPNs()`

Alternatively, use the method with query parameters.

`vpns, err := api.ListVPNs(page, per_page, sort, query, fields)`

To paginate the list of VPNs received in the response use `page` and `per_page` parameters. Set ` per_page` to the number of VPNs that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of VPNs sorted in expected order pass a VPN property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the VPN instances that contain it.

To retrieve a collection of VPNs containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve information about a VPN:**

`vpn, err := api.GetVPN(vpn_id)`

**Create a VPN:**

`vpn, err := api.CreateVPN(vpn_name, vpn_description, datacenter_id)`

**Modify a VPN:**

`vpn, err := api.ModifyVPN(vpn_id, new_name, new_description)`

**Delete a VPN:**

`vpn, err := api.DeleteVPN(vpn_id)`

**Retrieve a VPN's configuration file:**

`base64_encoded_string, err := api.GetVPNConfigFile(vpn_id)`


### Monitoring Center

**List all usages and alerts of monitoring servers:**

`server_usages, err := api.ListMonitoringServersUsages()`

Alternatively, use the method with query parameters.

`server_usages, err := api.ListMonitoringServersUsages(page, per_page, sort, query, fields)`

To paginate the list of server usages received in the response use `page` and `per_page` parameters. Set `per_page` to the number of server usages that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of server usages sorted in expected order pass a server usage property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the usage instances that contain it.

To retrieve a collection of server usages containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,status.state"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is blank, it is ignored in the request.

**Retrieve the usages and alerts for a monitoring server:**

`server_usage, err := api.GetMonitoringServerUsage(server_id, period)`

`period` may be set to `"LAST_HOUR"`, `"LAST_24H"`, `"LAST_7D"`, `"LAST_30D"`, `"LAST_365D"` or `"CUSTOM"`. If `period` is set to `"CUSTOM"`, the `start_date` and `end_date` parameters are required to be set in **RFC 3339** date/time format (e.g. `2015-13-12T00:01:00Z`).

`server_usage, err := api.GetMonitoringServerUsage(server_id, period, start_date, end_date)`

### Monitoring Policies

**List all monitoring policies:**

`mon_policies, err := api.ListMonitoringPolicies()`

Alternatively, use the method with query parameters.

`mon_policies, err := api.ListMonitoringPolicies(page, per_page, sort, query, fields)`

To paginate the list of monitoring policies received in the response use `page` and `per_page` parameters. Set `per_page` to the number of monitoring policies that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of monitoring policies sorted in expected order pass a monitoring policy property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the monitoring policy instances that contain it.

To retrieve a collection of monitoring policies containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single monitoring policy:**

`mon_policy, err := api.GetMonitoringPolicy(mp_id)`


**Create a monitoring policy:**

```
request := oneandone.MonitoringPolicy {
    Name:  mp_name,
    Description: mp_desc,
    Email: mp_mail,
    Agent: true_or_false,
    Thresholds: &oneandone.MonitoringThreshold {
      Cpu: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
      },
      Ram: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
      },
      Disk: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
      },
      Transfer: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue  {
          Value: threshold_value,
          Alert: true_or_false,
        },
      },
      InternalPing: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: threshold_value,
          Alert: true_or_false,
        },
      },
    },
    Ports: []oneandone.MonitoringPort {
      {
        Protocol: protocol,
        Port: port,
        AlertIf: responding_or_not_responding,
        EmailNotification: true_or_false,
      },
    },
    Processes: []oneandone.MonitoringProcess {
      {
        Process: process_name,
        AlertIf: running_or_not_running,
        EmailNotification: true_or_false,
      },
    },
  }
  
mpolicy_id, mon_policy, err := api.CreateMonitoringPolicy(&request)
```
All fields, except `Description`, are required. `AlertIf` property accepts values `"RESPONDING"`/`"NOT_RESPONDING"` for ports, and `"RUNNING"`/`"NOT_RUNNING"` for processes.


**Update a monitoring policy:**

```
request := oneandone.MonitoringPolicy {
    Name:  new_mp_name,
    Description: new_mp_desc,
    Email: new_mp_mail,
    Thresholds: &oneandone.MonitoringThreshold {
      Cpu: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
      },
      Ram: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
      },
      Disk: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
      },
      Transfer: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue  {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
      },
      InternalPing: &oneandone.MonitoringLevel {
        Warning: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
        Critical: &oneandone.MonitoringValue {
          Value: new_threshold_value,
          Alert: true_or_false,
        },
      },
    },
  }
  
mon_policy, err := api.UpdateMonitoringPolicy(mp_id, &request)
```
All fields of the request are optional. When a threshold is specified in the request, the threshold fields are required.

**Delete a monitoring policy:**

`mon_policy, err := api.DeleteMonitoringPolicy(mp_id)`


**List all ports of a monitoring policy:**

`mp_ports, err := api.ListMonitoringPolicyPorts(mp_id)`


**Retrieve information about a port of a monitoring policy:**

`mp_port, err := api.GetMonitoringPolicyPort(mp_id, port_id)`


**Add new ports to a monitoring policy:**

```
mp_ports := []oneandone.MonitoringPort {
    {
      Protocol: protocol1,
      Port: port1,
      AlertIf: responding_or_not_responding,
      EmailNotification: true_or_false,
    },
    {
      Protocol: protocol2,
      Port: port2,
      AlertIf: responding_or_not_responding,
      EmailNotification: true_or_false,
    },
  }

mon_policy, err := api.AddMonitoringPolicyPorts(mp_id, mp_ports)
```
Port properties are mandatory.


**Modify a port of a monitoring policy:**

```
mp_port := oneandone.MonitoringPort {
    Protocol: protocol,
    Port: port,
    AlertIf: responding_or_not_responding,
    EmailNotification: true_or_false,
  }
  
mon_policy, err := api.ModifyMonitoringPolicyPort(mp_id, port_id, &mp_port)
```
*Note:* `Protocol` and `Port` cannot be changed.


**Remove a port from a monitoring policy:**

`mon_policy, err := api.DeleteMonitoringPolicyPort(mp_id, port_id)`


**List the processes of a monitoring policy:**

`mp_processes, err := api.ListMonitoringPolicyProcesses(mp_id)`


**Retrieve information about a process of a monitoring policy:**

`mp_process, err := api.GetMonitoringPolicyProcess(mp_id, process_id)`


**Add new processes to a monitoring policy:**

```
processes := []oneandone.MonitoringProcess {
    {
      Process: process_name1,
      AlertIf: running_or_not_running,
      EmailNotification: true_or_false,
    },
    {
      Process: process_name2,
      AlertIf: running_or_not_running,
      EmailNotification: true_or_false,
    },
  }

mon_policy, err := api.AddMonitoringPolicyProcesses(mp_id, processes)
```
All properties of the `MonitoringProcess` instance are required.


**Modify a process of a monitoring policy:**

```
process := oneandone.MonitoringProcess {
    Process: process_name,
    AlertIf: running_or_not_running,
    EmailNotification: true_or_false,
  }

mon_policy, err := api.ModifyMonitoringPolicyProcess(mp_id, process_id, &process)
```

*Note:* Process name cannot be changed.

**Remove a process from a monitoring policy:**

`mon_policy, err := api.DeleteMonitoringPolicyProcess(mp_id, process_id)`

**List all servers attached to a monitoring policy:**

`mp_servers, err := api.ListMonitoringPolicyServers(mp_id)`

**Retrieve information about a server attached to a monitoring policy:**

`mp_server, err := api.GetMonitoringPolicyServer(mp_id, server_id)`

**Attach servers to a monitoring policy:**

`mon_policy, err := api.AttachMonitoringPolicyServers(mp_id, server_ids)`

`server_ids` is a slice of server ID's.

**Remove a server from a monitoring policy:**

`mon_policy, err := api.RemoveMonitoringPolicyServer(mp_id, server_id)`


### Logs

**List all logs:**

`logs, err := api.ListLogs(period, nil, nil)`

`period` can be set to `"LAST_HOUR"`, `"LAST_24H"`, `"LAST_7D"`, `"LAST_30D"`, `"LAST_365D"` or `"CUSTOM"`. If `period` is set to `"CUSTOM"`, the `start_date` and `end_date` parameters are required to be set in **RFC 3339** date/time format (e.g. `2015-13-12T00:01:00Z`).

`logs, err := api.ListLogs(period, start_date, end_date)`

Additional query parameters can be used.

`logs, err := api.ListLogs(period, start_date, end_date, page, per_page, sort, query, fields)`

To paginate the list of logs received in the response use `page` and `per_page` parameters. Set ` per_page` to the number of logs that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of logs sorted in expected order pass a logs property (e.g. `"action"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the logs instances that contain it.

To retrieve a collection of logs containing only the requested fields pass a list of comma separated properties (e.g. `"id,action,type"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve a single log:**

`log, err := api.GetLog(log_id)`


### Users

**List all users:**

`users, err := api.ListUsers()`

Alternatively, use the method with query parameters.

`users, err := api.ListUsers(page, per_page, sort, query, fields)`

To paginate the list of users received in the response use `page` and `per_page` parameters. Set ` per_page` to the number of users that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of users sorted in expected order pass a user property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the user instances that contain it.

To retrieve a collection of users containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date,email"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve information about a user:**

`user, err := api.GetUser(user_id)`

**Create a user:**

```
request := oneandone.UserRequest {
    Name: username, 
    Description: user_description,
    Password: password,
    Email: user_email,
  }

user_id, user, err := api.CreateUser(&request)
```

`Name` and `Password` are required parameters. The password must contain at least 8 characters using uppercase letters, numbers and other special symbols.

**Modify a user:**

```
request := oneandone.UserRequest {
    Description: new_desc,
    Email: new_mail,
    Password: new_pass,
    State: state,
  }

user, err := api.ModifyUser(user_id, &request)
```

All listed fields in the request are optional. `State` can be set to `"ACTIVE"` or `"DISABLED"`.

**Delete a user:**

`user, err := api.DeleteUser(user_id)`

**Retrieve information about a user's API privileges:**

`api_info, err := api.GetUserApi(user_id)`

**Retrieve a user's API key:**

`api_key, err := api.GetUserApiKey(user_id)`

**List IP's from which API access is allowed for a user:**

`allowed_ips, err := api.ListUserApiAllowedIps(user_id)`

**Add new IP's to a user:**

```
user_ips := []string{ my_public_ip, "192.168.7.77", "10.81.12.101" }
user, err := api.AddUserApiAlowedIps(user_id, user_ips)
```

**Remove an IP and forbid API access from it:**

`user, err := api.RemoveUserApiAllowedIp(user_id, ip)`

**Modify a user's API privileges:**

`user, err :=  api.ModifyUserApi(user_id, is_active)`

**Renew a user's API key:**

`user, err := api.RenewUserApiKey(user_id)`

**Retrieve current user permissions:**

`permissions, err := api.GetCurrentUserPermissions()`


### Roles

**List all roles:**

`roles, err := api.ListRoles()`

Alternatively, use the method with query parameters.

`roles, err := api.ListRoles(page, per_page, sort, query, fields)`

To paginate the list of roles received in the response use `page` and `per_page` parameters. Set ` per_page` to the number of roles that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of roles sorted in expected order pass a role property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the role instances that contain it.

To retrieve a collection of roles containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,creation_date"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.

**Retrieve information about a role:**

`role, err := api.GetRole(role_id)`

**Create a role:**

`role, err := api.CreateRole(role_name)`

**Clone a role:**

`role, err := api.CloneRole(role_id, new_role_name)`

**Modify a role:**

`role, err := api.ModifyRole(role_id, new_name, new_description, new_state)`

`ACTIVE` and `DISABLE` are valid values for the state.

**Delete a role:**

`role, err := api.DeleteRole(role_id)`

**Retrieve information about a role's permissions:**

`permissions, err := api.GetRolePermissions(role_id)`

**Modify a role's permissions:**

`role, err := api.ModifyRolePermissions(role_id, permissions)`

**Assign users to a role:**

`role, err := api.AssignRoleUsers(role_id, user_ids)`

`user_ids` is a slice of user ID's.

**List a role's users:**

`users, err := api.ListRoleUsers(role_id)`

**Retrieve information about a role's user:**

`user, err := api.GetRoleUser(role_id, user_id)`

**Remove a role's user:**

`role, err := api.RemoveRoleUser(role_id, user_id)`


### Usages

**List your usages:**

`usages, err := api.ListUsages(period, nil, nil)`

`period` can be set to `"LAST_HOUR"`, `"LAST_24H"`, `"LAST_7D"`, `"LAST_30D"`, `"LAST_365D"` or `"CUSTOM"`. If `period` is set to `"CUSTOM"`, the `start_date` and `end_date` parameters are required to be set in **RFC 3339** date/time format (e.g. `2015-13-12T00:01:00Z`).

`usages, err := api.ListUsages(period, start_date, end_date)`

Additional query parameters can be used.

`usages, err := api.ListUsages(period, start_date, end_date, page, per_page, sort, query, fields)`

To paginate the list of usages received in the response use `page` and `per_page` parameters. Set ` per_page` to the number of usages that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of usages sorted in expected order pass a usages property (e.g. `"name"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the usages instances that contain it.

To retrieve a collection of usages containing only the requested fields pass a list of comma separated properties (e.g. `"id,name"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is set to an empty string, it is ignored in the request.


### Server Appliances

**List all the appliances that you can use to create a server:**

`server_appliances, err := api.ListServerAppliances()`

Alternatively, use the method with query parameters.

`server_appliances, err := api.ListServerAppliances(page, per_page, sort, query, fields)`

To paginate the list of server appliances received in the response use `page` and `per_page` parameters. Set `per_page` to the number of server appliances that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of server appliances sorted in expected order pass a server appliance property (e.g. `"os"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the server appliance instances that contain it.

To retrieve a collection of server appliances containing only the requested fields pass a list of comma separated properties (e.g. `"id,os,architecture"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is blank, it is ignored in the request.

**Retrieve information about specific appliance:**

`server_appliance, err := api.GetServerAppliance(appliance_id)`


### DVD ISO

**List all operative systems and tools that you can load into your virtual DVD unit:**

`dvd_isos, err := api.ListDvdIsos()`

Alternatively, use the method with query parameters.

`dvd_isos, err := api.ListDvdIsos(page, per_page, sort, query, fields)`

To paginate the list of ISO DVDs received in the response use `page` and `per_page` parameters. Set `per_page` to the number of ISO DVDs that will be shown in each page. `page` indicates the current page. When set to an integer value that is less or equal to zero, the parameters are ignored by the framework.

To receive the list of ISO DVDs sorted in expected order pass a ISO DVD property (e.g. `"type"`) in `sort` parameter. Prefix the sorting attribute with `-` sign for sorting in descending order.

Use `query` parameter to search for a string in the response and return only the ISO DVD instances that contain it.

To retrieve a collection of ISO DVDs containing only the requested fields pass a list of comma separated properties (e.g. `"id,name,type"`) in `fields` parameter.

If any of the parameters `sort`, `query` or `fields` is blank, it is ignored in the request.

**Retrieve a specific ISO image:**

`dvd_iso, err := api.GetDvdIso(dvd_id)`


### Ping

**Check if 1&amp;1 REST API is running:**

`response, err := api.Ping()`

If the API is running, the response is a single-element slice `["PONG"]`.

**Validate if 1&amp;1 REST API is running and the authorization token is valid:**

`response, err := api.PingAuth()`

The response should be a single-element slice `["PONG"]` if the API is running and the token is valid.


### Pricing

**Show prices for all available resources in the Cloud Panel:**

`pricing, err := api.GetPricing()`


### Data Centers

**List all 1&amp;1 Cloud Server data centers:**

`datacenters, err := api.ListDatacenters()`

Here is another example of an alternative form of the list function that includes query parameters.

`datacenters, err := api.ListDatacenters(0, 0, "country_code", "DE", "id,country_code")`

**Retrieve a specific data center:**

`datacenter, err := api.GetDatacenter(datacenter_id)`


## Examples

```Go
package main

import (
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"time"
)

func main() {
	//Set an authentication token
	token := oneandone.SetToken("82ee732b8d47e451be5c6ad5b7b56c81")
	//Create an API client
	api := oneandone.New(token, oneandone.BaseUrl)

	// List server appliances
	saps, err := api.ListServerAppliances()

	var sa oneandone.ServerAppliance
	for _, a := range saps {
		if a.Type == "IMAGE" {
			sa = a
		}
	}

	// Create a server
	req := oneandone.ServerRequest{
		Name:        "Example Server",
		Description: "Example server description.",
		ApplianceId: sa.Id,
		PowerOn:	 true,
		Hardware:    oneandone.Hardware{
			Vcores:            1,
			CoresPerProcessor: 1,
			Ram:               2,
			Hdds: []oneandone.Hdd {
				oneandone.Hdd {
						Size:   sa.MinHddSize,
						IsMain: true,
				},
			},
		},
	}

	server_id, server, err := api.CreateServer(&req)

	if err == nil {
		// Wait until server is created and powered on for at most 60 x 10 seconds
		err = api.WaitForState(server, "POWERED_ON", 10, 60)
	}

	// Get the server
	server, err = api.GetServer(server_id)

	// Create a load balancer
	lbr := oneandone.LoadBalancerRequest {
		Name: "Load Balancer Example", 
		Description: "API created load balancer.",
		Method: "ROUND_ROBIN",
		Persistence: oneandone.Bool2Pointer(true),
		PersistenceTime: oneandone.Int2Pointer(1200),
		HealthCheckTest: "TCP",
		HealthCheckInterval: oneandone.Int2Pointer(40),
		Rules: []oneandone.LoadBalancerRule {
				{
					Protocol: "TCP",
					PortBalancer: 80,
					PortServer: 80,
					Source: "0.0.0.0",
				},
		},
	}

	var lb *oneandone.LoadBalancer
	var lb_id string

	lb_id, lb, err = api.CreateLoadBalancer(&lbr)
	if err != nil {
		api.WaitForState(lb, "ACTIVE", 10, 30)
	}

	// Get the load balancer
	lb, err = api.GetLoadBalancer(lb.Id)

	// Assign the load balancer to server's IP
	server, err = api.AssignServerIpLoadBalancer(server.Id, server.Ips[0].Id, lb_id)

	// Create a firewall policy
	fpr := oneandone.FirewallPolicyRequest{
		Name: "Firewall Policy Example", 
		Description: "API created firewall policy.",
		Rules: []oneandone.FirewallPolicyRule {
			{
				Protocol: "TCP",
				PortFrom: oneandone.Int2Pointer(80),
				PortTo: oneandone.Int2Pointer(80),
			},
		},
	}

	var fp *oneandone.FirewallPolicy

	fp_id, fp, err = api.CreateFirewallPolicy(&fpr)
	if err == nil {
		api.WaitForState(fp, "ACTIVE", 10, 30)
	}

	// Get the firewall policy
	fp, err = api.GetFirewallPolicy(fp_id)

	// Add servers IPs to the firewall policy.
	ips := []string{ server.Ips[0].Id }

	fp, err = api.AddFirewallPolicyServerIps(fp.Id, ips)
	if err == nil {
		api.WaitForState(fp, "ACTIVE", 10, 60)
	}

	//Shutdown the server using 'SOFTWARE' method
	server, err = api.ShutdownServer(server.Id, false)
	if err != nil {
		err = api.WaitForState(server, "POWERED_OFF", 5, 20)
	}

	// Delete the load balancer
	lb, err = api.DeleteLoadBalancer(lb.Id)
	if err != nil {
		err = api.WaitUntilDeleted(lb)
	}

	// Delete the firewall policy
	fp, err = api.DeleteFirewallPolicy(fp.Id)
	if err != nil {
		err = api.WaitUntilDeleted(fp)
	}

	// List usages in last 24h
	var usages *oneandone.Usages
	usages, err = api.ListUsages("LAST_24H", nil, nil)

	fmt.Println(usages.Servers)

	// List usages in last 5 hours
	n := time.Now()
	ed := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), 0, time.UTC)
	sd := ed.Add(-(time.Hour * 5))

	usages, err = api.ListUsages("CUSTOM", &sd, &ed)

	//Create a shared storage
	ssr := oneandone.SharedStorageRequest {
		Name: "Shared Storage Example", 
		Description: "API alocated 100 GB disk.",
		Size: oneandone.Int2Pointer(100),
	}

	var ss *oneandone.SharedStorage
	var ss_id string

	ss_id, ss, err = api.CreateSharedStorage(&ssr)
	if err != nil {
		api.WaitForState(ss, "ACTIVE", 10, 30)
	}

	// List shared storages on page 1, 5 results per page and sort by 'name' field.
	// Include only 'name', 'size' and 'minimum_size_allowed' fields in the result.
	var shs []oneandone.SharedStorage
	shs, err = api.ListSharedStorages(1, 5, "name", "", "name,size,minimum_size_allowed")

	// List all shared storages that contain 'example' string
	shs, err = api.ListSharedStorages(0, 0, "", "example", "")

	// Delete the shared storage
	ss, err = api.DeleteSharedStorage(ss_id)
	if err == nil {
		err = api.WaitUntilDeleted(ss)
	}

	// Delete the server
	server, err = api.DeleteServer(server.Id, false)
	if err == nil {
		err = api.WaitUntilDeleted(server)
	}
}

```
The next example illustrates how to create a `TYPO3` application server of a fixed size with an initial password and a firewall policy that has just been created.

```Go
package main

import "github.com/1and1/oneandone-cloudserver-sdk-go"

func main() {
	token := oneandone.SetToken("bde36026df9d548f699ea97e75a7e87f")
	client := oneandone.New(token, oneandone.BaseUrl)

	// Create a new firewall policy
	fpr := oneandone.FirewallPolicyRequest{
		Name: "HTTPS Traffic Policy",
		Rules: []oneandone.FirewallPolicyRule{
			{
				Protocol: "TCP",
				PortFrom: oneandone.Int2Pointer(443),
				PortTo:   oneandone.Int2Pointer(443),
			},
		},
	}

	_, fp, err := client.CreateFirewallPolicy(&fpr)
	if fp != nil && err == nil {
		client.WaitForState(fp, "ACTIVE", 5, 60)

		// Look for the TYPO3 application appliance
		saps, _ := client.ListServerAppliances(0, 0, "", "typo3", "")

		var sa oneandone.ServerAppliance
		for _, a := range saps {
			if a.Type == "APPLICATION" {
				sa = a
				break
			}
		}

		var fixed_flavours []oneandone.FixedInstanceInfo
		var fixed_size_id string

		fixed_flavours, err = client.ListFixedInstanceSizes()
		for _, fl := range fixed_flavours {
			//look for 'M' size
			if fl.Name == "M" {
				fixed_size_id = fl.Id
				break
			}
		}

		req := oneandone.ServerRequest{
			Name:        "TYPO3 Server",
			ApplianceId: sa.Id,
			PowerOn:     true,
			Password:    "ucr_kXW8,.2SdMU",
			Hardware: oneandone.Hardware{
				FixedInsSizeId: fixed_size_id,
			},
			FirewallPolicyId: fp.Id,
		}
		_, server, _ := client.CreateServer(&req)
		if server != nil {
			client.WaitForState(server, "POWERED_ON", 10, 90)
		}
	}
}
```


## Index

```Go
func New(token string, url string) *API
```

```Go
func (api *API) AddFirewallPolicyRules(fp_id string, fp_rules []FirewallPolicyRule) (*FirewallPolicy, error)
```

```Go
func (api *API) AddFirewallPolicyServerIps(fp_id string, ip_ids []string) (*FirewallPolicy, error)
```

```Go
func (api *API) AddLoadBalancerRules(lb_id string, lb_rules []LoadBalancerRule) (*LoadBalancer, error)
```

```Go
func (api *API) AddLoadBalancerServerIps(lb_id string, ip_ids []string) (*LoadBalancer, error)
```

```Go
func (api *API) AddMonitoringPolicyPorts(mp_id string, mp_ports []MonitoringPort) (*MonitoringPolicy, error)
```

```Go
func (api *API) AddMonitoringPolicyProcesses(mp_id string, mp_procs []MonitoringProcess) (*MonitoringPolicy, error)
```

```Go
func (api *API) AddServerHdds(server_id string, hdds *ServerHdds) (*Server, error)
```

```Go
func (api *API) AddSharedStorageServers(st_id string, servers []SharedStorageServer) (*SharedStorage, error)
```

```Go
func (api *API) AddUserApiAlowedIps(user_id string, ips []string) (*User, error)
```

```Go
func (api *API) AssignRoleUsers(role_id string, user_ids []string) (*Role, error)
```

```Go
func (api *API) AssignServerIp(server_id string, ip_type string) (*Server, error)
```

```Go
func (api *API) AssignServerIpFirewallPolicy(server_id string, ip_id string, fp_id string) (*Server, error)
```

```Go
func (api *API) AssignServerIpLoadBalancer(server_id string, ip_id string, lb_id string) (*Server, error)
```

```Go
func (api *API) AssignServerPrivateNetwork(server_id string, pn_id string) (*Server, error)
```

```Go
func (api *API) AttachMonitoringPolicyServers(mp_id string, sids []string) (*MonitoringPolicy, error)
```

```Go
func (api *API) AttachPrivateNetworkServers(pn_id string, sids []string) (*PrivateNetwork, error)
```

```Go
func (api *API) CloneRole(role_id string, name string) (*Role, error)
```

```Go
func (api *API) CloneServer(server_id string, new_name string, datacenter_id string) (*Server, error)
```

```Go
func (api *API) CreateFirewallPolicy(fp_data *FirewallPolicyRequest) (string, *FirewallPolicy, error)
```

```Go
func (api *API) CreateImage(request *ImageConfig) (string, *Image, error)
```

```Go
func (api *API) CreateLoadBalancer(request *LoadBalancerRequest) (string, *LoadBalancer, error)
```

```Go
func (api *API) CreateMonitoringPolicy(mp *MonitoringPolicy) (string, *MonitoringPolicy, error)
```

```Go
func (api *API) CreatePrivateNetwork(request *PrivateNetworkRequest) (string, *PrivateNetwork, error)
```

```Go
func (api *API) CreatePublicIp(ip_type string, reverse_dns string, datacenter_id string) (string, *PublicIp, error)
```

```Go
func (api *API) CreateRole(name string) (string, *Role, error)
```

```Go
func (api *API) CreateServer(request *ServerRequest) (string, *Server, error)
```

```Go
func (api *API) CreateServerEx(request *ServerRequest, timeout int) (string, string, error)
```

```Go
func (api *API) CreateServerSnapshot(server_id string) (*Server, error)
```

```Go
func (api *API) CreateSharedStorage(request *SharedStorageRequest) (string, *SharedStorage, error)
```

```Go
func (api *API) CreateUser(user *UserRequest) (string, *User, error)
```

```Go
func (api *API) CreateVPN(name string, description string, datacenter_id string) (string, *VPN, error)
```

```Go
func (api *API) DeleteFirewallPolicy(fp_id string) (*FirewallPolicy, error)
```

```Go
func (api *API) DeleteFirewallPolicyRule(fp_id string, rule_id string) (*FirewallPolicy, error)
```

```Go
func (api *API) DeleteFirewallPolicyServerIp(fp_id string, ip_id string) (*FirewallPolicy, error)
```

```Go
func (api *API) DeleteImage(img_id string) (*Image, error)
```

```Go
func (api *API) DeleteLoadBalancer(lb_id string) (*LoadBalancer, error)
```

```Go
func (api *API) DeleteLoadBalancerRule(lb_id string, rule_id string) (*LoadBalancer, error)
```

```Go
func (api *API) DeleteLoadBalancerServerIp(lb_id string, ip_id string) (*LoadBalancer, error)
```

```Go
func (api *API) DeleteMonitoringPolicy(mp_id string) (*MonitoringPolicy, error)
```

```Go
func (api *API) DeleteMonitoringPolicyPort(mp_id string, port_id string) (*MonitoringPolicy, error)
```

```Go
func (api *API) DeleteMonitoringPolicyProcess(mp_id string, proc_id string) (*MonitoringPolicy, error)
```

```Go
func (api *API) DeletePrivateNetwork(pn_id string) (*PrivateNetwork, error)
```

```Go
func (api *API) DeletePublicIp(ip_id string) (*PublicIp, error)
```

```Go
func (api *API) DeleteRole(role_id string) (*Role, error)
```

```Go
func (api *API) DeleteServer(server_id string, keep_ips bool) (*Server, error)
```

```Go
func (api *API) DeleteServerHdd(server_id string, hdd_id string) (*Server, error)
```

```Go
func (api *API) DeleteServerIp(server_id string, ip_id string, keep_ip bool) (*Server, error)
```

```Go
func (api *API) DeleteServerSnapshot(server_id string, snapshot_id string) (*Server, error)
```

```Go
func (api *API) DeleteSharedStorage(ss_id string) (*SharedStorage, error)
```

```Go
func (api *API) DeleteSharedStorageServer(st_id string, ser_id string) (*SharedStorage, error)
```

```Go
func (api *API) DeleteUser(user_id string) (*User, error)
```

```Go
func (api *API) DeleteVPN(vpn_id string) (*VPN, error)
```

```Go
func (api *API) DetachPrivateNetworkServer(pn_id string, pns_id string) (*PrivateNetwork, error)
```

```Go
func (api *API) EjectServerDvd(server_id string) (*Server, error)
```

```Go
func (api *API) GetCurrentUserPermissions() (*Permissions, error)
```

```Go
func (api *API) GetDatacenter(dc_id string) (*Datacenter, error)
```

```Go
func (api *API) GetDvdIso(dvd_id string) (*DvdIso, error)
```

```Go
func (api *API) GetFirewallPolicy(fp_id string) (*FirewallPolicy, error)
```

```Go
func (api *API) GetFirewallPolicyRule(fp_id string, rule_id string) (*FirewallPolicyRule, error)
```

```Go
func (api *API) GetFirewallPolicyServerIp(fp_id string, ip_id string) (*ServerIpInfo, error)
```

```Go
func (api *API) GetFixedInstanceSize(fis_id string) (*FixedInstanceInfo, error)
```

```Go
func (api *API) GetImage(img_id string) (*Image, error)
```

```Go
func (api *API) GetLoadBalancer(lb_id string) (*LoadBalancer, error)
```

```Go
func (api *API) GetLoadBalancerRule(lb_id string, rule_id string) (*LoadBalancerRule, error)
```

```Go
func (api *API) GetLoadBalancerServerIp(lb_id string, ip_id string) (*ServerIpInfo, error)
```

```Go
func (api *API) GetLog(log_id string) (*Log, error)
```

```Go
func (api *API) GetMonitoringPolicy(mp_id string) (*MonitoringPolicy, error)
```

```Go
func (api *API) GetMonitoringPolicyPort(mp_id string, port_id string) (*MonitoringPort, error)
```

```Go
func (api *API) GetMonitoringPolicyProcess(mp_id string, proc_id string) (*MonitoringProcess, error)
```

```Go
func (api *API) GetMonitoringPolicyServer(mp_id string, ser_id string) (*Identity, error)
```

```Go
func (api *API) GetMonitoringServerUsage(ser_id string, period string, dates ...time.Time) (*MonServerUsageDetails, error)
```

```Go
func (api *API) GetPricing() (*Pricing, error)
```

```Go
func (api *API) GetPrivateNetwork(pn_id string) (*PrivateNetwork, error)
```

```Go
func (api *API) GetPrivateNetworkServer(pn_id string, server_id string) (*Identity, error)
```

```Go
func (api *API) GetPublicIp(ip_id string) (*PublicIp, error)
```

```Go
func (api *API) GetRole(role_id string) (*Role, error)
```

```Go
func (api *API) GetRolePermissions(role_id string) (*Permissions, error)
```

```Go
func (api *API) GetRoleUser(role_id string, user_id string) (*Identity, error)
```

```Go
func (api *API) GetServer(server_id string) (*Server, error)
```

```Go
func (api *API) GetServerAppliance(sa_id string) (*ServerAppliance, error)
```

```Go
func (api *API) GetServerDvd(server_id string) (*Identity, error)
```

```Go
func (api *API) GetServerHardware(server_id string) (*Hardware, error)
```

```Go
func (api *API) GetServerHdd(server_id string, hdd_id string) (*Hdd, error)
```

```Go
func (api *API) GetServerImage(server_id string) (*Identity, error)
```

```Go
func (api *API) GetServerIp(server_id string, ip_id string) (*ServerIp, error)
```

```Go
func (api *API) GetServerIpFirewallPolicy(server_id string, ip_id string) (*Identity, error)
```

```Go
func (api *API) GetServerPrivateNetwork(server_id string, pn_id string) (*PrivateNetwork, error)
```

```Go
func (api *API) GetServerSnapshot(server_id string) (*ServerSnapshot, error)
```

```Go
func (api *API) GetServerStatus(server_id string) (*Status, error)
```

```Go
func (api *API) GetSharedStorage(ss_id string) (*SharedStorage, error)
```

```Go
func (api *API) GetSharedStorageCredentials() ([]SharedStorageAccess, error)
```

```Go
func (api *API) GetSharedStorageServer(st_id string, ser_id string) (*SharedStorageServer, error)
```

```Go
func (api *API) GetUser(user_id string) (*User, error)
```

```Go
func (api *API) GetUserApi(user_id string) (*UserApi, error)
```

```Go
func (api *API) GetUserApiKey(user_id string) (*UserApiKey, error)
```

```Go
func (api *API) GetVPN(vpn_id string) (*VPN, error)
```

```Go
func (api *API) GetVPNConfigFile(vpn_id string) (string, error)
```

```Go
func (api *API) ListDatacenters(args ...interface{}) ([]Datacenter, error)
```

```Go
func (api *API) ListDvdIsos(args ...interface{}) ([]DvdIso, error)
```

```Go
func (api *API) ListFirewallPolicies(args ...interface{}) ([]FirewallPolicy, error)
```

```Go
func (api *API) ListFirewallPolicyRules(fp_id string) ([]FirewallPolicyRule, error)
```

```Go
func (api *API) ListFirewallPolicyServerIps(fp_id string) ([]ServerIpInfo, error)
```

```Go
func (api *API) ListFixedInstanceSizes() ([]FixedInstanceInfo, error)
```

```Go
func (api *API) ListImages(args ...interface{}) ([]Image, error)
```

```Go
func (api *API) ListLoadBalancerRules(lb_id string) ([]LoadBalancerRule, error)
```

```Go
func (api *API) ListLoadBalancerServerIps(lb_id string) ([]ServerIpInfo, error)
```

```Go
func (api *API) ListLoadBalancers(args ...interface{}) ([]LoadBalancer, error)
```

```Go
func (api *API) ListLogs(period string, sd *time.Time, ed *time.Time, args ...interface{}) ([]Log, error)
```

```Go
func (api *API) ListMonitoringPolicies(args ...interface{}) ([]MonitoringPolicy, error)
```

```Go
func (api *API) ListMonitoringPolicyPorts(mp_id string) ([]MonitoringPort, error)
```

```Go
func (api *API) ListMonitoringPolicyProcesses(mp_id string) ([]MonitoringProcess, error)
```

```Go
func (api *API) ListMonitoringPolicyServers(mp_id string) ([]Identity, error)
```

```Go
func (api *API) ListMonitoringServersUsages(args ...interface{}) ([]MonServerUsageSummary, error)
```

```Go
func (api *API) ListPrivateNetworkServers(pn_id string) ([]Identity, error)
```

```Go
func (api *API) ListPrivateNetworks(args ...interface{}) ([]PrivateNetwork, error)
```

```Go
func (api *API) ListPublicIps(args ...interface{}) ([]PublicIp, error)
```

```Go
func (api *API) ListRoleUsers(role_id string) ([]Identity, error)
```

```Go
func (api *API) ListRoles(args ...interface{}) ([]Role, error)
```

```Go
func (api *API) ListServerAppliances(args ...interface{}) ([]ServerAppliance, error)
```

```Go
func (api *API) ListServerHdds(server_id string) ([]Hdd, error)
```

```Go
func (api *API) ListServerIpLoadBalancers(server_id string, ip_id string) ([]Identity, error)
```

```Go
func (api *API) ListServerIps(server_id string) ([]ServerIp, error)
```

```Go
func (api *API) ListServerPrivateNetworks(server_id string) ([]Identity, error)
```

```Go
func (api *API) ListServers(args ...interface{}) ([]Server, error)
```

```Go
func (api *API) ListSharedStorageServers(st_id string) ([]SharedStorageServer, error)
```

```Go
func (api *API) ListSharedStorages(args ...interface{}) ([]SharedStorage, error)
```

```Go
func (api *API) ListUsages(period string, sd *time.Time, ed *time.Time, args ...interface{}) (*Usages, error)
```

```Go
func (api *API) ListUserApiAllowedIps(user_id string) ([]string, error)
```

```Go
func (api *API) ListUsers(args ...interface{}) ([]User, error)
```

```Go
func (api *API) ListVPNs(args ...interface{}) ([]VPN, error)
```

```Go
func (api *API) LoadServerDvd(server_id string, dvd_id string) (*Server, error)
```

```Go
func (api *API) ModifyMonitoringPolicyPort(mp_id string, port_id string, mp_port *MonitoringPort) (*MonitoringPolicy, error)
```

```Go
func (api *API) ModifyMonitoringPolicyProcess(mp_id string, proc_id string, mp_proc *MonitoringProcess) (*MonitoringPolicy, error)
```

```Go
func (api *API) ModifyRole(role_id string, name string, description string, state string) (*Role, error)
```

```Go
func (api *API) ModifyRolePermissions(role_id string, perm *Permissions) (*Role, error)
```

```Go
func (api *API) ModifyUser(user_id string, user *UserRequest) (*User, error)
```

```Go
func (api *API) ModifyUserApi(user_id string, active bool) (*User, error)
```

```Go
func (api *API) ModifyVPN(vpn_id string, name string, description string) (*VPN, error)
```

```Go
func (api *API) Ping() ([]string, error)
```

```Go
func (api *API) PingAuth() ([]string, error)
```

```Go
func (api *API) RebootServer(server_id string, is_hardware bool) (*Server, error)
```

```Go
func (api *API) ReinstallServerImage(server_id string, image_id string, password string, fp_id string) (*Server, error)
```

```Go
func (api *API) RemoveMonitoringPolicyServer(mp_id string, ser_id string) (*MonitoringPolicy, error)
```

```Go
func (api *API) RemoveRoleUser(role_id string, user_id string) (*Role, error)
```

```Go
func (api *API) RemoveServerPrivateNetwork(server_id string, pn_id string) (*Server, error)
```

```Go
func (api *API) RemoveUserApiAllowedIp(user_id string, ip string) (*User, error)
```

```Go
func (api *API) RenameServer(server_id string, new_name string, new_desc string) (*Server, error)
```

```Go
func (api *API) RenewUserApiKey(user_id string) (*User, error)
```

```Go
func (api *API) ResizeServerHdd(server_id string, hdd_id string, new_size int) (*Server, error)
```

```Go
func (api *API) RestoreServerSnapshot(server_id string, snapshot_id string) (*Server, error)
```

```Go
func (api *API) ShutdownServer(server_id string, is_hardware bool) (*Server, error)
```

```Go
func (api *API) StartServer(server_id string) (*Server, error)
```

```Go
func (api *API) UnassignServerIpFirewallPolicy(server_id string, ip_id string) (*Server, error)
```

```Go
func (api *API) UnassignServerIpLoadBalancer(server_id string, ip_id string, lb_id string) (*Server, error)
```

```Go
func (api *API) UpdateFirewallPolicy(fp_id string, fp_new_name string, fp_new_desc string) (*FirewallPolicy, error)
```

```Go
func (api *API) UpdateImage(img_id string, new_name string, new_desc string, new_freq string) (*Image, error)
```

```Go
func (api *API) UpdateLoadBalancer(lb_id string, request *LoadBalancerRequest) (*LoadBalancer, error)
```

```Go
func (api *API) UpdateMonitoringPolicy(mp_id string, mp *MonitoringPolicy) (*MonitoringPolicy, error)
```

```Go
func (api *API) UpdatePrivateNetwork(pn_id string, request *PrivateNetworkRequest) (*PrivateNetwork, error)
```

```Go
func (api *API) UpdatePublicIp(ip_id string, reverse_dns string) (*PublicIp, error)
```

```Go
func (api *API) UpdateServerHardware(server_id string, hardware *Hardware) (*Server, error)
```

```Go
func (api *API) UpdateSharedStorage(ss_id string, request *SharedStorageRequest) (*SharedStorage, error)
```

```Go
func (api *API) UpdateSharedStorageCredentials(new_pass string) ([]SharedStorageAccess, error)
```

```Go
func (api *API) WaitForState(in ApiInstance, state string, sec time.Duration, count int) error
```

```Go
func (api *API) WaitUntilDeleted(in ApiInstance) error
```

```Go
func (fp *FirewallPolicy) GetState() (string, error)
```

```Go
func (im *Image) GetState() (string, error)
```

```Go
func (lb *LoadBalancer) GetState() (string, error)
```

```Go
func (mp *MonitoringPolicy) GetState() (string, error)
```

```Go
func (pn *PrivateNetwork) GetState() (string, error)
```

```Go
func (ip *PublicIp) GetState() (string, error)
```

```Go
func (role *Role) GetState() (string, error)
```

```Go
func (s *Server) GetState() (string, error)
```

```Go
func (ss *SharedStorage) GetState() (string, error)
```

```Go
func (u *User) GetState() (string, error)
```

```Go
func (u *User) GetState() (string, error)
```

```Go
func (vpn *VPN) GetState() (string, error)
```

```Go
func Bool2Pointer(input bool) *bool
```

```Go
func Int2Pointer(input int) *int
```

```Go
func (bp *BackupPerm) SetAll(value bool)
```

```Go
func (fp *FirewallPerm) SetAll(value bool)
```

```Go
func (imp *ImagePerm) SetAll(value bool)
```

```Go
unc (inp *InvoicePerm) SetAll(value bool)
```

```Go
func (ipp *IPPerm) SetAll(value bool)
```

```Go
func (lbp *LoadBalancerPerm) SetAll(value bool)
```

```Go
func (lp *LogPerm) SetAll(value bool)
```

```Go
func (mcp *MonitorCenterPerm) SetAll(value bool)
```

```Go
func (mpp *MonitorPolicyPerm) SetAll(value bool)
```

```Go
func (p *Permissions) SetAll(v bool)
```

```Go
func (pnp *PrivateNetworkPerm) SetAll(value bool)
```

```Go
func (rp *RolePerm) SetAll(value bool)
```

```Go
func (sp *ServerPerm) SetAll(value bool)
```

```Go
func (ssp *SharedStoragePerm) SetAll(value bool)
```

```Go
func (up *UsagePerm) SetAll(value bool)
```

```Go
func (up *UserPerm) SetAll(value bool)
```

```Go
func (vpnp *VPNPerm) SetAll(value bool)
```

```Go
func SetBaseUrl(newbaseurl string) string
```

```Go
func SetToken(newtoken string) string
```


package azurerm

import (
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
)

func (armClient *ArmClient) ListNetworkInterfaceConfigurations(resourceGroupName, networkInterfaceName string) []map[string]string {
	ipConfigurations := make([]map[string]string, 0)
	interfaces, _ := armClient.ifaceClient.List(resourceGroupName)
	for _, val := range *interfaces.Value {
		for _, ip := range *val.IPConfigurations {
			/*
				addressPools := make([]string, 0, len(*ip.LoadBalancerBackendAddressPools))
				for _, pool := range *ip.LoadBalancerBackendAddressPools {
					addressPools = append(addressPools, *pool.ID)
				}
				natRules := make([]string, 0, len(*ip.LoadBalancerInboundNatRules))
				for _, pool := range *ip.LoadBalancerInboundNatRules {
					natRules = append(natRules, *pool.ID)
				}
			*/

			ipConfiguration := map[string]string{
				"name":                                    *ip.Name,
				"subnet_id":                               *ip.Subnet.ID,
				"interface":                               *val.Name,
				"private_ip_address":                      *ip.PrivateIPAddress,
				"private_ip_address_allocation":           string(ip.PrivateIPAllocationMethod),
				"load_balancer_backend_address_pools_ids": "", //strings.Join(addressPools, ","),
				"load_balancer_inbound_nat_rules_ids ":    "", // strings.Join(natRules, ","),
			}
			if ip.PublicIPAddress != nil {
				ipConfiguration["public_ip_address_id"] = *ip.PublicIPAddress.ID
			}
			ipConfigurations = append(ipConfigurations, ipConfiguration)
		}
	}

	return ipConfigurations
}

func (armClient *ArmClient) ListResourcesByGroup(resourceGroupName, filters, expand string) (m map[string][]string, err error) {
	m = make(map[string][]string)
	results, err := armClient.resourceGroupClient.ListResources(resourceGroupName, filters, expand, nil)
	if err != nil {
		log.Println(err.Error())
		return m, nil
	}

	if &results != nil {
		for _, v := range *results.Value {
			t := *v.Type
			id := *v.ID
			name := *v.Name
			if _, ok := m[t]; !ok {
				m[t] = make([]string, 0)
			}

			if t == "Microsoft.Network/virtualNetworks" {
				// Look for Subnets
				res, _ := armClient.subnetClient.List(resourceGroupName, name)
				if &res != nil {
					for _, sub := range *res.Value {
						subid := *sub.ID
						subT := "Microsoft.Network/subnets"
						if _, ok := m[subT]; !ok {
							m[subT] = make([]string, 0)
						}

						m[subT] = append(m["azurerm_subnet"], subid)
					}
				}
			}

			if t == "Microsoft.Storage/storageAccounts" {
				// Look for Storage Containers
				conT := "Microsoft.Storage/storageContainers"
				m[conT] = make([]string, 0)
				blobClient, _, _ := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, name)
				containers, err := blobClient.ListContainers(storage.ListContainersParameters{})
				if err != nil {
					log.Println(err.Error())
				}
				for _, container := range containers.Containers {
					access, err := blobClient.GetContainerPermissions(container.Name, 0, "")
					if err != nil {
						log.Println(err.Error())
					}
					t := string(access.AccessType)
					parts := strings.Split(id, "/")

					tid := "/" + conT + "/" + parts[4] + "::" + parts[len(parts)-1] + "::" + container.Name + "::" + t
					m[conT] = append(m[conT], tid)
				}
			}

			m[t] = append(m[t], id)
		}
	}

	// Import resource groups
	rg, err := armClient.resourceGroupClient.Get(resourceGroupName)
	if &rg != nil {
		t := "Microsoft.Network/loadBalancers"
		m[t] = append(m[t], *rg.ID)
	}

	for k, v := range m {
		log.Println(k)
		for _, s := range v {
			log.Println(" - " + s)
		}
	}

	return m, nil
}

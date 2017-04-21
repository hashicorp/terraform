package azurerm

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGroupAndLBNameFromId(loadBalancerId string) (string, string, error) {
	id, err := parseAzureResourceID(loadBalancerId)
	if err != nil {
		return "", "", err
	}
	name := id.Path["loadBalancers"]
	resGroup := id.ResourceGroup

	return resGroup, name, nil
}

func retrieveLoadBalancerById(loadBalancerId string, meta interface{}) (*network.LoadBalancer, bool, error) {
	loadBalancerClient := meta.(*ArmClient).loadBalancerClient

	resGroup, name, err := resourceGroupAndLBNameFromId(loadBalancerId)
	if err != nil {
		return nil, false, errwrap.Wrapf("Error Getting LoadBalancer Name and Group: {{err}}", err)
	}

	resp, err := loadBalancerClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Error making Read request on Azure LoadBalancer %s: %s", name, err)
	}

	return &resp, true, nil
}

func findLoadBalancerBackEndAddressPoolByName(lb *network.LoadBalancer, name string) (*network.BackendAddressPool, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.BackendAddressPools == nil {
		return nil, -1, false
	}

	for i, apc := range *lb.LoadBalancerPropertiesFormat.BackendAddressPools {
		if apc.Name != nil && *apc.Name == name {
			return &apc, i, true
		}
	}

	return nil, -1, false
}

func findLoadBalancerFrontEndIpConfigurationByName(lb *network.LoadBalancer, name string) (*network.FrontendIPConfiguration, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations == nil {
		return nil, -1, false
	}

	for i, feip := range *lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations {
		if feip.Name != nil && *feip.Name == name {
			return &feip, i, true
		}
	}

	return nil, -1, false
}

func findLoadBalancerRuleByName(lb *network.LoadBalancer, name string) (*network.LoadBalancingRule, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.LoadBalancingRules == nil {
		return nil, -1, false
	}

	for i, lbr := range *lb.LoadBalancerPropertiesFormat.LoadBalancingRules {
		if lbr.Name != nil && *lbr.Name == name {
			return &lbr, i, true
		}
	}

	return nil, -1, false
}

func findLoadBalancerNatRuleByName(lb *network.LoadBalancer, name string) (*network.InboundNatRule, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.InboundNatRules == nil {
		return nil, -1, false
	}

	for i, nr := range *lb.LoadBalancerPropertiesFormat.InboundNatRules {
		if nr.Name != nil && *nr.Name == name {
			return &nr, i, true
		}
	}

	return nil, -1, false
}

func findLoadBalancerNatPoolByName(lb *network.LoadBalancer, name string) (*network.InboundNatPool, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.InboundNatPools == nil {
		return nil, -1, false
	}

	for i, np := range *lb.LoadBalancerPropertiesFormat.InboundNatPools {
		if np.Name != nil && *np.Name == name {
			return &np, i, true
		}
	}

	return nil, -1, false
}

func findLoadBalancerProbeByName(lb *network.LoadBalancer, name string) (*network.Probe, int, bool) {
	if lb == nil || lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.Probes == nil {
		return nil, -1, false
	}

	for i, p := range *lb.LoadBalancerPropertiesFormat.Probes {
		if p.Name != nil && *p.Name == name {
			return &p, i, true
		}
	}

	return nil, -1, false
}

func loadbalancerStateRefreshFunc(client *ArmClient, resourceGroupName string, loadbalancer string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.loadBalancerClient.Get(resourceGroupName, loadbalancer, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in loadbalancerStateRefreshFunc to Azure ARM for LoadBalancer '%s' (RG: '%s'): %s", loadbalancer, resourceGroupName, err)
		}

		return res, *res.LoadBalancerPropertiesFormat.ProvisioningState, nil
	}
}

func validateLoadBalancerPrivateIpAddressAllocation(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	if value != "static" && value != "dynamic" {
		errors = append(errors, fmt.Errorf("LoadBalancer Allocations can only be Static or Dynamic"))
	}
	return
}

// sets the loadbalancer_id in the ResourceData from the sub resources full id
func loadBalancerSubResourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	r, err := regexp.Compile(`.+\/loadBalancers\/.+?\/`)
	if err != nil {
		return nil, err
	}

	lbID := strings.TrimSuffix(r.FindString(d.Id()), "/")
	parsed, err := parseAzureResourceID(lbID)
	if err != nil {
		return nil, fmt.Errorf("unable to parse loadbalancer id from %s", d.Id())
	}

	if parsed.Path["loadBalancers"] == "" {
		return nil, fmt.Errorf("parsed ID is invalid")
	}

	d.Set("loadbalancer_id", lbID)
	return []*schema.ResourceData{d}, nil
}

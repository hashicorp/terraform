package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmLoadBalancer returns the *schema.Resource
// associated to load balancer resources on ARM.
func resourceArmLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerCreate,
		Read:   resourceArmLoadBalancerRead,
		Update: resourceArmLoadBalancerUpdate,
		Delete: resourceArmLoadBalancerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"frontend_ip": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true, // TODET
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"private_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"allocation_method": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"subnet": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"public_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"inbound_nat_rules": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"inbound_nat_pools": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"outbound_nat_rules": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"load_balancing_rules": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Set: makeHashFunction(
					[]string{"name", "allocation_method", "subnet", "public_ip_address"},
					[]string{"inbound_nat_rules", "inbound_nat_pools", "outbound_nat_rules", "load_balancing_rules"},
				),
			},

			"backend_address_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true, // TODET
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"backend_ips": &schema.Schema{
							Type:     schema.TypeList,
							Required: true, // TODET
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"load_balancing_rules": &schema.Schema{
							Type:     schema.TypeList,
							Required: true, // TODET
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"outbound_nat_rule": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
					},
				},
				Set: makeHashFunction(
					[]string{"name", "outbound_nat_rule"},
					[]string{"backend_ips", "load_balancing_rules"},
				),
			},

			"load_balancing_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true, // TODET
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
						"backend_address_pool": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODED
						},
						"probe": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
						"load_distribution": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
						"frontend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"backend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"enable_floating_ip": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true, // TODET
						},
					},
				},
				Set: makeHashFunction(
					[]string{
						"name", "frontend_ip", "backend_address_pool", "probe", "protocol",
						"load_distribution", "frontend_port", "backend_port", "timeout", "enable_floating_ip",
					},
					nil,
				),
			},

			"probe": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"load_balancing_rules": &schema.Schema{
							Type:     schema.TypeList,
							Required: true, // TODET
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"time_interval": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"probe_number": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true, // TODET
						},
						"request_path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true, // TODET
						},
					},
				},
				Set: makeHashFunction(
					[]string{"name", "protocol", "port", "time_interval", "probe_number", "request_path"},
					[]string{"load_balancing_rules"},
				),
			},

			"inbound_nat_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"backend_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"backend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"enable_floating_ip": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				Set: makeHashFunction(
					[]string{
						"name", "frontend_ip", "backend_ip", "protocol",
						"frontend_port", "backend_port", "timeout", "enable_floating_ip",
					},
					nil,
				),
			},

			"outbound_nat_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"allocated_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"frontend_ips": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"backend_address_pool": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: makeHashFunction(
					[]string{"name", "allocated_port", "backend_address_pool"},
					[]string{"frontend_ips"},
				),
			},

			"inbound_nat_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_ip": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_port_start_range": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"frontend_port_end_range": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"backend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: makeHashFunction(
					[]string{
						"name", "frontend_ip", "protocol", "frontend_port_start_range",
						"frontend_port_end_range", "backend_port",
					},
					nil,
				),
			},
		},
	}
}

// resourceArmLoadBalancerCreate goes ahead and creates the specified ARM load balancer.
func resourceArmLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	lbClient := meta.(*AzureClient).armClient.loadBalancerClient

	// first; fetch a bunch of fields:
	typ := d.Get("type").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGrp := d.Get("resource_group_name").(string)

	loadBalancer := network.LoadBalancer{
		Name:       &name,
		Type:       &typ,
		Location:   &location,
		Properties: &network.LoadBalancerPropertiesFormat{},
	}

	// then, read all the frontend IP configurations:
	if fips, ok := d.GetOk("frontend_ip"); ok {
		configMaps := readSet(fips, []string{"inbound_nat_rules", "inbound_nat_pools", "outbound_nat_rules", "load_balancing_rules"})

		// iterate through all the frontend ip configurations and add them:
		fipconfs := []network.FrontendIPConfiguration{}
		for _, config := range configMaps {
			name := config["name"].(string)
			allmeth := config["allocation_method"].(string)
			subnet := config["subnet"].(string)
			pubip := config["public"].(string)
			inNatRules := config["inbound_nat_rules"].([]string)
			inNatPools := config["inbound_nat_pools"].([]string)
			outNatRules := config["outbound_nat_rules"].([]string)
			rules := config["load_balancing_rules"].([]string)

			fipconfs = append(fipconfs, network.FrontendIPConfiguration{
				Name: &name,
				Properties: &network.FrontendIPConfigurationPropertiesFormat{
					PrivateIPAddress:          &pubip,
					PrivateIPAllocationMethod: network.IPAllocationMethod(allmeth),
					Subnet:             makeNetworkSubResourceRef(subnet),
					PublicIPAddress:    makeNetworkSubResourceRef(pubip),
					InboundNatRules:    makeNetworkSubResourcesListRef(inNatRules),
					InboundNatPools:    makeNetworkSubResourcesListRef(inNatPools),
					OutboundNatRules:   makeNetworkSubResourcesListRef(outNatRules),
					LoadBalancingRules: makeNetworkSubResourcesListRef(rules),
				},
			})
		}

		// then, add the frontend ip configs to the load balancer:
		loadBalancer.Properties.FrontendIPConfigurations = &fipconfs
	}

	// now; check for any set "backend_address_pools":
	if baps, ok := d.GetOk("backend_address_pool"); ok {
		poolMaps := readSet(baps, []string{"backend_ips", "load_balancing_rules"})

		// iterate through all the pools and declare them:
		pools := []network.BackendAddressPool{}
		for _, pool := range poolMaps {
			name := pool["name"].(string)
			ipConfigs := pool["backend_ips"].([]string)
			lbRules := pool["load_balancing_rules"].([]string)
			outRule := pool["outbound_nat_rule"].(string)

			pools = append(pools, network.BackendAddressPool{
				Name: &name,
				Properties: &network.BackendAddressPoolPropertiesFormat{
					BackendIPConfigurations: makeNetworkSubResourcesListRef(ipConfigs),
					LoadBalancingRules:      makeNetworkSubResourcesListRef(lbRules),
					OutboundNatRule:         makeNetworkSubResourceRef(outRule),
				},
			})
		}

		loadBalancer.Properties.BackendAddressPools = &pools
	}

	// now; "load_balancing_rule"s:
	if lbRules, ok := d.GetOk("load_balancing_rules"); ok {
		rulesMaps := readSet(lbRules, nil)

		// iterate through the rules and add them all:
		rules := []network.LoadBalancingRule{}
		for _, rule := range rulesMaps {
			name := rule["name"].(string)
			fipConfig := rule["frontend_ip"].(string)
			backendPool := rule["backend_address_pool"].(string)
			probe := rule["probe"].(string)
			proto := rule["protocol"].(string)
			loadDist := rule["load_distribution"].(string)
			frontPort := rule["frontend_port"].(int)
			backPort := rule["backend_port"].(int)
			timeout := rule["timeout"].(int)
			enableFloating := rule["enable_floating_ip"].(bool)

			rules = append(rules, network.LoadBalancingRule{
				Name: &name,
				Properties: &network.LoadBalancingRulePropertiesFormat{
					FrontendIPConfiguration: makeNetworkSubResourceRef(fipConfig),
					BackendAddressPool:      makeNetworkSubResourceRef(backendPool),
					Probe:                   makeNetworkSubResourceRef(probe),
					Protocol:                network.TransportProtocol(proto),
					LoadDistribution:        network.LoadDistribution(loadDist),
					FrontendPort:            &frontPort,
					BackendPort:             &backPort,
					IdleTimeoutInMinutes:    &timeout,
					EnableFloatingIP:        &enableFloating,
				},
			})

			loadBalancer.Properties.LoadBalancingRules = &rules
		}
	}

	// now, "probe"s:
	if probeSet, ok := d.GetOk("probe"); ok {
		probeMaps := readSet(probeSet, []string{"load_balancing_rules"})

		// iterate through all probes and add them to our load balancer:
		probes := []network.Probe{}
		for _, probe := range probeMaps {
			name := probe["name"].(string)
			lbRules := probe["load_balancing_rules"].([]string)
			proto := probe["protocol"].(string)
			port := probe["port"].(int)
			interval := probe["time_interval"].(int)
			number := probe["probe_number"].(int)
			reqPath := probe["request_path"].(string)

			probes = append(probes, network.Probe{
				Name: &name,
				Properties: &network.ProbePropertiesFormat{
					LoadBalancingRules: makeNetworkSubResourcesListRef(lbRules),
					Protocol:           network.ProbeProtocol(proto),
					Port:               &port,
					IntervalInSeconds:  &interval,
					NumberOfProbes:     &number,
					RequestPath:        &reqPath,
				},
			})
		}

		loadBalancer.Properties.Probes = &probes
	}

	// now; "inbound_nat_rule"s:
	if inRulesSet, ok := d.GetOk("inbound_nat_rule"); ok {
		inRuleMaps := readSet(inRulesSet, nil)

		// iterate and collect all the rules from the set:
		inNatRules := []network.InboundNatRule{}
		for _, rule := range inRuleMaps {
			name := rule["name"].(string)
			frontip := rule["frontend_ip"].(string)
			backip := rule["backend_ip"].(string)
			proto := rule["protocol"].(string)
			frontPort := rule["frontend_port"].(int)
			backPort := rule["backend_port"].(int)
			timeout := rule["timeout"].(int)
			enableFloating := rule["enable_floating_ip"].(bool)

			inNatRules = append(inNatRules, network.InboundNatRule{
				Name: &name,
				Properties: &network.InboundNatRulePropertiesFormat{
					FrontendIPConfiguration: makeNetworkSubResourceRef(frontip),
					BackendIPConfiguration:  makeNetworkSubResourceRef(backip),
					Protocol:                network.TransportProtocol(proto),
					FrontendPort:            &frontPort,
					BackendPort:             &backPort,
					IdleTimeoutInMinutes:    &timeout,
					EnableFloatingIP:        &enableFloating,
				},
			})
		}

		loadBalancer.Properties.InboundNatRules = &inNatRules
	}

	// now; "outbound_nat_rule"s:
	if outNatRulesSet, ok := d.GetOk("outbound_nat_rule"); ok {
		outRuleMaps := readSet(outNatRulesSet, []string{"frontend_ips"})

		// iterate and record each outward rule:
		outNatRules := []network.OutboundNatRule{}
		for _, rule := range outRuleMaps {
			name := rule["name"].(string)
			frontipis := rule["frontend_ips"].([]string)
			port := rule["allocated_port"].(int)
			backAddressPool := rule["backend_address_pool"].(string)

			outNatRules = append(outNatRules, network.OutboundNatRule{
				Name: &name,
				Properties: &network.OutboundNatRulePropertiesFormat{
					AllocatedOutboundPorts:   &port,
					FrontendIPConfigurations: makeNetworkSubResourcesListRef(frontipis),
					BackendAddressPool:       makeNetworkSubResourceRef(backAddressPool),
				},
			})
		}

		loadBalancer.Properties.OutboundNatRules = &outNatRules
	}

	// and finally; "inbound_nat_pool"s:
	if natPoolsSet, ok := d.GetOk("inbound_nat_pool"); ok {
		natPoolMaps := readSet(natPoolsSet, nil)

		// iterate through all the declared pools:
		inNatPools := []network.InboundNatPool{}
		for _, pool := range natPoolMaps {
			name := pool["name"].(string)
			frontip := pool["frontend_ip"].(string)
			proto := pool["protocol"].(string)
			portStart := pool["frontend_port_start_range"].(int)
			portStop := pool["frontend_port_end_range"].(int)
			port := pool["backend_port"].(int)

			inNatPools = append(inNatPools, network.InboundNatPool{
				Name: &name,
				Properties: &network.InboundNatPoolPropertiesFormat{
					FrontendIPConfiguration: makeNetworkSubResourceRef(frontip),
					Protocol:                network.TransportProtocol(proto),
					FrontendPortRangeStart:  &portStart,
					FrontendPortRangeEnd:    &portStop,
					BackendPort:             &port,
				},
			})
		}

		loadBalancer.Properties.InboundNatPools = &inNatPools
	}

	_, err := lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	return err
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	lbClient := meta.(*AzureClient).armClient.loadBalancerClient

	name := d.Get("name").(string)
	resGrp := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of LoadBalancer '%s' off Azure.", name)

	loadBalancer, err := lbClient.Get(resGrp, name)
	if err != nil {
		return fmt.Errorf("Error reading the state of the LoadBalancer off Azure: %s", err)
	}

	log.Printf("[INFO] Succesfully retrieved details for LoadBalancer '%s'.", *loadBalancer.Name)

	// read all the required details:

	d.Set("frontend_ip", extractSet(
		d.Get("frontend_ip").(*schema.Set),
		loadBalancer.Properties.FrontendIPConfigurations,
		[]setElementField{
			setElementField{"private_ip_address", "privateIPAddress", schema.TypeString, false},
			setElementField{"allocation_method", "privateIPAllocationMethod", schema.TypeString, false},
			setElementField{"subnet", "subnet", schema.TypeString, false},
			setElementField{"public_ip_address", "publicIPAddress", schema.TypeString, false},
			setElementField{"inbound_nat_rules", "inboundNatRules", schema.TypeList, true},
			setElementField{"inbound_nat_pools", "inboundNatPools", schema.TypeList, true},
			setElementField{"outbound_nat_rules", "outboundNatRules", schema.TypeList, true},
			setElementField{"load_balancing_rules", "loadBalancingRules", schema.TypeList, true},
		},
	))

	d.Set("backend_address_pool", extractSet(
		d.Get("backend_address_pool").(*schema.Set),
		loadBalancer.Properties.BackendAddressPools,
		[]setElementField{
			setElementField{"backend_ips", "backendIPConfigurations", schema.TypeList, true},
			setElementField{"load_balancing_rules", "loadBalancingRules", schema.TypeList, true},
			setElementField{"outbound_nat_rule", "outboundNatRule", schema.TypeString, true},
		},
	))

	d.Set("load_balancing_rule", extractSet(
		d.Get("load_balancing_rule").(*schema.Set),
		loadBalancer.Properties.LoadBalancingRules,
		[]setElementField{
			setElementField{"frontend_ip", "frontendIPConfiguration", schema.TypeString, true},
			setElementField{"backend_address_pool", "backendAddressPool", schema.TypeString, true},
			setElementField{"probe", "probe", schema.TypeString, true},
			setElementField{"protocol", "protocl", schema.TypeString, true},
			setElementField{"load_distribution", "loadDistribution", schema.TypeString, true},
			setElementField{"frontend_port", "frontendPort", schema.TypeInt, false},
			setElementField{"backend_port", "backendPort", schema.TypeInt, false},
			setElementField{"timeout", "timeout", schema.TypeInt, false},
			setElementField{"enable_floating_ip", "enableFloatingIP", schema.TypeBool, false},
		},
	))

	d.Set("probe", extractSet(
		d.Get("probe").(*schema.Set),
		loadBalancer.Properties.Probes,
		[]setElementField{
			setElementField{"load_balancing_rules", "loadBalancingRules", schema.TypeList, true},
			setElementField{"protocol", "protocol", schema.TypeString, false},
			setElementField{"port", "port", schema.TypeInt, false},
			setElementField{"time_interval", "intervalInSeconds", schema.TypeInt, false},
			setElementField{"probe_number", "numberOfProbes", schema.TypeInt, false},
			setElementField{"request_path", "requestPath", schema.TypeString, false},
		},
	))

	d.Set("inbound_nat_rule", extractSet(
		d.Get("inbound_nat_rule").(*schema.Set),
		loadBalancer.Properties.InboundNatRules,
		[]setElementField{
			setElementField{"frontend_ip", "frontendIPConfiguration", schema.TypeString, true},
			setElementField{"backend_ip", "backendIPConfiguration", schema.TypeString, true},
			setElementField{"protocol", "protocol", schema.TypeString, false},
			setElementField{"frontend_port", "frontendPort", schema.TypeInt, false},
			setElementField{"backend_port", "backendPort", schema.TypeInt, false},
			setElementField{"timeout", "timeout", schema.TypeInt, false},
			setElementField{"enable_floating_ip", "enableFloatingIP", schema.TypeInt, false},
		},
	))

	d.Set("outbound_nat_rule", extractSet(
		d.Get("outbound_nat_rule").(*schema.Set),
		loadBalancer.Properties.OutboundNatRules,
		[]setElementField{
			// NOTE: must be changed to singular following the SDK's update:
			setElementField{"allocated_port", "allocatedOutboundPort", schema.TypeInt, false},
			setElementField{"frontend_ips", "frontendIPConfigurations", schema.TypeList, true},
			setElementField{"backend_address_pool", "backendAddressPool", schema.TypeString, true},
		},
	))

	d.Set("inbound_nat_pool", extractSet(
		d.Get("inbound_nat_pool").(*schema.Set),
		loadBalancer.Properties.InboundNatPools,
		[]setElementField{
			setElementField{"frontend_ip", "frontendIPConfiguration", schema.TypeString, true},
			setElementField{"protocol", "protocol", schema.TypeString, false},
			setElementField{"frontend_port_start_range", "frontendPortRangeStart", schema.TypeInt, false},
			setElementField{"frontend_port_end_range", "frontendPortRangeEnd", schema.TypeInt, false},
			setElementField{"backend_port", "backendPort", schema.TypeInt, false},
		},
	))

	return nil
}

// resourceArmLoadBalancerUpdate goes ahead and updates the corresponding ARM load balancer.
func resourceArmLoadBalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: considering Create's idempotency, Update may safely call Create:
	return resourceArmLoadBalancerCreate(d, meta)
}

// resourceArmLoadBalancerDelete deletes the specified ARM load balancer.
func resourceArmLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
	lbClient := meta.(*AzureClient).armClient.loadBalancerClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Sending Load Balancer delete request to Azure.")
	_, err := lbClient.Delete(resGroup, name)

	return err
}

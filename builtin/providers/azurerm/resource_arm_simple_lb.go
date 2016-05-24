package azurerm

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmLoadBalancer returns the *schema.Resource
// associated to load balancer resources on ARM.
func resourceArmSimpleLb() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSimpleLbCreate,
		Read:   resourceArmSimpleLbRead,
		Update: resourceArmSimpleLbUpdate,
		Delete: resourceArmSimpleLbDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"backend_pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"frontend_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

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
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: azureRMNormalizeLocation,
				ForceNew:  true,
			},
			"frontend_private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"frontend_allocation_method": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAllocationMethod,
				ForceNew:     true,
			},
			"frontend_subnet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"frontend_public_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
						"protocol": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateProbeProtocolType,
						},
						"request_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"interval": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"number_of_probes": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"probe_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceARMLoadBalancerProbeHash,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"rule_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"protocol": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateProtocolType,
						},
						"probe_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"load_distribution": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateLoadDistribution,
						},
						"frontend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"backend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceARMLoadBalancerRuleHash,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceARMLoadBalancerRuleHash(v interface{}) int {
	m := v.(map[string]interface{})
	rule := fmt.Sprintf("%s-%s-%s-%s-%d-%d", m["name"].(string), m["protocol"].(string), m["probe_name"].(string), m["load_distribution"].(string), m["frontend_port"], m["backend_port"])
	return hashcode.String(strings.ToLower(rule))
}

func resourceARMLoadBalancerProbeHash(v interface{}) int {
	m := v.(map[string]interface{})
	rule := fmt.Sprintf("%s-%s-%d-%d-%d", m["name"].(string), m["protocol"].(string), m["port"], m["number_of_probes"], m["interval"])
	if m["request_path"] != nil {
		rule = rule + "-" + m["request_path"].(string)
	}
	return hashcode.String(strings.ToLower(rule))
}

func validateAllocationMethod(allocation interface{}, k string) (ws []string, errors []error) {
	if !isOneOf([]string{"static", "dynamic"}, allocation) {
		errors = append(errors, fmt.Errorf("Allocation method can only be Static or Dynamic"))
	}
	return
}

func validateProtocolType(protocol interface{}, k string) (ws []string, errors []error) {
	if !isOneOf([]string{"tcp", "udp"}, protocol) {
		errors = append(errors, fmt.Errorf("Protocol can only be tcp or udp"))
	}
	return
}

func validateProbeProtocolType(protocol interface{}, k string) (ws []string, errors []error) {
	if !isOneOf([]string{"tcp", "http"}, protocol) {
		errors = append(errors, fmt.Errorf("Protocol can only be tcp or http"))
	}
	return
}

func validateLoadDistribution(distribution interface{}, k string) (ws []string, errors []error) {
	if !isOneOf([]string{"default", "sourceip", "sourceipprotocol"}, distribution) {
		errors = append(errors, fmt.Errorf("Load Distribution can only be default, sourceIp, or sourceIpProtocol"))
	}
	return
}

func findRuleByName(ruleArray *[]network.LoadBalancingRule, ruleName string) (network.LoadBalancingRule, error) {
	for _, rule := range *ruleArray {
		if *rule.Name == ruleName {
			log.Printf("[findRuleByName] found rule %v", rule)
			return rule, nil
		}
	}
	return network.LoadBalancingRule{}, fmt.Errorf("Error loading the rule named %s", ruleName)
}

func findProbeByName(probeArray *[]network.Probe, probeName string) (network.Probe, error) {
	for _, probe := range *probeArray {
		if *probe.Name == probeName {
			log.Printf("[findProbeByName] found probe %v", probe)
			return probe, nil
		}
	}

	return network.Probe{}, fmt.Errorf("Error loading the probe named %s", probeName)
}

func findProbeById(probeArray *[]network.Probe, probeId string) (network.Probe, error) {
	for _, probe := range *probeArray {
		if *probe.ID == probeId {
			log.Printf("[findProbeById] found probe %v", probe)
			return probe, nil
		}
	}

	return network.Probe{}, fmt.Errorf("Error finding the probe with ID %s", probeId)
}

func pullOutLbRules(d *schema.ResourceData, loadBalancer network.LoadBalancer) (*[]network.LoadBalancingRule, error) {
	log.Printf("[resourceArmSimpleLb] pullOutLbRules[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutLbRules[exit]")

	log.Printf("[resourceArmSimpleLb] pullOutLbRules %v", *loadBalancer.Properties.Probes)
	outRules := []network.LoadBalancingRule{}

	backendPoolID := (*loadBalancer.Properties.BackendAddressPools)[0].ID
	frontendIpID := (*loadBalancer.Properties.FrontendIPConfigurations)[0].ID
	log.Printf("[resourceArmSimpleLb] pullOutLbRules will use frontend %s and backend %s", *frontendIpID, *backendPoolID)

	rules := d.Get("rule").(*schema.Set)
	if rules.Len() == 0 {
		log.Printf("[resourceArmSimpleLb] pullOutLbRules no rules found")
		return &outRules, nil
	}

	log.Printf("[resourceArmSimpleLb] pullOutLbRules found %d rules in plan", rules.Len())
	for _, rule := range rules.List() {
		rule := rule.(map[string]interface{})
		log.Printf("[resourceArmSimpleLb] pullOutLbRules %v", rule)

		ruleName := rule["name"].(string)

		existingRule, err := findRuleByName(loadBalancer.Properties.LoadBalancingRules, ruleName)
		if err == nil {
			log.Printf("[resourceArmSimpleLb] pullOutLbRules found the existing rule %s", ruleName)
		}

		probe, err := findProbeByName(loadBalancer.Properties.Probes, rule["probe_name"].(string))
		if err != nil {
			return nil, err
		}
		log.Printf("[resourceArmSimpleLb] pullOutLbRules rule %s is using probe %s", ruleName, *probe.ID)

		props := network.LoadBalancingRulePropertiesFormat{
			Protocol:                network.TransportProtocol(rule["protocol"].(string)),
			LoadDistribution:        network.LoadDistribution(rule["load_distribution"].(string)),
			FrontendPort:            Int(rule["frontend_port"]),
			BackendPort:             Int(rule["backend_port"]),
			Probe:                   &network.SubResource{ID: probe.ID},
			BackendAddressPool:      &network.SubResource{ID: backendPoolID},
			FrontendIPConfiguration: &network.SubResource{ID: frontendIpID},
		}

		outRules = append(outRules, network.LoadBalancingRule{
			Name:       &ruleName,
			Properties: &props,
			ID:         existingRule.ID,
		})
	}

	return &outRules, nil
}

func pullOutProbes(d *schema.ResourceData) (*[]network.Probe, error) {
	log.Printf("[resourceArmSimpleLb] pullOutProbes[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutProbes[exit]")

	outProbes := []network.Probe{}
	if probes := d.Get("probe").(*schema.Set); probes.Len() > 0 {
		for _, probe := range probes.List() {
			probe := probe.(map[string]interface{})

			probeProps := network.ProbePropertiesFormat{
				Protocol:          network.ProbeProtocol(probe["protocol"].(string)),
				Port:              Int(probe["port"]),
				IntervalInSeconds: Int(probe["interval"]),
				NumberOfProbes:    Int(probe["number_of_probes"]),
			}
			if requestPath := probe["request_path"].(string); requestPath != "" {
				probeProps.RequestPath = &requestPath
			}

			outProbes = append(outProbes, network.Probe{
				Name:       String(probe["name"]),
				Properties: &probeProps,
			})
		}
	}

	return &outProbes, nil
}

func pullOutFrontEndIps(d *schema.ResourceData) (*[]network.FrontendIPConfiguration, error) {
	log.Printf("[resourceArmSimpleLb] pullOutFrontEndIps[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutFrontEndIps[exit]")

	returnRules := []network.FrontendIPConfiguration{}

	frontedIpName := fmt.Sprintf("%sfrontendip", d.Get("name").(string))
	frontedIpAllocationMethod := network.IPAllocationMethod(d.Get("frontend_allocation_method").(string))
	frontedIpSubnet := d.Get("frontend_subnet").(string)
	frontedIpPublicIpAddress := d.Get("frontend_public_ip_address").(string)
	frontedIpPrivateIpAddress := d.Get("frontend_private_ip_address").(string)

	if frontedIpSubnet == "" && frontedIpPublicIpAddress == "" {
		var logMsg = fmt.Sprintf("[ERROR] Either a subnet of a public ip address must be provided")
		log.Printf("[resourceArmSimpleLb] %s", logMsg)
		return nil, fmt.Errorf(logMsg)
	}

	if frontedIpPrivateIpAddress == "" && frontedIpAllocationMethod == network.Static {
		var logMsg = fmt.Sprintf("An private IP address must be provided if static allocation is used.")
		log.Printf("[resourceArmSimpleLb] %s", logMsg)
		return nil, fmt.Errorf(logMsg)
	}

	ipProps := network.FrontendIPConfigurationPropertiesFormat{
		PrivateIPAllocationMethod: frontedIpAllocationMethod}

	if frontedIpSubnet != "" {
		subnet := network.Subnet{ID: &frontedIpSubnet}
		ipProps.Subnet = &subnet
	}
	if frontedIpPublicIpAddress != "" {
		pubIp := network.PublicIPAddress{ID: &frontedIpPublicIpAddress}
		ipProps.PublicIPAddress = &pubIp
	}
	if frontedIpPrivateIpAddress != "" {
		ipProps.PrivateIPAddress = &frontedIpPrivateIpAddress
	}

	frontendIpConf := network.FrontendIPConfiguration{Name: &frontedIpName, Properties: &ipProps}
	returnRules = append(returnRules, frontendIpConf)
	return &returnRules, nil
}

func resourceArmSimpleLbCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbCreate[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbCreate[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	// first; fetch a bunch of fields:
	typ := d.Get("type").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGrp := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	loadBalancer := network.LoadBalancer{
		Name:       &name,
		Type:       &typ,
		Location:   &location,
		Properties: &network.LoadBalancerPropertiesFormat{},
		Tags:       expandTags(tags),
	}

	fipconfs, err := pullOutFrontEndIps(d)
	if err != nil {
		return err
	}
	loadBalancer.Properties.FrontendIPConfigurations = fipconfs
	probes, err := pullOutProbes(d)
	if err != nil {
		return err
	}
	loadBalancer.Properties.Probes = probes

	new_backend_pool_name := fmt.Sprintf("%sbackendpool", name)
	backendpool := network.BackendAddressPool{Name: &new_backend_pool_name}
	backendPoolConfs := []network.BackendAddressPool{}
	backendPoolConfs = append(backendPoolConfs, backendpool)
	loadBalancer.Properties.BackendAddressPools = &backendPoolConfs
	loadBalancer.Properties.LoadBalancingRules = &[]network.LoadBalancingRule{}

	resp, err := lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}
	log.Printf("[resourceArmSimpleLb] Create LB got status %d.  Provision State %s", resp.StatusCode, *resp.Properties.ProvisioningState)

	//Possible status values are Updating|Deleting|Failed|Succeeded
	if *resp.Properties.ProvisioningState != "Succeeded" {
		return fmt.Errorf("The load balancer was not properly deployed.  The provisioning state %s", *resp.Properties.ProvisioningState)
	}

	log.Printf("[resourceArmSimpleLb] We have the IDs now updating to set rules")
	loadBalancer.Properties.LoadBalancingRules, err = pullOutLbRules(d, resp)
	if err != nil {
		return err
	}
	log.Printf("[resourceArmSimpleLb] created %d rules", len(*loadBalancer.Properties.LoadBalancingRules))

	resp, err = lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR When trying to set the rules.  LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}
	log.Printf("[resourceArmSimpleLb] Set the rules on the LB.  Provision State %s", *resp.Properties.ProvisioningState)

	return flattenAllOfLb(resp, d, meta)
}

func flattenAzureRmFrontendIp(frontenIpArray []network.FrontendIPConfiguration, d *schema.ResourceData) error {
	log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[exit]")

	if len(frontenIpArray) < 1 {
		return nil
	}
	if len(frontenIpArray) > 1 {
		log.Printf("[WARN] More than 1 frontend ip was found.  The simpleLB resource will just use the first one.")
	}

	frontenIp := frontenIpArray[0]

	if frontenIp.Properties.PrivateIPAddress != nil {
		d.Set("frontend_private_ip_address", *frontenIp.Properties.PrivateIPAddress)
	}
	d.Set("frontend_allocation_method", string(frontenIp.Properties.PrivateIPAllocationMethod))
	if frontenIp.Properties.Subnet != nil {
		d.Set("frontend_subnet", *frontenIp.Properties.Subnet.ID)
	}
	if frontenIp.Properties.PublicIPAddress != nil {
		d.Set("frontend_public_ip_address", *frontenIp.Properties.PublicIPAddress.ID)
	}

	return nil
}

func flattenAzureRmLoadBalancerRules(loadBalancer network.LoadBalancer, d *schema.ResourceData) error {
	log.Printf("[resourceArmSimpleLb] flattenAzureRmLoadBalancerRules[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmLoadBalancerRules[exit]")

	loadBalancingRuleArray := loadBalancer.Properties.LoadBalancingRules

	ruleSet := &schema.Set{
		F: resourceARMLoadBalancerProbeHash,
	}

	log.Printf("[resourceArmSimpleLb] flattenAzureRmLoadBalancerRules found %d rules", len(*loadBalancingRuleArray))

	for _, rule := range *loadBalancingRuleArray {
		r := map[string]interface{}{}

		log.Printf("[resourceArmSimpleLb] Found LB RULE %s", *rule.Name)
		r["name"] = *rule.Name
		r["rule_id"] = *rule.ID
		if rule.Properties != nil {
			r["protocol"] = string(rule.Properties.Protocol)
			r["load_distribution"] = string(rule.Properties.LoadDistribution)
			r["frontend_port"] = *rule.Properties.FrontendPort
			r["backend_port"] = *rule.Properties.BackendPort

			ruleProbeID := *rule.Properties.Probe.ID
			conf, err := findProbeById(loadBalancer.Properties.Probes, ruleProbeID)
			if err != nil {
				return err
			}
			r["probe_name"] = *conf.Name
		}
		ruleSet.Add(r)
	}
	d.Set("rule", ruleSet)

	return nil
}

func flattenAzureRmProbe(probeArray *[]network.Probe, d *schema.ResourceData) error {
	log.Printf("[resourceArmSimpleLb] flattenAzureRmProbe[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmProbe[exit]")

	probeSet := &schema.Set{
		F: resourceARMLoadBalancerProbeHash,
	}

	for _, probe := range *probeArray {
		p := map[string]interface{}{}

		p["name"] = *probe.Name
		p["probe_id"] = *probe.ID
		if probe.Properties != nil {
			p["protocol"] = string(probe.Properties.Protocol)
			p["port"] = *probe.Properties.Port
			p["interval"] = *probe.Properties.IntervalInSeconds
			p["number_of_probes"] = *probe.Properties.NumberOfProbes
			if probe.Properties.RequestPath != nil {
				p["request_path"] = *probe.Properties.RequestPath
			}
		}

		probeSet.Add(p)
	}
	d.Set("probe", probeSet)

	return nil
}

func resourceArmSimpleLbUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbUpdate[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbUpdate[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	// first; fetch a bunch of fields:
	name := d.Get("name").(string)
	resGrp := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	loadBalancer, err := lbClient.Get(resGrp, name, "")
	if err != nil {
		return fmt.Errorf("Error could not find the LB %s.  %s", name, err)
	}

	loadBalancer.Tags = expandTags(tags)

	probes, err := pullOutProbes(d)
	if err != nil {
		return err
	}
	loadBalancer.Properties.Probes = probes

	new_backend_pool_name := fmt.Sprintf("%sbackendpool", name)
	backendpool := network.BackendAddressPool{Name: &new_backend_pool_name}
	backendPoolConfs := []network.BackendAddressPool{}
	backendPoolConfs = append(backendPoolConfs, backendpool)
	loadBalancer.Properties.BackendAddressPools = &backendPoolConfs
	loadBalancer.Properties.LoadBalancingRules = &[]network.LoadBalancingRule{}

	resp, err := lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR Update LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}
	log.Printf("[resourceArmSimpleLb] Update LB got status %d.  Provision State %s", resp.StatusCode, *resp.Properties.ProvisioningState)

	//Possible status values are Updating|Deleting|Failed|Succeeded
	if *resp.Properties.ProvisioningState != "Succeeded" {
		return fmt.Errorf("The load balancer was not properly deployed.  The provisioning state %s", *resp.Properties.ProvisioningState)
	}

	log.Printf("[resourceArmSimpleLb] We have the IDs now updating to set rules")
	loadBalancer.Properties.LoadBalancingRules, err = pullOutLbRules(d, resp)
	if err != nil {
		return err
	}
	log.Printf("[resourceArmSimpleLb] created %d rules", len(*loadBalancer.Properties.LoadBalancingRules))

	resp, err = lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR When trying to set the rules.  Update LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}
	log.Printf("[resourceArmSimpleLb] Set the rules on the LB.  Provision State %s", *resp.Properties.ProvisioningState)

	return flattenAllOfLb(resp, d, meta)
}

func resourceArmSimpleLbDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbDelete[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbDelete[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing deletion request to Azure ARM for load balancer '%s'.", name)

	resp, err := lbClient.Delete(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM delete request for load balancer '%s': %s", name, err)
	}

	log.Printf("[resourceArmSimpleLb] delete response %d %s", resp.StatusCode, resp.Status)

	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmSimpleLbRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	resGrp := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of load balancer '%s' off Azure.", name)

	loadBalancer, err := lbClient.Get(resGrp, name, "")
	if err != nil {
		return fmt.Errorf("Error reading the state of the load balancer off Azure: %s", err)
	}

	return flattenAllOfLb(loadBalancer, d, meta)
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func flattenAllOfLb(loadBalancer network.LoadBalancer, d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] flattenAllOfLb[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAllOfLb[exit]")

	log.Printf("[INFO] Succesfully retrieved details for load balancer '%s'.", *loadBalancer.Name)

	fip := loadBalancer.Properties.FrontendIPConfigurations

	d.Set("location", loadBalancer.Location)
	d.Set("type", loadBalancer.Type)

	err := flattenAzureRmFrontendIp(*fip, d)
	if err != nil {
		return err
	}
	err = flattenAzureRmProbe(loadBalancer.Properties.Probes, d)
	if err != nil {
		return err
	}
	if loadBalancer.Properties.BackendAddressPools == nil || len(*loadBalancer.Properties.BackendAddressPools) != 1 {
		return fmt.Errorf("There must be exactly 1 backend pool to use this resource")
	}
	d.Set("backend_pool_id", (*loadBalancer.Properties.BackendAddressPools)[0].ID)
	if loadBalancer.Properties.FrontendIPConfigurations == nil || len(*loadBalancer.Properties.FrontendIPConfigurations) != 1 {
		return fmt.Errorf("There must be exactly 1 fronted to use this resource")
	}
	d.Set("frontend_id", (*loadBalancer.Properties.FrontendIPConfigurations)[0].ID)
	err = flattenAzureRmLoadBalancerRules(loadBalancer, d)
	if err != nil {
		return err
	}
	flattenAndSetTags(d, loadBalancer.Tags)

	d.SetId(*loadBalancer.ID)

	return nil
}

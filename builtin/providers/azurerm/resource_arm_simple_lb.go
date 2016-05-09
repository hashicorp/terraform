package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
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
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: azureRMNormalizeLocation,
			},
			"frontend_private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"frontend_allocation_method": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAllocationMethod,
			},
			"frontend_subnet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"frontend_public_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"probe": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
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
				Optional: true,
				Computed: true,
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
	rule := fmt.Sprintf("%s-%s-%s%d-%d", m["name"].(string), m["protocol"].(string), m["load_distribution"].(string), m["frontend_port"], m["backend_port"])
	return hashcode.String(rule)
}

func resourceARMLoadBalancerProbeHash(v interface{}) int {
	m := v.(map[string]interface{})
	rule := fmt.Sprintf("%s-%s-%d-%d", m["name"].(string), m["protocol"].(string), m["frontend_port"], m["backend_port"])
	if m["request_path"] != nil {
		rule = rule + "-" + m["request_path"].(string)
	}
	return hashcode.String(rule)
}

func validateAllocationMethod(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"static":  true,
		"dynamic": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Allocation method can only be Static or Dynamic"))
	}
	return
}

func validateProtocolType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"tcp": true,
		"udp": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Protocol can only be tcp or udp"))
	}
	return
}

func validateProbeProtocolType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"tcp":  true,
		"http": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Protocol can only be tcp or udp"))
	}
	return
}

func validateLoadDistribution(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"default":          true,
		"sourceip":         true,
		"sourceipprotocol": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Load Distribution can only be default, sourceIp, or sourceIpProtocol"))
	}
	return
}

func findProbeConfByName(probeArray *[]network.Probe, probeName string) (int, error) {
	for i := 0; i < len(*probeArray); i++ {
		tmpProbe := (*probeArray)[i]
		if *tmpProbe.Name == probeName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Error loading the probe named %s", probeName)
}

func findProbeConfById(probeArray *[]network.Probe, probeId string) (int, error) {
	for i := 0; i < len(*probeArray); i++ {
		tmpProbe := (*probeArray)[i]
		if *tmpProbe.ID == probeId {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Error finding the probe ID %s", probeId)
}

func pullOutLbRules(d *schema.ResourceData, loadBalancer network.LoadBalancer) (*[]network.LoadBalancingRule, error) {
	log.Printf("[resourceArmSimpleLb] pullOutLbRules[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutLbRules[exit]")

	backendPoolId := d.Get("backend_pool_id").(string)
	frontendIpId := d.Get("frontend_id").(string)
	backendPoolRef := network.SubResource{ID: &backendPoolId}
	frontendIpRef := network.SubResource{ID: &frontendIpId}

	// then; the subnets:
	outRules := []network.LoadBalancingRule{}
	if rules := d.Get("rule").(*schema.Set); rules.Len() > 0 {
		for _, rule := range rules.List() {
			rule := rule.(map[string]interface{})

			ruleName := rule["name"].(string)
			ruleProtocol := rule["protocol"].(string)
			ruleProbeName := rule["probe_name"].(string)
			loadDistribution := rule["load_distribution"].(string)
			frontendPort := rule["frontend_port"].(int)
			backendendPort := rule["backend_port"].(int)

			i, err := findProbeConfByName(loadBalancer.Properties.Probes, ruleProbeName)
			if err != nil {
				return nil, err
			}

			probeRef := network.SubResource{ID: (*loadBalancer.Properties.Probes)[i].ID}

			props := network.LoadBalancingRulePropertiesFormat{
				Protocol:                network.TransportProtocol(ruleProtocol),
				LoadDistribution:        network.LoadDistribution(loadDistribution),
				FrontendPort:            &frontendPort,
				BackendPort:             &backendendPort,
				Probe:                   &probeRef,
				BackendAddressPool:      &backendPoolRef,
				FrontendIPConfiguration: &frontendIpRef,
			}
			ruleObj := network.LoadBalancingRule{
				Name:       &ruleName,
				Properties: &props,
			}
			outRules = append(outRules, ruleObj)
		}
	}

	return &outRules, nil
}

func pullOutProbes(d *schema.ResourceData) (*[]network.Probe, error) {
	log.Printf("[resourceArmSimpleLb] pullOutProbes[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutProbes[exit]")

	// then; the subnets:
	outProbes := []network.Probe{}
	if probes := d.Get("probe").(*schema.Set); probes.Len() > 0 {
		for _, probe := range probes.List() {
			probe := probe.(map[string]interface{})

			probeName := probe["name"].(string)
			ruleProtocol := probe["protocol"].(string)
			requestPath := probe["request_path"].(string)
			port := probe["port"].(int)
			interval := probe["interval"].(int)
			numberOfProbes := probe["number_of_probes"].(int)

			probeProps := network.ProbePropertiesFormat{
				Protocol:          network.ProbeProtocol(ruleProtocol),
				Port:              &port,
				IntervalInSeconds: &interval,
				NumberOfProbes:    &numberOfProbes,
			}
			if requestPath != "" {
				probeProps.RequestPath = &requestPath
			}

			probeObj := network.Probe{
				Name:       &probeName,
				Properties: &probeProps,
			}
			outProbes = append(outProbes, probeObj)
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

	resp, err := lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}
	log.Printf("[resourceArmSimpleLb] Create LB got status %d", resp.StatusCode)

	d.SetId(*resp.ID)
	err = iResourceArmSimpleLbRead(d, meta)
	if err != nil {
		return err
	}

	log.Printf("[resourceArmSimpleLb] We have the IDs now updating to set rules")
	loadBalancer.Properties.LoadBalancingRules, err = pullOutLbRules(d, resp)
	if err != nil {
		return err
	}
	resp, err = lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmSimpleLb] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}

	return iResourceArmSimpleLbRead(d, meta)
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
	log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[exit]")

	loadBalancingRuleArray := loadBalancer.Properties.LoadBalancingRules

	ruleSet := &schema.Set{
		F: resourceARMLoadBalancerProbeHash,
	}

	for _, rule := range *loadBalancingRuleArray {
		r := map[string]interface{}{}

		r["name"] = *rule.Name
		r["rule_id"] = *rule.ID
		if rule.Properties != nil {
			r["protocol"] = string(rule.Properties.Protocol)
			r["load_distribution"] = string(rule.Properties.LoadDistribution)
			r["frontend_port"] = *rule.Properties.FrontendPort
			r["backend_port"] = *rule.Properties.BackendPort

			ruleProbeID := *rule.Properties.Probe.ID
			i, err := findProbeConfById(loadBalancer.Properties.Probes, ruleProbeID)
			if err != nil {
				return err
			}
			r["probe_name"] = *(*loadBalancer.Properties.Probes)[i].Name
		}
		ruleSet.Add(r)
	}
	d.Set("subnet", ruleSet)

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
	d.Set("subnet", probeSet)

	return nil
}

func resourceArmSimpleLbUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbUpdate[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbUpdate[exit]")

	return resourceArmSimpleLbCreate(d, meta)
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
func iResourceArmSimpleLbRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] iResourceArmSimpleLbRead[enter]")
	defer log.Printf("[resourceArmSimpleLb] iResourceArmSimpleLbRead[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	resGrp := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of load balancer '%s' off Azure.", name)

	loadBalancer, err := lbClient.Get(resGrp, name, "")
	if err != nil {
		return fmt.Errorf("Error reading the state of the load balancer off Azure: %s", err)
	}

	log.Printf("[INFO] Succesfully retrieved details for load balancer '%s'.", *loadBalancer.Name)

	fip := loadBalancer.Properties.FrontendIPConfigurations

	d.Set("location", loadBalancer.Location)
	d.Set("type", loadBalancer.Type)

	err = flattenAzureRmFrontendIp(*fip, d)
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

	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmSimpleLbRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[exit]")

	return iResourceArmSimpleLbRead(d, meta)
}

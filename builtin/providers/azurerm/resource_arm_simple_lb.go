package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
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

			"probe_id": &schema.Schema{
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
			"probe_protocol": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateProtocolType,
			},
			"probe_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"probe_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"probe_number_of_probes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"probe_request_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"rule_protocol": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateProtocolType,
			},
			"rule_load_distribution": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateLoadDistribution,
			},
			"rule_frontend_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"rule_backend_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
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

func pullOutLbRules(d *schema.ResourceData) (*[]network.LoadBalancingRule, error) {
	log.Printf("[resourceArmSimpleLb] pullOutLbRules[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutLbRules[exit]")

	backendPoolId := d.Get("backend_pool_id").(string)
	frontendIpId := d.Get("frontend_id").(string)
	probeId := d.Get("probe_id").(string)

	backendPoolRef := network.SubResource{ID: &backendPoolId}
	frontendIpRef := network.SubResource{ID: &frontendIpId}
	probeRef := network.SubResource{ID: &probeId}

	returnRules := []network.LoadBalancingRule{}

	ruleName := fmt.Sprintf("%srule", d.Get("name").(string))
	ruleProtocol := network.TransportProtocol(d.Get("rule_protocol").(string))
	ruleFrontendPort := d.Get("rule_frontend_port").(int)
	ruleBackendPort := d.Get("rule_backend_port").(int)
	ruleLoadDistributionS := d.Get("rule_load_distribution").(string)
	if ruleLoadDistributionS == "" {
		ruleLoadDistributionS = "Default"
	}

	rulesProps := network.LoadBalancingRulePropertiesFormat{
		FrontendIPConfiguration: &frontendIpRef,
		BackendAddressPool:      &backendPoolRef,
		BackendPort:             &ruleBackendPort,
		FrontendPort:            &ruleFrontendPort,
		Protocol:                ruleProtocol,
		LoadDistribution:        network.LoadDistribution(ruleLoadDistributionS),
		Probe:                   &probeRef,
	}

	ruleType := network.LoadBalancingRule{
		Name:       &ruleName,
		Properties: &rulesProps,
	}

	returnRules = append(returnRules, ruleType)

	return &returnRules, nil
}

func pullOutProbes(d *schema.ResourceData) (*[]network.Probe, error) {
	log.Printf("[resourceArmSimpleLb] pullOutProbes[enter]")
	defer log.Printf("[resourceArmSimpleLb] pullOutProbes[exit]")

	returnRules := []network.Probe{}

	probeName := fmt.Sprintf("%sprobe", d.Get("name").(string))

	probeProtocol := network.ProbeProtocol(d.Get("probe_protocol").(string))
	probePort := d.Get("probe_port").(int)
	probeInterval := d.Get("probe_interval").(int)
	probeNumberOfProbes := d.Get("probe_number_of_probes").(int)
	probeRequestPath := d.Get("probe_request_path").(string)

	probeProps := network.ProbePropertiesFormat{
		Protocol:          probeProtocol,
		Port:              &probePort,
		IntervalInSeconds: &probeInterval,
		NumberOfProbes:    &probeNumberOfProbes,
	}
	if probeRequestPath != "" {
		probeProps.RequestPath = &probeRequestPath
	}
	probe := network.Probe{
		Name:       &probeName,
		Properties: &probeProps,
	}

	returnRules = append(returnRules, probe)
	return &returnRules, nil
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

	loadBalancer := network.LoadBalancer{
		Name:       &name,
		Type:       &typ,
		Location:   &location,
		Properties: &network.LoadBalancerPropertiesFormat{},
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
	loadBalancer.Properties.LoadBalancingRules, err = pullOutLbRules(d)
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

func flattenAzureRmLoadBalancerRules(loadBalancingRuleArray []network.LoadBalancingRule, d *schema.ResourceData) error {
	log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmFrontendIp[exit]")

	if len(loadBalancingRuleArray) < 1 {
		return nil
	}
	if len(loadBalancingRuleArray) > 1 {
		log.Printf("[WARN] More than 1 load balancing rule was found.  The simpleLB resource will just use the first one.")
	}

	loadBalancingRule := loadBalancingRuleArray[0]
	d.Set("rule_protocol", string(loadBalancingRule.Properties.Protocol))
	d.Set("rule_load_distribution", string(loadBalancingRule.Properties.LoadDistribution))
	d.Set("rule_frontend_port", *loadBalancingRule.Properties.FrontendPort)
	d.Set("rule_backend_port", *loadBalancingRule.Properties.BackendPort)

	return nil
}

func flattenAzureRmProbe(probeArray []network.Probe, d *schema.ResourceData) error {
	log.Printf("[resourceArmSimpleLb] flattenAzureRmProbe[enter]")
	defer log.Printf("[resourceArmSimpleLb] flattenAzureRmProbe[exit]")

	if len(probeArray) < 1 {
		return nil
	}
	if len(probeArray) > 1 {
		log.Printf("[WARN] More than 1 load balancing rule was found.  The simpleLB resource will just use the first one.")
	}

	probe := probeArray[0]

	d.Set("probe_protocol", string(probe.Properties.Protocol))
	d.Set("probe_port", *probe.Properties.Port)
	d.Set("probe_interval", *probe.Properties.IntervalInSeconds)
	d.Set("probe_number_of_probes", *probe.Properties.NumberOfProbes)
	if probe.Properties.RequestPath != nil {
		d.Set("probe_request_path", *probe.Properties.RequestPath)
	}

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
	err = flattenAzureRmProbe(*loadBalancer.Properties.Probes, d)
	if err != nil {
		return err
	}
	if loadBalancer.Properties.BackendAddressPools == nil || len(*loadBalancer.Properties.BackendAddressPools) != 1 {
		return fmt.Errorf("There must be exactly 1 backend pool to use this resource")
	}
	d.Set("backend_pool_id", (*loadBalancer.Properties.BackendAddressPools)[0].ID)
	if loadBalancer.Properties.Probes == nil || len(*loadBalancer.Properties.Probes) != 1 {
		return fmt.Errorf("There must be exactly 1 probe to use this resource")
	}
	d.Set("probe_id", (*loadBalancer.Properties.Probes)[0].ID)
	if loadBalancer.Properties.FrontendIPConfigurations == nil || len(*loadBalancer.Properties.FrontendIPConfigurations) != 1 {
		return fmt.Errorf("There must be exactly 1 probe to use this resource")
	}
	d.Set("frontend_id", (*loadBalancer.Properties.FrontendIPConfigurations)[0].ID)
	err = flattenAzureRmLoadBalancerRules(*loadBalancer.Properties.LoadBalancingRules, d)
	if err != nil {
		return err
	}
	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmSimpleLbRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmSimpleLbRead[exit]")

	return iResourceArmSimpleLbRead(d, meta)
}

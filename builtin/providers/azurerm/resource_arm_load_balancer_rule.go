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
func resourceArmLbRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRuleCreate,
		Read:   resourceArmLbRuleRead,
		Update: resourceArmLbRuleUpdate,
		Delete: resourceArmLbRuleDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"protocol": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateProtocolType,
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

			"frontend_ip_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"backend_pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"probe_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
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

func resourceArmRuleCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLbRule] resourceArmRuleCreate[enter]")
	defer log.Printf("[resourceArmLbRule] resourceArmRuleCreate[exit]")

	// first; fetch a bunch of fields:
	ruleName := d.Get("name").(string)
	protocol := d.Get("protocol").(string)
	frontendPort := d.Get("frontend_port").(int)
	backendPort := d.Get("backend_port").(int)
	loadDistribution := d.Get("load_distribution").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)
	frontendIpId := d.Get("frontend_ip_id").(string)
	backendPoolId := d.Get("backend_pool_id").(string)
	probeId := d.Get("probe_id").(string)

	lbClient := meta.(*ArmClient).loadBalancerClient
	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}

	frontendIpRef := network.SubResource{ID: &frontendIpId}
	backendPoolRef := network.SubResource{ID: &backendPoolId}
	probeIdRef := network.SubResource{ID: &probeId}

	ruleProps := network.LoadBalancingRulePropertiesFormat{
		FrontendPort:            &frontendPort,
		BackendPort:             &backendPort,
		LoadDistribution:        network.LoadDistribution(loadDistribution),
		Protocol:                network.TransportProtocol(protocol),
		FrontendIPConfiguration: &frontendIpRef,
		BackendAddressPool:      &backendPoolRef,
		Probe:                   &probeIdRef,
	}

	ruleStruct := network.LoadBalancingRule{Name: &ruleName, Properties: &ruleProps}
	i, err := findRuleConf(loadBalancer.Properties.LoadBalancingRules, ruleName)
	if err == nil {
		// If one by that name exists update it
		(*loadBalancer.Properties.LoadBalancingRules)[i] = ruleStruct
	} else {
		rulesArray := append(*loadBalancer.Properties.LoadBalancingRules, ruleStruct)
		loadBalancer.Properties.LoadBalancingRules = &rulesArray
	}

	loadBalancer, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmLbRule] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for rule '%s': %s", ruleName, err)
	}
	i, err = findRuleConf(loadBalancer.Properties.LoadBalancingRules, ruleName)
	if err != nil {
		return err
	}

	ruleOut := (*loadBalancer.Properties.LoadBalancingRules)[i]
	log.Printf("[resourceArmLbRule] Created the rule named % with ID %s", *ruleOut.Name, *ruleOut.ID)

	d.SetId(*ruleOut.ID)

	return nil
}

func resourceArmLbRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLbRule] resourceArmLbRuleUpdate[enter]")
	defer log.Printf("[resourceArmLbRule] resourceArmLbRuleUpdate[exit]")

	return resourceArmRuleCreate(d, meta)
}

func resourceArmLbRuleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLbRule] resourceLbRuleDelete[enter]")
	defer log.Printf("[resourceArmLbRule] resourceLbRuleDelete[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	ruleName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findRuleConf(loadBalancer.Properties.LoadBalancingRules, ruleName)
	if err != nil {
		return err
	}
	ruleA := append((*loadBalancer.Properties.LoadBalancingRules)[:i], (*loadBalancer.Properties.LoadBalancingRules)[i+1:]...)
	loadBalancer.Properties.LoadBalancingRules = &ruleA
	_, err = lbClient.CreateOrUpdate(resourceGroupName, loadBalancerName, loadBalancer)
	if err != nil {
		return err
	}
	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmLbRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLbRule] resourceLbRuleRead[enter]")
	defer log.Printf("[resourceArmLbRule] resourceLbRuleRead[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	ruleName := d.Get("name").(string)
	loadBalancerName := d.Get("load_balancer_name").(string)
	resourceGroupName := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of rule '%s'.", ruleName)

	loadBalancer, err := lbClient.Get(resourceGroupName, loadBalancerName, "")
	if err != nil {
		return err
	}
	i, err := findRuleConf(loadBalancer.Properties.LoadBalancingRules, ruleName)
	if err != nil {
		return err
	}

	ruleStruct := (*loadBalancer.Properties.LoadBalancingRules)[i]

	d.Set("frontend_port", *ruleStruct.Properties.FrontendPort)
	d.Set("backend_port", *ruleStruct.Properties.BackendPort)
	d.Set("load_distribution", string(ruleStruct.Properties.LoadDistribution))
	d.Set("protocol", string(ruleStruct.Properties.Protocol))
	d.SetId(*ruleStruct.ID)
	return nil
}

func findRuleConf(ruleArray *[]network.LoadBalancingRule, ruleName string) (int, error) {
	// Find the correct LB
	for i := 0; i < len(*ruleArray); i++ {
		tmpRule := (*ruleArray)[i]
		if *tmpRule.Name == ruleName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Error loading the rule named %s", ruleName)
}

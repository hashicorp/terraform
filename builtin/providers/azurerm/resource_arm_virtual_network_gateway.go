package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmVirtualNetworkGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualNetworkGatewayCreate,
		Read:   resourceArmVirtualNetworkGatewayRead,
		Update: resourceArmVirtualNetworkGatewayCreate,
		Delete: resourceArmVirtualNetworkGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(network.VirtualNetworkGatewayTypeExpressRoute),
					string(network.VirtualNetworkGatewayTypeVpn),
				}, false),
				ForceNew: true,
			},

			"vpn_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(network.RouteBased),
					string(network.PolicyBased),
				}, false),
				ForceNew: true,
			},

			"enable_bgp": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"active_active": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"sku": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.VirtualNetworkGatewaySkuNameBasic),
								string(network.VirtualNetworkGatewaySkuNameStandard),
								string(network.VirtualNetworkGatewaySkuNameHighPerformance),
								string(network.VirtualNetworkGatewaySkuNameUltraPerformance),
							}, false),
						},
						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.VirtualNetworkGatewaySkuTierBasic),
								string(network.VirtualNetworkGatewaySkuTierStandard),
								string(network.VirtualNetworkGatewaySkuTierHighPerformance),
								string(network.VirtualNetworkGatewaySkuTierUltraPerformance),
							}, false),
						},
						"capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
				Set: hashVirtualNetworkGatewaySku,
			},

			"ip_configuration": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 2,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "vnetGatewayConfig",
						},
						"private_ip_address_allocation": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Static),
								string(network.Dynamic),
							}, false),
							Default: string(network.Dynamic),
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"public_ip_address_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"vpn_client_configuration": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address_space": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"root_certificate": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"public_cert_data": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: hashVirtualNetworkGatewayRootCert,
						},
						"revoked_certificate": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"thumbprint": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: hashVirtualNetworkGatewayRevokedCert,
						},
					},
				},
				Set: hashVirtualNetworkGatewayVpnClientConfig,
			},

			"bgp_settings": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"asn": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"peering_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"peer_weight": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				Set: hashVirtualNetworkGatewayBgpSettings,
			},

			"default_local_network_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmVirtualNetworkGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vnetGatewayClient := client.vnetGatewayClient

	log.Printf("[INFO] preparing arguments for Azure ARM Virtual Network Gateway creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	gateway := network.VirtualNetworkGateway{
		Name:     &name,
		Location: &location,
		Tags:     expandTags(tags),
		VirtualNetworkGatewayPropertiesFormat: getArmVirtualNetworkGatewayProperties(d),
	}

	_, err := vnetGatewayClient.CreateOrUpdate(resGroup, name, gateway, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := vnetGatewayClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read VirtualNetworkGateway %s (resource group %s) ID", name, resGroup)
	}

	log.Printf("[DEBUG] Waiting for VirtualNetworkGateway (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: virtualNetworkGatewayStateRefreshFunc(client, resGroup, name, false),
		Timeout: 60 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for VirtualNetworkGateway (%s) to become available: %s", name, err)
	}

	d.SetId(*read.ID)

	return resourceArmVirtualNetworkGatewayRead(d, meta)
}

func resourceArmVirtualNetworkGatewayRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).vnetGatewayClient

	resGroup, name, err := resourceGroupAndVirtualNetworkGatewayFromId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on VirtualNetwork Gateway %s: %s", name, err)
	}

	gw := *resp.VirtualNetworkGatewayPropertiesFormat

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("type", string(gw.GatewayType))
	d.Set("enable_bgp", gw.EnableBgp)
	d.Set("active_active", gw.ActiveActive)
	d.Set("sku", schema.NewSet(hashVirtualNetworkGatewaySku, flattenArmVirtualNetworkGatewaySku(gw.Sku)))

	if string(gw.VpnType) != "" {
		d.Set("vpn_type", string(gw.VpnType))
	}

	if gw.GatewayDefaultSite != nil {
		d.Set("default_local_network_gateway_id", gw.GatewayDefaultSite.ID)
	}

	d.Set("ip_configuration", flattenArmVirtualNetworkGatewayIPConfigurations(gw.IPConfigurations))

	if gw.VpnClientConfiguration != nil {
		vpnConfigFlat := flattenArmVirtualNetworkGatewayVpnClientConfig(gw.VpnClientConfiguration)
		d.Set("vpn_client_configuration", schema.NewSet(hashVirtualNetworkGatewayVpnClientConfig, vpnConfigFlat))
	}

	if gw.BgpSettings != nil {
		bgpSettingsFlat := flattenArmVirtualNetworkGatewayBgpSettings(gw.BgpSettings)
		d.Set("bgp_settings", schema.NewSet(hashVirtualNetworkGatewayBgpSettings, bgpSettingsFlat))
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmVirtualNetworkGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vnetGatewayClient := client.vnetGatewayClient

	resGroup, name, err := resourceGroupAndVirtualNetworkGatewayFromId(d.Id())
	if err != nil {
		return err
	}

	_, err = vnetGatewayClient.Delete(resGroup, name, make(chan struct{}))
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for VirtualNetworkGateway (%s) to be removed", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Deleting"},
		Target:  []string{"NotFound"},
		Refresh: virtualNetworkGatewayStateRefreshFunc(client, resGroup, name, true),
		Timeout: 15 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for VirtualNetworkGateway (%s) to be removed: %s", name, err)
	}

	// Gateways are not fully cleaned up when the API indicates the delete operation
	// has finished, this workaround was suggested by Azure support to avoid conflicts
	// when modifying/deleting the related subnet or network. Although the API indicated
	// that the Virtual Network Gateway does not exist anymore, there is still a link
	// to the Gateway Subnet it has been associated with. This causes an error when
	// trying to delete the Gateway Subnet immediately after the Virtual Network Gateway
	// has been deleted.

	d.SetId("")
	return nil
}

func virtualNetworkGatewayStateRefreshFunc(client *ArmClient, resourceGroupName string, virtualNetworkGateway string, withNotFound bool) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.vnetGatewayClient.Get(resourceGroupName, virtualNetworkGateway)
		if err != nil {
			if withNotFound && res.StatusCode == http.StatusNotFound {
				return res, "NotFound", nil
			}
			return nil, "", fmt.Errorf("Error issuing read request in virtualNetworkGatewayStateRefreshFunc to Azure ARM for VirtualNetworkGateway '%s' (RG: '%s'): %s", virtualNetworkGateway, resourceGroupName, err)
		}

		return res, *res.VirtualNetworkGatewayPropertiesFormat.ProvisioningState, nil
	}
}

func getArmVirtualNetworkGatewayProperties(d *schema.ResourceData) *network.VirtualNetworkGatewayPropertiesFormat {
	gatewayType := network.VirtualNetworkGatewayType(d.Get("type").(string))
	vpnType := network.VpnType(d.Get("vpn_type").(string))
	enableBgp := d.Get("enable_bgp").(bool)
	activeActive := d.Get("active_active").(bool)

	props := &network.VirtualNetworkGatewayPropertiesFormat{
		GatewayType:      gatewayType,
		VpnType:          vpnType,
		EnableBgp:        &enableBgp,
		ActiveActive:     &activeActive,
		Sku:              expandArmVirtualNetworkGatewaySku(d),
		IPConfigurations: expandArmVirtualNetworkGatewayIPConfigurations(d),
	}

	if gatewayDefaultSiteID := d.Get("default_local_network_gateway_id").(string); gatewayDefaultSiteID != "" {
		props.GatewayDefaultSite = &network.SubResource{
			ID: &gatewayDefaultSiteID,
		}
	}

	if _, ok := d.GetOk("vpn_client_configuration"); ok {
		props.VpnClientConfiguration = expandArmVirtualNetworkGatewayVpnClientConfig(d)
	}

	if _, ok := d.GetOk("bgp_settings"); ok {
		props.BgpSettings = expandArmVirtualNetworkGatewayBgpSettings(d)
	}

	return props
}

func expandArmVirtualNetworkGatewayBgpSettings(d *schema.ResourceData) *network.BgpSettings {
	bgpSets := d.Get("bgp_settings").(*schema.Set).List()
	bgp := bgpSets[0].(map[string]interface{})

	asn := int64(bgp["asn"].(int))
	peeringAddress := bgp["peering_address"].(string)
	peerWeight := int32(bgp["peer_weight"].(int))

	return &network.BgpSettings{
		Asn:               &asn,
		BgpPeeringAddress: &peeringAddress,
		PeerWeight:        &peerWeight,
	}
}

func expandArmVirtualNetworkGatewayIPConfigurations(d *schema.ResourceData) *[]network.VirtualNetworkGatewayIPConfiguration {
	configs := d.Get("ip_configuration").([]interface{})
	ipConfigs := make([]network.VirtualNetworkGatewayIPConfiguration, 0, len(configs))

	for _, c := range configs {
		conf := c.(map[string]interface{})

		name := conf["name"].(string)
		privateIPAllocMethod := network.IPAllocationMethod(conf["private_ip_address_allocation"].(string))

		props := &network.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: privateIPAllocMethod,
		}

		if subnetID := conf["subnet_id"].(string); subnetID != "" {
			props.Subnet = &network.SubResource{
				ID: &subnetID,
			}
		}

		if publicIP := conf["public_ip_address_id"].(string); publicIP != "" {
			props.PublicIPAddress = &network.SubResource{
				ID: &publicIP,
			}
		}

		ipConfig := network.VirtualNetworkGatewayIPConfiguration{
			Name: &name,
			VirtualNetworkGatewayIPConfigurationPropertiesFormat: props,
		}

		ipConfigs = append(ipConfigs, ipConfig)
	}

	return &ipConfigs
}

func expandArmVirtualNetworkGatewayVpnClientConfig(d *schema.ResourceData) *network.VpnClientConfiguration {
	configSets := d.Get("vpn_client_configuration").(*schema.Set).List()
	conf := configSets[0].(map[string]interface{})

	confAddresses := conf["address_space"].([]interface{})
	addresses := make([]string, 0, len(confAddresses))
	for _, addr := range confAddresses {
		addresses = append(addresses, addr.(string))
	}

	var rootCerts []network.VpnClientRootCertificate
	for _, rootCertSet := range conf["root_certificate"].(*schema.Set).List() {
		rootCert := rootCertSet.(map[string]interface{})
		name := rootCert["name"].(string)
		publicCertData := rootCert["public_cert_data"].(string)
		r := network.VpnClientRootCertificate{
			Name: &name,
			VpnClientRootCertificatePropertiesFormat: &network.VpnClientRootCertificatePropertiesFormat{
				PublicCertData: &publicCertData,
			},
		}
		rootCerts = append(rootCerts, r)
	}

	var revokedCerts []network.VpnClientRevokedCertificate
	for _, revokedCertSet := range conf["revoked_certificate"].(*schema.Set).List() {
		revokedCert := revokedCertSet.(map[string]interface{})
		name := revokedCert["name"].(string)
		thumbprint := revokedCert["thumbprint"].(string)
		r := network.VpnClientRevokedCertificate{
			Name: &name,
			VpnClientRevokedCertificatePropertiesFormat: &network.VpnClientRevokedCertificatePropertiesFormat{
				Thumbprint: &thumbprint,
			},
		}
		revokedCerts = append(revokedCerts, r)
	}

	return &network.VpnClientConfiguration{
		VpnClientAddressPool: &network.AddressSpace{
			AddressPrefixes: &addresses,
		},
		VpnClientRootCertificates:    &rootCerts,
		VpnClientRevokedCertificates: &revokedCerts,
	}
}

func expandArmVirtualNetworkGatewaySku(d *schema.ResourceData) *network.VirtualNetworkGatewaySku {
	skuSets := d.Get("sku").(*schema.Set).List()
	sku := skuSets[0].(map[string]interface{})

	name := sku["name"].(string)
	tier := sku["tier"].(string)

	return &network.VirtualNetworkGatewaySku{
		Name: network.VirtualNetworkGatewaySkuName(name),
		Tier: network.VirtualNetworkGatewaySkuTier(tier),
	}
}

func flattenArmVirtualNetworkGatewayBgpSettings(settings *network.BgpSettings) []interface{} {
	flat := make(map[string]interface{})

	flat["asn"] = int(*settings.Asn)
	flat["peering_address"] = *settings.BgpPeeringAddress
	flat["peer_weight"] = int(*settings.PeerWeight)

	return []interface{}{flat}
}

func flattenArmVirtualNetworkGatewayIPConfigurations(ipConfigs *[]network.VirtualNetworkGatewayIPConfiguration) []interface{} {
	flat := make([]interface{}, 0, len(*ipConfigs))

	for _, cfg := range *ipConfigs {
		props := cfg.VirtualNetworkGatewayIPConfigurationPropertiesFormat
		v := make(map[string]interface{})

		v["name"] = *cfg.Name
		v["private_ip_address_allocation"] = string(props.PrivateIPAllocationMethod)
		v["subnet_id"] = *props.Subnet.ID
		v["public_ip_address_id"] = *props.PublicIPAddress.ID

		flat = append(flat, v)
	}

	return flat
}

func flattenArmVirtualNetworkGatewayVpnClientConfig(cfg *network.VpnClientConfiguration) []interface{} {
	flat := make(map[string]interface{})

	addressSpace := make([]interface{}, 0, len(*cfg.VpnClientAddressPool.AddressPrefixes))
	for _, addr := range *cfg.VpnClientAddressPool.AddressPrefixes {
		addressSpace = append(addressSpace, addr)
	}
	flat["address_space"] = addressSpace

	rootCerts := make([]interface{}, 0, len(*cfg.VpnClientRootCertificates))
	for _, cert := range *cfg.VpnClientRootCertificates {
		v := map[string]interface{}{
			"name":             *cert.Name,
			"public_cert_data": *cert.VpnClientRootCertificatePropertiesFormat.PublicCertData,
		}
		rootCerts = append(rootCerts, v)
	}
	flat["root_certificate"] = schema.NewSet(hashVirtualNetworkGatewayRootCert, rootCerts)

	revokedCerts := make([]interface{}, 0, len(*cfg.VpnClientRevokedCertificates))
	for _, cert := range *cfg.VpnClientRevokedCertificates {
		v := map[string]interface{}{
			"name":       *cert.Name,
			"thumbprint": *cert.VpnClientRevokedCertificatePropertiesFormat.Thumbprint,
		}
		revokedCerts = append(revokedCerts, v)
	}
	flat["revoked_certificate"] = schema.NewSet(hashVirtualNetworkGatewayRevokedCert, revokedCerts)

	return []interface{}{flat}
}

func flattenArmVirtualNetworkGatewaySku(sku *network.VirtualNetworkGatewaySku) []interface{} {
	flat := make(map[string]interface{})

	flat["name"] = string(sku.Name)
	flat["tier"] = string(sku.Tier)
	flat["capacity"] = int(*sku.Capacity)

	return []interface{}{flat}
}

func hashVirtualNetworkGatewayBgpSettings(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["asn"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["peering_address"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["peer_weight"].(int)))

	return hashcode.String(buf.String())
}

func hashVirtualNetworkGatewayRootCert(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["public_cert_data"].(string)))

	return hashcode.String(buf.String())
}

func hashVirtualNetworkGatewayRevokedCert(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["thumbprint"].(string)))

	return hashcode.String(buf.String())
}

func hashVirtualNetworkGatewayVpnClientConfig(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	addressList := m["address_space"].([]interface{})
	for _, a := range addressList {
		buf.WriteString(fmt.Sprintf("%s-", a.(string)))
	}

	rootCerts := m["root_certificate"].(*schema.Set)
	for _, cert := range rootCerts.List() {
		buf.WriteString(fmt.Sprintf("%d-", rootCerts.F(cert)))
	}

	if m["revoked_certificate"] != nil {
		revokedCerts := m["revoked_certificate"].(*schema.Set)
		for _, cert := range revokedCerts.List() {
			buf.WriteString(fmt.Sprintf("%d-", revokedCerts.F(cert)))
		}
	}

	return hashcode.String(buf.String())
}

func hashVirtualNetworkGatewaySku(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["tier"].(string)))

	return hashcode.String(buf.String())
}

func resourceGroupAndVirtualNetworkGatewayFromId(virtualNetworkGatewayId string) (string, string, error) {
	id, err := parseAzureResourceID(virtualNetworkGatewayId)
	if err != nil {
		return "", "", err
	}
	name := id.Path["virtualNetworkGateways"]
	resGroup := id.ResourceGroup

	return resGroup, name, nil
}

func retrieveVirtualNetworkGatewayById(virtualNetworkGatewayId string, meta interface{}) (*network.VirtualNetworkGateway, bool, error) {
	vnetGatewayClient := meta.(*ArmClient).vnetGatewayClient

	resGroup, name, err := resourceGroupAndVirtualNetworkGatewayFromId(virtualNetworkGatewayId)
	if err != nil {
		return nil, false, errwrap.Wrapf("Error Getting VirtualNetworkGateway Name and Group: {{err}}", err)
	}

	resp, err := vnetGatewayClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Error making Read request on Azure VirtualNetworkGateway %s: %s", name, err)
	}

	return &resp, true, nil
}

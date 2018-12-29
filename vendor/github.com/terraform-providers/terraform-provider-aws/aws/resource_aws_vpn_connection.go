package aws

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type XmlVpnConnectionConfig struct {
	Tunnels []XmlIpsecTunnel `xml:"ipsec_tunnel"`
}

type XmlIpsecTunnel struct {
	OutsideAddress   string `xml:"vpn_gateway>tunnel_outside_address>ip_address"`
	BGPASN           string `xml:"vpn_gateway>bgp>asn"`
	BGPHoldTime      int    `xml:"vpn_gateway>bgp>hold_time"`
	PreSharedKey     string `xml:"ike>pre_shared_key"`
	CgwInsideAddress string `xml:"customer_gateway>tunnel_inside_address>ip_address"`
	VgwInsideAddress string `xml:"vpn_gateway>tunnel_inside_address>ip_address"`
}

type TunnelInfo struct {
	Tunnel1Address          string
	Tunnel1CgwInsideAddress string
	Tunnel1VgwInsideAddress string
	Tunnel1PreSharedKey     string
	Tunnel1BGPASN           string
	Tunnel1BGPHoldTime      int
	Tunnel2Address          string
	Tunnel2CgwInsideAddress string
	Tunnel2VgwInsideAddress string
	Tunnel2PreSharedKey     string
	Tunnel2BGPASN           string
	Tunnel2BGPHoldTime      int
}

func (slice XmlVpnConnectionConfig) Len() int {
	return len(slice.Tunnels)
}

func (slice XmlVpnConnectionConfig) Less(i, j int) bool {
	return slice.Tunnels[i].OutsideAddress < slice.Tunnels[j].OutsideAddress
}

func (slice XmlVpnConnectionConfig) Swap(i, j int) {
	slice.Tunnels[i], slice.Tunnels[j] = slice.Tunnels[j], slice.Tunnels[i]
}

func resourceAwsVpnConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpnConnectionCreate,
		Read:   resourceAwsVpnConnectionRead,
		Update: resourceAwsVpnConnectionUpdate,
		Delete: resourceAwsVpnConnectionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpn_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"customer_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"static_routes_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"tunnel1_inside_cidr": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateVpnConnectionTunnelInsideCIDR,
			},

			"tunnel1_preshared_key": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateVpnConnectionTunnelPreSharedKey,
			},

			"tunnel2_inside_cidr": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateVpnConnectionTunnelInsideCIDR,
			},

			"tunnel2_preshared_key": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateVpnConnectionTunnelPreSharedKey,
			},

			"tags": tagsSchema(),

			// Begin read only attributes
			"customer_gateway_configuration": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"tunnel1_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel1_cgw_inside_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel1_vgw_inside_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel1_bgp_asn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel1_bgp_holdtime": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"tunnel2_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel2_cgw_inside_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel2_vgw_inside_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel2_bgp_asn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tunnel2_bgp_holdtime": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"routes": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_cidr_block": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"source": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"state": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["destination_cidr_block"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["source"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["state"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"vgw_telemetry": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"accepted_route_count": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
						},

						"last_status_change": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"outside_ip_address": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"status": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"status_message": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["outside_ip_address"].(string)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsVpnConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Fill the tunnel options for the EC2 API
	options := []*ec2.VpnTunnelOptionsSpecification{
		{}, {},
	}

	if v, ok := d.GetOk("tunnel1_inside_cidr"); ok {
		options[0].TunnelInsideCidr = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tunnel2_inside_cidr"); ok {
		options[1].TunnelInsideCidr = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tunnel1_preshared_key"); ok {
		options[0].PreSharedKey = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tunnel2_preshared_key"); ok {
		options[1].PreSharedKey = aws.String(v.(string))
	}

	connectOpts := &ec2.VpnConnectionOptionsSpecification{
		StaticRoutesOnly: aws.Bool(d.Get("static_routes_only").(bool)),
		TunnelOptions:    options,
	}

	createOpts := &ec2.CreateVpnConnectionInput{
		CustomerGatewayId: aws.String(d.Get("customer_gateway_id").(string)),
		Options:           connectOpts,
		Type:              aws.String(d.Get("type").(string)),
		VpnGatewayId:      aws.String(d.Get("vpn_gateway_id").(string)),
	}

	// Create the VPN Connection
	log.Printf("[DEBUG] Creating vpn connection")
	resp, err := conn.CreateVpnConnection(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating vpn connection: %s", err)
	}

	// Store the ID
	vpnConnection := resp.VpnConnection
	d.SetId(*vpnConnection.VpnConnectionId)
	log.Printf("[INFO] VPN connection ID: %s", *vpnConnection.VpnConnectionId)

	// Wait for the connection to become available. This has an obscenely
	// high default timeout because AWS VPN connections are notoriously
	// slow at coming up or going down. There's also no point in checking
	// more frequently than every ten seconds.
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"available"},
		Refresh:    vpnConnectionRefreshFunc(conn, *vpnConnection.VpnConnectionId),
		Timeout:    40 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for VPN connection (%s) to become ready: %s",
			*vpnConnection.VpnConnectionId, stateErr)
	}

	// Create tags.
	if err := setTags(conn, d); err != nil {
		return err
	}

	// Read off the API to populate our RO fields.
	return resourceAwsVpnConnectionRead(d, meta)
}

func vpnConnectionRefreshFunc(conn *ec2.EC2, connectionId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			VpnConnectionIds: []*string{aws.String(connectionId)},
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VPNConnectionRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.VpnConnections) == 0 {
			return nil, "", nil
		}

		connection := resp.VpnConnections[0]
		return connection, *connection.State, nil
	}
}

func resourceAwsVpnConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
		VpnConnectionIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error finding VPN connection: %s", err)
			return err
		}
	}

	if len(resp.VpnConnections) != 1 {
		return fmt.Errorf("Error finding VPN connection: %s", d.Id())
	}

	vpnConnection := resp.VpnConnections[0]
	if vpnConnection == nil || *vpnConnection.State == "deleted" {
		// Seems we have lost our VPN Connection
		d.SetId("")
		return nil
	}

	// Set attributes under the user's control.
	d.Set("vpn_gateway_id", vpnConnection.VpnGatewayId)
	d.Set("customer_gateway_id", vpnConnection.CustomerGatewayId)
	d.Set("type", vpnConnection.Type)
	d.Set("tags", tagsToMap(vpnConnection.Tags))

	if vpnConnection.Options != nil {
		if err := d.Set("static_routes_only", vpnConnection.Options.StaticRoutesOnly); err != nil {
			return err
		}
	} else {
		//If there no Options on the connection then we do not support *static_routes*
		d.Set("static_routes_only", false)
	}

	// Set read only attributes.
	d.Set("customer_gateway_configuration", vpnConnection.CustomerGatewayConfiguration)

	if vpnConnection.CustomerGatewayConfiguration != nil {
		if tunnelInfo, err := xmlConfigToTunnelInfo(*vpnConnection.CustomerGatewayConfiguration); err != nil {
			log.Printf("[ERR] Error unmarshaling XML configuration for (%s): %s", d.Id(), err)
		} else {
			d.Set("tunnel1_address", tunnelInfo.Tunnel1Address)
			d.Set("tunnel1_cgw_inside_address", tunnelInfo.Tunnel1CgwInsideAddress)
			d.Set("tunnel1_vgw_inside_address", tunnelInfo.Tunnel1VgwInsideAddress)
			d.Set("tunnel1_preshared_key", tunnelInfo.Tunnel1PreSharedKey)
			d.Set("tunnel1_bgp_asn", tunnelInfo.Tunnel1BGPASN)
			d.Set("tunnel1_bgp_holdtime", tunnelInfo.Tunnel1BGPHoldTime)
			d.Set("tunnel2_address", tunnelInfo.Tunnel2Address)
			d.Set("tunnel2_preshared_key", tunnelInfo.Tunnel2PreSharedKey)
			d.Set("tunnel2_cgw_inside_address", tunnelInfo.Tunnel2CgwInsideAddress)
			d.Set("tunnel2_vgw_inside_address", tunnelInfo.Tunnel2VgwInsideAddress)
			d.Set("tunnel2_bgp_asn", tunnelInfo.Tunnel2BGPASN)
			d.Set("tunnel2_bgp_holdtime", tunnelInfo.Tunnel2BGPHoldTime)
		}
	}

	if err := d.Set("vgw_telemetry", telemetryToMapList(vpnConnection.VgwTelemetry)); err != nil {
		return err
	}
	if err := d.Set("routes", routesToMapList(vpnConnection.Routes)); err != nil {
		return err
	}

	return nil
}

func resourceAwsVpnConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Update tags if required.
	if err := setTags(conn, d); err != nil {
		return err
	}

	d.SetPartial("tags")

	return resourceAwsVpnConnectionRead(d, meta)
}

func resourceAwsVpnConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	_, err := conn.DeleteVpnConnection(&ec2.DeleteVpnConnectionInput{
		VpnConnectionId: aws.String(d.Id()),
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
			return nil
		} else {
			log.Printf("[ERROR] Error deleting VPN connection: %s", err)
			return err
		}
	}

	// These things can take quite a while to tear themselves down and any
	// attempt to modify resources they reference (e.g. CustomerGateways or
	// VPN Gateways) before deletion will result in an error. Furthermore,
	// they don't just disappear. The go into "deleted" state. We need to
	// wait to ensure any other modifications the user might make to their
	// VPC stack can safely run.
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Refresh:    vpnConnectionRefreshFunc(conn, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for VPN connection (%s) to delete: %s", d.Id(), err)
	}

	return nil
}

// routesToMapList turns the list of routes into a list of maps.
func routesToMapList(routes []*ec2.VpnStaticRoute) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(routes))
	for _, r := range routes {
		staticRoute := make(map[string]interface{})
		staticRoute["destination_cidr_block"] = *r.DestinationCidrBlock
		staticRoute["state"] = *r.State

		if r.Source != nil {
			staticRoute["source"] = *r.Source
		}

		result = append(result, staticRoute)
	}

	return result
}

// telemetryToMapList turns the VGW telemetry into a list of maps.
func telemetryToMapList(telemetry []*ec2.VgwTelemetry) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(telemetry))
	for _, t := range telemetry {
		vgw := make(map[string]interface{})
		vgw["accepted_route_count"] = *t.AcceptedRouteCount
		vgw["outside_ip_address"] = *t.OutsideIpAddress
		vgw["status"] = *t.Status
		vgw["status_message"] = *t.StatusMessage

		// LastStatusChange is a time.Time(). Convert it into a string
		// so it can be handled by schema's type system.
		vgw["last_status_change"] = t.LastStatusChange.String()
		result = append(result, vgw)
	}

	return result
}

func xmlConfigToTunnelInfo(xmlConfig string) (*TunnelInfo, error) {
	var vpnConfig XmlVpnConnectionConfig
	if err := xml.Unmarshal([]byte(xmlConfig), &vpnConfig); err != nil {
		return nil, fmt.Errorf("Error Unmarshalling XML: %s", err)
	}

	// don't expect consistent ordering from the XML
	sort.Sort(vpnConfig)

	tunnelInfo := TunnelInfo{
		Tunnel1Address:          vpnConfig.Tunnels[0].OutsideAddress,
		Tunnel1PreSharedKey:     vpnConfig.Tunnels[0].PreSharedKey,
		Tunnel1CgwInsideAddress: vpnConfig.Tunnels[0].CgwInsideAddress,
		Tunnel1VgwInsideAddress: vpnConfig.Tunnels[0].VgwInsideAddress,
		Tunnel1BGPASN:           vpnConfig.Tunnels[0].BGPASN,
		Tunnel1BGPHoldTime:      vpnConfig.Tunnels[0].BGPHoldTime,
		Tunnel2Address:          vpnConfig.Tunnels[1].OutsideAddress,
		Tunnel2PreSharedKey:     vpnConfig.Tunnels[1].PreSharedKey,
		Tunnel2CgwInsideAddress: vpnConfig.Tunnels[1].CgwInsideAddress,
		Tunnel2VgwInsideAddress: vpnConfig.Tunnels[1].VgwInsideAddress,
		Tunnel2BGPASN:           vpnConfig.Tunnels[1].BGPASN,
		Tunnel2BGPHoldTime:      vpnConfig.Tunnels[1].BGPHoldTime,
	}

	return &tunnelInfo, nil
}

func validateVpnConnectionTunnelPreSharedKey(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if (len(value) < 8) || (len(value) > 64) {
		errors = append(errors, fmt.Errorf("%q must be between 8 and 64 characters in length", k))
	}

	if strings.HasPrefix(value, "0") {
		errors = append(errors, fmt.Errorf("%q cannot start with zero character", k))
	}

	if !regexp.MustCompile(`^[0-9a-zA-Z_.]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q can only contain alphanumeric, period and underscore characters", k))
	}

	return
}

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_VpnTunnelOptionsSpecification.html
func validateVpnConnectionTunnelInsideCIDR(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, ipnet, err := net.ParseCIDR(value)

	if err != nil {
		errors = append(errors, fmt.Errorf("%q must contain a valid CIDR, got error parsing: %s", k, err))
		return
	}

	if !strings.HasSuffix(ipnet.String(), "/30") {
		errors = append(errors, fmt.Errorf("%q must be /30 CIDR", k))
	}

	if !strings.HasPrefix(ipnet.String(), "169.254.") {
		errors = append(errors, fmt.Errorf("%q must be within 169.254.0.0/16", k))
	} else if ipnet.String() == "169.254.0.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.0.0/30", k))
	} else if ipnet.String() == "169.254.1.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.1.0/30", k))
	} else if ipnet.String() == "169.254.2.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.2.0/30", k))
	} else if ipnet.String() == "169.254.3.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.3.0/30", k))
	} else if ipnet.String() == "169.254.4.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.4.0/30", k))
	} else if ipnet.String() == "169.254.5.0/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.5.0/30", k))
	} else if ipnet.String() == "169.254.169.252/30" {
		errors = append(errors, fmt.Errorf("%q cannot be 169.254.169.252/30", k))
	}

	return
}

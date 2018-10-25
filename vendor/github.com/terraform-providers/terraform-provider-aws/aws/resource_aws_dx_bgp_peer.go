package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDxBgpPeer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxBgpPeerCreate,
		Read:   resourceAwsDxBgpPeerRead,
		Delete: resourceAwsDxBgpPeerDelete,

		Schema: map[string]*schema.Schema{
			"address_family": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{directconnect.AddressFamilyIpv4, directconnect.AddressFamilyIpv6}, false),
			},
			"bgp_asn": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"virtual_interface_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"amazon_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"bgp_auth_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"customer_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"bgp_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceAwsDxBgpPeerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	vifId := d.Get("virtual_interface_id").(string)
	addrFamily := d.Get("address_family").(string)
	asn := int64(d.Get("bgp_asn").(int))

	req := &directconnect.CreateBGPPeerInput{
		VirtualInterfaceId: aws.String(vifId),
		NewBGPPeer: &directconnect.NewBGPPeer{
			AddressFamily: aws.String(addrFamily),
			Asn:           aws.Int64(asn),
		},
	}
	if v, ok := d.GetOk("amazon_address"); ok && v.(string) != "" {
		req.NewBGPPeer.AmazonAddress = aws.String(v.(string))
	}
	if v, ok := d.GetOk("bgp_auth_key"); ok && v.(string) != "" {
		req.NewBGPPeer.AuthKey = aws.String(v.(string))
	}
	if v, ok := d.GetOk("customer_address"); ok && v.(string) != "" {
		req.NewBGPPeer.CustomerAddress = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Direct Connect BGP peer: %#v", req)
	_, err := conn.CreateBGPPeer(req)
	if err != nil {
		return fmt.Errorf("Error creating Direct Connect BGP peer: %s", err)
	}

	d.SetId(fmt.Sprintf("%s-%s-%d", vifId, addrFamily, asn))

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			directconnect.BGPPeerStatePending,
		},
		Target: []string{
			directconnect.BGPPeerStateAvailable,
			directconnect.BGPPeerStateVerifying,
		},
		Refresh:    dxBgpPeerStateRefresh(conn, vifId, addrFamily, asn),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect BGP peer (%s) to be available: %s", d.Id(), err)
	}

	return resourceAwsDxBgpPeerRead(d, meta)
}

func resourceAwsDxBgpPeerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	vifId := d.Get("virtual_interface_id").(string)
	addrFamily := d.Get("address_family").(string)
	asn := int64(d.Get("bgp_asn").(int))

	bgpPeerRaw, state, err := dxBgpPeerStateRefresh(conn, vifId, addrFamily, asn)()
	if err != nil {
		return fmt.Errorf("Error reading Direct Connect BGP peer: %s", err)
	}
	if state == directconnect.BGPPeerStateDeleted {
		log.Printf("[WARN] Direct Connect BGP peer (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	bgpPeer := bgpPeerRaw.(*directconnect.BGPPeer)
	d.Set("amazon_address", bgpPeer.AmazonAddress)
	d.Set("bgp_auth_key", bgpPeer.AuthKey)
	d.Set("customer_address", bgpPeer.CustomerAddress)
	d.Set("bgp_status", bgpPeer.BgpStatus)

	return nil
}

func resourceAwsDxBgpPeerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	vifId := d.Get("virtual_interface_id").(string)
	addrFamily := d.Get("address_family").(string)
	asn := int64(d.Get("bgp_asn").(int))

	log.Printf("[DEBUG] Deleting Direct Connect BGP peer: %s", d.Id())
	_, err := conn.DeleteBGPPeer(&directconnect.DeleteBGPPeerInput{
		Asn:                aws.Int64(asn),
		CustomerAddress:    aws.String(d.Get("customer_address").(string)),
		VirtualInterfaceId: aws.String(vifId),
	})
	if err != nil {
		// This is the error returned if the BGP peering has already gone.
		if isAWSErr(err, "DirectConnectClientException", "The last BGP Peer on a Virtual Interface cannot be deleted") {
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			directconnect.BGPPeerStateAvailable,
			directconnect.BGPPeerStateDeleting,
			directconnect.BGPPeerStatePending,
			directconnect.BGPPeerStateVerifying,
		},
		Target: []string{
			directconnect.BGPPeerStateDeleted,
		},
		Refresh:    dxBgpPeerStateRefresh(conn, vifId, addrFamily, asn),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect BGP peer (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func dxBgpPeerStateRefresh(conn *directconnect.DirectConnect, vifId, addrFamily string, asn int64) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		vif, err := dxVirtualInterfaceRead(vifId, conn)
		if err != nil {
			return nil, "", err
		}
		if vif == nil {
			return "", directconnect.BGPPeerStateDeleted, nil
		}

		for _, bgpPeer := range vif.BgpPeers {
			if aws.StringValue(bgpPeer.AddressFamily) == addrFamily && aws.Int64Value(bgpPeer.Asn) == asn {
				return bgpPeer, aws.StringValue(bgpPeer.BgpPeerState), nil
			}
		}

		return "", directconnect.BGPPeerStateDeleted, nil
	}
}

package alicloud

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/slb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAliyunSlb() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSlbCreate,
		Read:   resourceAliyunSlbRead,
		Update: resourceAliyunSlbUpdate,
		Delete: resourceAliyunSlbDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSlbName,
				Computed:     true,
			},

			"internet": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"internet_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateSlbInternetChargeType,
			},

			"bandwidth": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validateSlbBandwidth,
			},

			"listener": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_port": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateInstancePort,
							Required:     true,
						},

						"instance_protocol": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateInstanceProtocol,
							Required:     true,
						},

						"lb_port": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateInstancePort,
							Required:     true,
						},

						"lb_protocol": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateInstanceProtocol,
							Required:     true,
						},

						"ssl_certificate_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"bandwidth": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateSlbListenerBandwidth,
							Required:     true,
						},
					},
				},
				Set: resourceAliyunSlbListenerHash,
			},

			"instances": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set:      schema.HashString,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAliyunSlbCreate(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn

	var slbName string
	if v, ok := d.GetOk("name"); ok {
		slbName = v.(string)
	} else {
		slbName = resource.PrefixedUniqueId("tf-lb-")
		d.Set("name", slbName)
	}

	slbArgs := &slb.CreateLoadBalancerArgs{
		RegionId:         getRegion(d, meta),
		LoadBalancerName: slbName,
	}

	if internet, ok := d.GetOk("internet"); ok && internet.(bool) {
		slbArgs.AddressType = slb.InternetAddressType
		d.Set("internet", true)
	} else {
		slbArgs.AddressType = slb.IntranetAddressType
		d.Set("internet", false)
	}

	if v, ok := d.GetOk("internet_charge_type"); ok && v.(string) != "" {
		slbArgs.InternetChargeType = common.InternetChargeType(v.(string))
	}

	if v, ok := d.GetOk("bandwidth"); ok && v.(int) != 0 {
		slbArgs.Bandwidth = v.(int)
	}

	if v, ok := d.GetOk("vswitch_id"); ok && v.(string) != "" {
		slbArgs.VSwitchId = v.(string)
	}

	slb, err := slbconn.CreateLoadBalancer(slbArgs)
	if err != nil {
		return err
	}

	d.SetId(slb.LoadBalancerId)

	d.Partial(true)
	d.SetPartial("name")
	d.SetPartial("internet")
	d.SetPartial("internet_charge_type")
	d.SetPartial("bandwidth")
	d.SetPartial("vswitch_id")

	return resourceAliyunSlbUpdate(d, meta)
}

func resourceAliyunSlbRead(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn
	loadBalancer, err := slbconn.DescribeLoadBalancerAttribute(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", loadBalancer.LoadBalancerName)

	if loadBalancer.AddressType == slb.InternetAddressType {
		d.Set("internal", true)
	} else {
		d.Set("internal", false)
	}

	d.Set("internet_charge_type", loadBalancer.InternetChargeType)
	d.Set("bandwidth", loadBalancer.Bandwidth)
	d.Set("vswitch_id", loadBalancer.VSwitchId)
	d.Set("address", loadBalancer.Address)
	d.Set("vpc_id", loadBalancer.VpcId)

	return nil
}

func resourceAliyunSlbUpdate(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn

	d.Partial(true)

	if d.HasChange("name") {
		err := slbconn.SetLoadBalancerName(d.Id(), d.Get("name").(string))
		if err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.Get("vswitch_id") == "" {
		//don't intranet web, then can modify bandwidth
		if d.HasChange("bandwidth") {
			args := &slb.ModifyLoadBalancerInternetSpecArgs{
				LoadBalancerId: d.Id(),
				Bandwidth:      d.Get("bandwidth").(int),
			}
			err := slbconn.ModifyLoadBalancerInternetSpec(args)
			if err != nil {
				return err
			}

			d.SetPartial("bandwidth")
		}
	}

	if d.HasChange("listener") {
		o, n := d.GetChange("listener")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove, _ := expandListeners(os.Difference(ns).List())
		add, _ := expandListeners(ns.Difference(os).List())

		if len(remove) > 0 {
			for _, listener := range remove {
				err := slbconn.DeleteLoadBalancerListener(d.Id(), listener.LoadBalancerPort)
				if err != nil {
					return fmt.Errorf("Failure removing outdated SLB listeners: %s", err)
				}
			}
		}

		if len(add) > 0 {
			for _, listener := range add {
				err := createListener(slbconn, d.Id(), listener)
				if err != nil {
					return fmt.Errorf("Failure add SLB listeners: %s", err)
				}
			}
		}

		d.SetPartial("listener")
	}

	// If we currently have instances, or did have instances,
	// we want to figure out what to add and remove from the load
	// balancer
	if d.HasChange("instances") {
		o, n := d.GetChange("instances")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandBackendServers(os.Difference(ns).List())
		add := expandBackendServers(ns.Difference(os).List())

		if len(add) > 0 {
			_, err := slbconn.AddBackendServers(d.Id(), add)
			if err != nil {
				return err
			}
		}
		if len(remove) > 0 {
			removeBackendServers := make([]string, 0, len(remove))
			for _, e := range remove {
				removeBackendServers = append(removeBackendServers, e.ServerId)
			}
			_, err := slbconn.RemoveBackendServers(d.Id(), removeBackendServers)
			if err != nil {
				return err
			}
		}

		d.SetPartial("instances")
	}

	return resourceAliyunSlbRead(d, meta)
}

func resourceAliyunSlbDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).slbconn

	err := conn.DeleteLoadBalancer(d.Id())
	if err != nil {
		return err
	}
	return nil
}

func resourceAliyunSlbListenerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["instance_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["instance_protocol"].(string))))
	buf.WriteString(fmt.Sprintf("%d-", m["lb_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["lb_protocol"].(string))))

	buf.WriteString(fmt.Sprintf("%d-", m["bandwidth"].(int)))

	if v, ok := m["ssl_certificate_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func createListener(conn *slb.Client, loadBalancerId string, listener *Listener) error {
	if listener.Protocol == strings.ToLower("tcp") {
		args := &slb.CreateLoadBalancerTCPListenerArgs{
			LoadBalancerId:    loadBalancerId,
			ListenerPort:      listener.LoadBalancerPort,
			BackendServerPort: listener.InstancePort,
			Bandwidth:         listener.Bandwidth,
		}
		if err := conn.CreateLoadBalancerTCPListener(args); err != nil {
			return err
		}
	}

	if listener.Protocol == strings.ToLower("http") {
		args := &slb.CreateLoadBalancerHTTPListenerArgs{
			LoadBalancerId:    loadBalancerId,
			ListenerPort:      listener.LoadBalancerPort,
			BackendServerPort: listener.InstancePort,
			Bandwidth:         listener.Bandwidth,
			StickySession:     slb.OffFlag,
			HealthCheck:       slb.OffFlag,
		}

		if err := conn.CreateLoadBalancerHTTPListener(args); err != nil {
			return err
		}
	}

	if err := conn.StartLoadBalancerListener(loadBalancerId, listener.LoadBalancerPort); err != nil {
		return err
	}

	return nil
}

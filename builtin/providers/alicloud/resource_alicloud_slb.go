package alicloud

import (
	"bytes"
	"fmt"
	"strings"

	"errors"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/slb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
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
				Default:      "paybytraffic",
				ValidateFunc: validateSlbInternetChargeType,
			},

			"bandwidth": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateSlbBandwidth,
				Computed:     true,
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

						"bandwidth": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateSlbListenerBandwidth,
							Required:     true,
						},
						"scheduler": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateSlbListenerScheduler,
							Optional:     true,
							Default:      slb.WRRScheduler,
						},
						//http & https
						"sticky_session": &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validateAllowedStringValue([]string{
								string(slb.OnFlag),
								string(slb.OffFlag)}),
							Optional: true,
							Default:  slb.OffFlag,
						},
						//http & https
						"sticky_session_type": &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validateAllowedStringValue([]string{
								string(slb.InsertStickySessionType),
								string(slb.ServerStickySessionType)}),
							Optional: true,
						},
						//http & https
						"cookie_timeout": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateSlbListenerCookieTimeout,
							Optional:     true,
						},
						//http & https
						"cookie": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateSlbListenerCookie,
							Optional:     true,
						},
						//tcp & udp
						"persistence_timeout": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateSlbListenerPersistenceTimeout,
							Optional:     true,
							Default:      0,
						},
						//http & https
						"health_check": &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validateAllowedStringValue([]string{
								string(slb.OnFlag),
								string(slb.OffFlag)}),
							Optional: true,
							Default:  slb.OffFlag,
						},
						//tcp
						"health_check_type": &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validateAllowedStringValue([]string{
								string(slb.TCPHealthCheckType),
								string(slb.HTTPHealthCheckType)}),
							Optional: true,
							Default:  slb.TCPHealthCheckType,
						},
						//http & https & tcp
						"health_check_domain": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateSlbListenerHealthCheckDomain,
							Optional:     true,
						},
						//http & https & tcp
						"health_check_uri": &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validateSlbListenerHealthCheckUri,
							Optional:     true,
						},
						"health_check_connect_port": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateSlbListenerHealthCheckConnectPort,
							Optional:     true,
						},
						"healthy_threshold": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateIntegerInRange(1, 10),
							Optional:     true,
						},
						"unhealthy_threshold": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateIntegerInRange(1, 10),
							Optional:     true,
						},

						"health_check_timeout": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateIntegerInRange(1, 50),
							Optional:     true,
						},
						"health_check_interval": &schema.Schema{
							Type:         schema.TypeInt,
							ValidateFunc: validateIntegerInRange(1, 5),
							Optional:     true,
						},
						//http & https & tcp
						"health_check_http_code": &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validateAllowedSplitStringValue([]string{
								string(slb.HTTP_2XX),
								string(slb.HTTP_3XX),
								string(slb.HTTP_4XX),
								string(slb.HTTP_5XX)}, ","),
							Optional: true,
						},
						//https
						"ssl_certificate_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						//https
						//"ca_certificate_id": &schema.Schema{
						//	Type:     schema.TypeString,
						//	Optional: true,
						//},
					},
				},
				Set: resourceAliyunSlbListenerHash,
			},

			//deprecated
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
		slbArgs.InternetChargeType = slb.InternetChargeType(v.(string))
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

	if loadBalancer == nil {
		d.SetId("")
		return nil
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

	if d.Get("internet") == true && d.Get("internet_charge_type") == "paybybandwidth" {
		//don't intranet web and paybybandwidth, then can modify bandwidth
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
					return fmt.Errorf("Failure removing outdated SLB listeners: %#v", err)
				}
			}
		}

		if len(add) > 0 {
			for _, listener := range add {
				err := createListener(slbconn, d.Id(), listener)
				if err != nil {
					return fmt.Errorf("Failure add SLB listeners: %#v", err)
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

	d.Partial(false)

	return resourceAliyunSlbRead(d, meta)
}

func resourceAliyunSlbDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).slbconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		err := conn.DeleteLoadBalancer(d.Id())

		if err != nil {
			return resource.NonRetryableError(err)
		}

		loadBalancer, err := conn.DescribeLoadBalancerAttribute(d.Id())
		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == LoadBalancerNotFound {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		if loadBalancer != nil {
			return resource.RetryableError(fmt.Errorf("LoadBalancer in use - trying again while it deleted."))
		}
		return nil
	})
}

func resourceAliyunSlbListenerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["instance_port"].(int)))
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

	errTypeJudge := func(err error) error {
		if err != nil {
			if listenerType, ok := err.(*ListenerErr); ok {
				if listenerType.ErrType == HealthCheckErrType {
					return fmt.Errorf("When the HealthCheck is %s, then related HealthCheck parameter "+
						"must have.", slb.OnFlag)
				} else if listenerType.ErrType == StickySessionErrType {
					return fmt.Errorf("When the StickySession is %s, then StickySessionType parameter "+
						"must have.", slb.OnFlag)
				} else if listenerType.ErrType == CookieTimeOutErrType {
					return fmt.Errorf("When the StickySession is %s and StickySessionType is %s, "+
						"then CookieTimeout parameter must have.", slb.OnFlag, slb.InsertStickySessionType)
				} else if listenerType.ErrType == CookieErrType {
					return fmt.Errorf("When the StickySession is %s and StickySessionType is %s, "+
						"then Cookie parameter must have.", slb.OnFlag, slb.ServerStickySessionType)
				}
				return fmt.Errorf("slb listener check errtype not found.")
			}
		}
		return nil
	}

	if listener.Protocol == strings.ToLower("tcp") {

		args := getTcpListenerArgs(loadBalancerId, listener)

		if err := conn.CreateLoadBalancerTCPListener(&args); err != nil {
			return err
		}
	} else if listener.Protocol == strings.ToLower("http") {
		args, argsErr := getHttpListenerArgs(loadBalancerId, listener)
		if paramErr := errTypeJudge(argsErr); paramErr != nil {
			return paramErr
		}

		if err := conn.CreateLoadBalancerHTTPListener(&args); err != nil {
			return err
		}
	} else if listener.Protocol == strings.ToLower("https") {
		listenerType, err := getHttpListenerType(loadBalancerId, listener)
		if paramErr := errTypeJudge(err); paramErr != nil {
			return paramErr
		}

		args := &slb.CreateLoadBalancerHTTPSListenerArgs{
			HTTPListenerType: listenerType,
		}
		if listener.SSLCertificateId == "" {
			return fmt.Errorf("Server Certificated Id cann't be null")
		}

		args.ServerCertificateId = listener.SSLCertificateId

		if err := conn.CreateLoadBalancerHTTPSListener(args); err != nil {
			return err
		}
	} else if listener.Protocol == strings.ToLower("udp") {
		args := getUdpListenerArgs(loadBalancerId, listener)

		if err := conn.CreateLoadBalancerUDPListener(&args); err != nil {
			return err
		}
	}

	if err := conn.StartLoadBalancerListener(loadBalancerId, listener.LoadBalancerPort); err != nil {
		return err
	}

	return nil
}

func getTcpListenerArgs(loadBalancerId string, listener *Listener) slb.CreateLoadBalancerTCPListenerArgs {
	args := slb.CreateLoadBalancerTCPListenerArgs{
		LoadBalancerId:            loadBalancerId,
		ListenerPort:              listener.LoadBalancerPort,
		BackendServerPort:         listener.InstancePort,
		Bandwidth:                 listener.Bandwidth,
		Scheduler:                 listener.Scheduler,
		PersistenceTimeout:        listener.PersistenceTimeout,
		HealthCheckType:           listener.HealthCheckType,
		HealthCheckDomain:         listener.HealthCheckDomain,
		HealthCheckURI:            listener.HealthCheckURI,
		HealthCheckConnectPort:    listener.HealthCheckConnectPort,
		HealthyThreshold:          listener.HealthyThreshold,
		UnhealthyThreshold:        listener.UnhealthyThreshold,
		HealthCheckConnectTimeout: listener.HealthCheckTimeout,
		HealthCheckInterval:       listener.HealthCheckInterval,
		HealthCheckHttpCode:       listener.HealthCheckHttpCode,
	}
	return args
}

func getUdpListenerArgs(loadBalancerId string, listener *Listener) slb.CreateLoadBalancerUDPListenerArgs {
	args := slb.CreateLoadBalancerUDPListenerArgs{
		LoadBalancerId:            loadBalancerId,
		ListenerPort:              listener.LoadBalancerPort,
		BackendServerPort:         listener.InstancePort,
		Bandwidth:                 listener.Bandwidth,
		PersistenceTimeout:        listener.PersistenceTimeout,
		HealthCheckConnectTimeout: listener.HealthCheckTimeout,
		HealthCheckInterval:       listener.HealthCheckInterval,
	}
	return args
}

func getHttpListenerType(loadBalancerId string, listener *Listener) (listenType slb.HTTPListenerType, err error) {

	if listener.HealthCheck == slb.OnFlag {
		if listener.HealthCheckURI == "" || listener.HealthCheckDomain == "" || listener.HealthCheckConnectPort == 0 ||
			listener.HealthyThreshold == 0 || listener.UnhealthyThreshold == 0 || listener.HealthCheckTimeout == 0 ||
			listener.HealthCheckHttpCode == "" || listener.HealthCheckInterval == 0 {

			errMsg := errors.New("err: HealthCheck empty.")
			return listenType, &ListenerErr{HealthCheckErrType, errMsg}
		}
	}

	if listener.StickySession == slb.OnFlag {
		if listener.StickySessionType == "" {
			errMsg := errors.New("err: stickySession empty.")
			return listenType, &ListenerErr{StickySessionErrType, errMsg}
		}

		if listener.StickySessionType == slb.InsertStickySessionType {
			if listener.CookieTimeout == 0 {
				errMsg := errors.New("err: cookieTimeout empty.")
				return listenType, &ListenerErr{CookieTimeOutErrType, errMsg}
			}
		} else if listener.StickySessionType == slb.ServerStickySessionType {
			if listener.Cookie == "" {
				errMsg := errors.New("err: cookie empty.")
				return listenType, &ListenerErr{CookieErrType, errMsg}
			}
		}
	}

	httpListenertType := slb.HTTPListenerType{
		LoadBalancerId:         loadBalancerId,
		ListenerPort:           listener.LoadBalancerPort,
		BackendServerPort:      listener.InstancePort,
		Bandwidth:              listener.Bandwidth,
		Scheduler:              listener.Scheduler,
		HealthCheck:            listener.HealthCheck,
		StickySession:          listener.StickySession,
		StickySessionType:      listener.StickySessionType,
		CookieTimeout:          listener.CookieTimeout,
		Cookie:                 listener.Cookie,
		HealthCheckDomain:      listener.HealthCheckDomain,
		HealthCheckURI:         listener.HealthCheckURI,
		HealthCheckConnectPort: listener.HealthCheckConnectPort,
		HealthyThreshold:       listener.HealthyThreshold,
		UnhealthyThreshold:     listener.UnhealthyThreshold,
		HealthCheckTimeout:     listener.HealthCheckTimeout,
		HealthCheckInterval:    listener.HealthCheckInterval,
		HealthCheckHttpCode:    listener.HealthCheckHttpCode,
	}

	return httpListenertType, err
}

func getHttpListenerArgs(loadBalancerId string, listener *Listener) (listenType slb.CreateLoadBalancerHTTPListenerArgs, err error) {
	httpListenerType, err := getHttpListenerType(loadBalancerId, listener)
	if err != nil {
		return listenType, err
	}

	httpArgs := slb.CreateLoadBalancerHTTPListenerArgs(httpListenerType)
	return httpArgs, err
}

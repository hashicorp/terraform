package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElbListener() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElbListenerCreate,
		Read:   resourceAwsElbListenerRead,
		Delete: resourceAwsElbListenerDelete,

		Schema: map[string]*schema.Schema{
			"loadbalancer_names": {
				Type:     schema.TypeList,
				ForceNew: true,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"instance_port": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,

				ValidateFunc: validateIntegerInRange(1, 65535),
			},

			"instance_protocol": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateListenerProtocol,
			},

			"lb_port": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateIntegerInRange(1, 65535),
			},

			"lb_protocol": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateListenerProtocol,
			},

			"ssl_certificate_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}
func expandElbListener(d *schema.ResourceData) ([]*elb.Listener, error) {
	listeners := make([]*elb.Listener, 0)
	l := &elb.Listener{
		InstancePort:     aws.Int64(int64(d.Get("instance_port").(int))),
		InstanceProtocol: aws.String(d.Get("instance_protocol").(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("lb_port").(int))),
		Protocol:         aws.String(d.Get("lb_protocol").(string)),
	}

	if v, ok := d.GetOk("ssl_certificate_id"); ok {
		l.SSLCertificateId = aws.String(v.(string))
	}

	var valid bool
	if l.SSLCertificateId != nil && *l.SSLCertificateId != "" {
		// validate the protocol is correct
		for _, p := range []string{"https", "ssl"} {
			if (strings.ToLower(*l.InstanceProtocol) == p) || (strings.ToLower(*l.Protocol) == p) {
				valid = true
			}
		}
	} else {
		valid = true
	}

	if valid {
		listeners = append(listeners, l)
	} else {
		return nil, fmt.Errorf("[ERR] ELB Listener: ssl_certificate_id may be set only when protocol is 'https' or 'ssl'")
	}

	return listeners, nil
}

func resourceAwsElbListenerCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	listeners, elbErr := expandElbListener(d)
	if elbErr != nil {
		return elbErr
	}

	loadBalancers := d.Get("loadbalancer_names").([]interface{})
	for _, lb := range loadBalancers {
		input := &elb.CreateLoadBalancerListenersInput{
			Listeners:        listeners,
			LoadBalancerName: aws.String(lb.(string)),
		}

		_, err := elbconn.CreateLoadBalancerListeners(input)
		if err != nil {
			return err
		}
	}

	d.SetId(strconv.Itoa(d.Get("lb_port").(int)))

	return resourceAwsElbListenerRead(d, meta)
}

func resourceAwsElbListenerRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbnames := d.Get("loadbalancer_names").([]interface{})

	resp, err := elbconn.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: expandStringList(lbnames),
	})
	if err != nil {
		return err
	}
	if len(resp.LoadBalancerDescriptions) != len(lbnames) {
		log.Printf("[ERROR] Unable to find all of the LoadBalancers specified")
	}

	// for all of the loadbalancers in the resource
	// range the listeners for the specific load balancer port
	// if a listener with the specific port is not found, then error out
	// otherwise continue
	for _, lb := range resp.LoadBalancerDescriptions {
		found := false
		for _, listener := range lb.ListenerDescriptions {
			if *listener.Listener.LoadBalancerPort == int64(d.Get("lb_port").(int)) {
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Listener with port %q not found in %q listener list", d.Get("lb_port").(int), *lb.LoadBalancerName)
			d.SetId("")
		}
	}
	return nil
}
func resourceAwsElbListenerDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	loadBalancers := d.Get("loadbalancer_names").([]interface{})
	for _, lb := range loadBalancers {
		input := &elb.DeleteLoadBalancerListenersInput{
			LoadBalancerName:  aws.String(lb.(string)),
			LoadBalancerPorts: []*int64{aws.Int64(int64(d.Get("lb_port").(int)))},
		}

		_, err := elbconn.DeleteLoadBalancerListeners(input)
		if err != nil {
			return err
		}

	}
	return nil
}

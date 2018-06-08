package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsLbListener() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLbListenerRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"load_balancer_arn", "port"},
			},

			"load_balancer_arn": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"arn"},
			},
			"port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"arn"},
			},

			"protocol": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssl_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"certificate_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_action": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_group_arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsLbListenerRead(d *schema.ResourceData, meta interface{}) error {
	if _, ok := d.GetOk("arn"); ok {
		d.SetId(d.Get("arn").(string))
		//log.Printf("[DEBUG] read listener %s", d.Get("arn").(string))
		return resourceAwsLbListenerRead(d, meta)
	}

	conn := meta.(*AWSClient).elbv2conn
	lbArn, lbOk := d.GetOk("load_balancer_arn")
	port, portOk := d.GetOk("port")
	if !lbOk || !portOk {
		return errors.New("both load_balancer_arn and port must be set")
	}
	resp, err := conn.DescribeListeners(&elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbArn.(string)),
	})
	if err != nil {
		return err
	}
	if len(resp.Listeners) == 0 {
		return fmt.Errorf("[DEBUG] no listener exists for load balancer: %s", lbArn)
	}
	for _, listener := range resp.Listeners {
		if *listener.Port == int64(port.(int)) {
			//log.Printf("[DEBUG] get listener arn for %s:%s: %s", lbArn, port, *listener.Port)
			d.SetId(*listener.ListenerArn)
			return resourceAwsLbListenerRead(d, meta)
		}
	}

	return errors.New("failed to get listener arn with given arguments")
}

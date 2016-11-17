package akamai

import (
	"fmt"
	"log"

	"github.com/Comcast/go-edgegrid/edgegrid"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAkamaiGTMProperty() *schema.Resource {
	return &schema.Resource{
		Create: resourceGTMPropertyCreate,
		Read:   resourceGTMPropertyRead,
		Update: resourceGTMPropertyUpdate,
		Delete: resourceGTMPropertyDelete,

		Schema: map[string]*schema.Schema{
			"balance_by_download_score": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"dynamic_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"failover_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"failback_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"handout_mode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"health_max": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"health_multiplier": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"health_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"load_imbalance_percentage": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"ipv6": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"score_aggregation_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"static_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"stickiness_bonus_percentage": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"stickiness_bonus_constant": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"use_computed_targets": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"liveness_test": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"http_error_3xx": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"http_error_4xx": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"http_error_5xx": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"test_interval": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"test_object": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"test_object_port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"test_object_protocol": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"test_object_username": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"test_object_password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"test_timeout": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"disable_nonstandard_port_warning": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"request_string": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"response_string": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"ssl_client_private_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"ssl_certificate": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"host_header": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"traffic_target": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"data_center_id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"weight": &schema.Schema{
							Type:     schema.TypeFloat,
							Required: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"servers": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set: func(v interface{}) int {
								return hashcode.String(v.(string))
							},
						},
					},
				},
			},
		},
	}
}

func resourceGTMPropertyCreate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	property := property(d)
	log.Printf("[INFO] Creating GTM property: %s", name)
	created, err := meta.(*Clients).GTM.PropertyCreate(d.Get("domain").(string), property)
	if err != nil {
		return err
	}

	d.SetId(created.Property.Name)

	err = resourceGTMWaitUntilDeployed(d, meta)
	if err != nil {
		return err
	}

	return resourceGTMPropertyRead(d, meta)
}

func resourceGTMPropertyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Reading GTM property: %s", d.Id())
	prop, err := meta.(*Clients).GTM.Property(d.Get("domain").(string), d.Get("name").(string))
	if err != nil {
		return err
	}

	d.Set("name", prop.Name)

	return nil
}

func resourceGTMPropertyUpdate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	log.Printf("[INFO] Updating GTM property: %s", name)
	updated, err := meta.(*Clients).GTM.PropertyCreate(d.Get("domain").(string), property(d))
	if err != nil {
		return err
	}

	d.SetId(updated.Property.Name)

	err = resourceGTMWaitUntilDeployed(d, meta)
	if err != nil {
		return err
	}

	return resourceGTMPropertyRead(d, meta)
}

func resourceGTMPropertyDelete(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	log.Printf("[INFO] Deleting property: %s", name)
	_, err := meta.(*Clients).GTM.PropertyDelete(d.Get("domain").(string), name)
	if err != nil {
		return nil
	}

	d.SetId("")

	return nil
}

func property(d *schema.ResourceData) *edgegrid.Property {
	return &edgegrid.Property{
		Cname:                     d.Get("cname").(string),
		Name:                      d.Get("name").(string),
		Type:                      d.Get("type").(string),
		Ipv6:                      d.Get("ipv6").(bool),
		DynamicTTL:                d.Get("dynamic_ttl").(int),
		StaticTTL:                 d.Get("static_ttl").(int),
		HandoutMode:               d.Get("handout_mode").(string),
		FailbackDelay:             d.Get("failback_delay").(int),
		FailoverDelay:             d.Get("failover_delay").(int),
		ScoreAggregationType:      d.Get("score_aggregation_type").(string),
		LoadImbalancePercentage:   d.Get("load_imbalance_percentage").(float64),
		StickinessBonusPercentage: d.Get("stickiness_bonus_percentage").(int),
		TrafficTargets:            trafficTargets(d),
		LivenessTests:             livenessTests(d),
	}
}

func trafficTargets(d *schema.ResourceData) []edgegrid.TrafficTarget {
	targets := []edgegrid.TrafficTarget{}
	targetsCount := d.Get("traffic_target.#").(int)

	for i := 0; i < targetsCount; i++ {
		prefix := fmt.Sprintf("traffic_target.%d", i)

		targets = append(targets, edgegrid.TrafficTarget{
			Name:         d.Get(prefix + ".name").(string),
			Weight:       d.Get(prefix + ".weight").(float64),
			Enabled:      d.Get(prefix + ".enabled").(bool),
			Servers:      stringSetToStringSlice(d.Get(prefix + ".servers").(*schema.Set)),
			DataCenterID: d.Get(prefix + ".data_center_id").(int),
		})
	}

	return targets
}

func livenessTests(d *schema.ResourceData) []edgegrid.LivenessTest {
	tests := []edgegrid.LivenessTest{}
	testsCount := d.Get("liveness_test.#").(int)

	for i := 0; i < testsCount; i++ {
		prefix := fmt.Sprintf("liveness_test.%d", i)

		tests = append(tests, edgegrid.LivenessTest{
			Name:                          d.Get(prefix + ".name").(string),
			TestInterval:                  int64(d.Get(prefix + ".test_interval").(int)),
			HTTPError3xx:                  d.Get(prefix + ".http_error_3xx").(bool),
			HTTPError4xx:                  d.Get(prefix + ".http_error_4xx").(bool),
			HTTPError5xx:                  d.Get(prefix + ".http_error_5xx").(bool),
			TestObjectPort:                int64(d.Get(prefix + ".test_object_port").(int)),
			TestTimeout:                   d.Get(prefix + ".test_timeout").(float64),
			TestObject:                    d.Get(prefix + ".test_object").(string),
			TestObjectProtocol:            d.Get(prefix + ".test_object_protocol").(string),
			DisableNonstandardPortWarning: d.Get(prefix + ".disable_nonstandard_port_warning").(bool),
		})
	}

	return tests
}

func getServers(prefix string, d *schema.ResourceData) []string {
	servers := []string{}
	serversPrefix := d.Get(prefix + ".servers")
	serversCount := d.Get(fmt.Sprintf("%d.#", serversPrefix)).(int)

	for i := 0; i < serversCount; i++ {
		serverPrefix := fmt.Sprintf("%d.%s", serversPrefix, string(i))
		servers = append(servers, d.Get(serverPrefix).(string))
	}

	return servers
}

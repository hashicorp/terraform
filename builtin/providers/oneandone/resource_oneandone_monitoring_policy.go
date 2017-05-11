package oneandone

import (
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceOneandOneMonitoringPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceOneandOneMonitoringPolicyCreate,
		Read:   resourceOneandOneMonitoringPolicyRead,
		Update: resourceOneandOneMonitoringPolicyUpdate,
		Delete: resourceOneandOneMonitoringPolicyDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"email": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"agent": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"thresholds": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"warning": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
									"critical": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
								},
							},
							Required: true,
						},
						"ram": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"warning": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
									"critical": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
								},
							},
							Required: true,
						},
						"disk": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"warning": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
									"critical": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
								},
							},
							Required: true,
						},
						"transfer": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"warning": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
									"critical": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
								},
							},
							Required: true,
						},
						"internal_ping": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"warning": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
									"critical": {
										Type: schema.TypeSet,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"value": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"alert": {
													Type:     schema.TypeBool,
													Required: true,
												},
											},
										},
										Required: true,
									},
								},
							},
							Required: true,
						},
					},
				},
				Required: true,
			},
			"ports": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email_notification": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"alert_if": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Optional: true,
			},
			"processes": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"email_notification": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"process": {
							Type:     schema.TypeString,
							Required: true,
						},
						"alert_if": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Optional: true,
			},
		},
	}
}

func resourceOneandOneMonitoringPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	mp_request := oneandone.MonitoringPolicy{
		Name:       d.Get("name").(string),
		Agent:      d.Get("agent").(bool),
		Thresholds: getThresholds(d.Get("thresholds")),
	}

	if raw, ok := d.GetOk("ports"); ok {
		mp_request.Ports = getPorts(raw)
	}

	if raw, ok := d.GetOk("processes"); ok {
		mp_request.Processes = getProcesses(raw)
	}

	mp_id, mp, err := config.API.CreateMonitoringPolicy(&mp_request)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
	if err != nil {
		return err
	}

	d.SetId(mp_id)

	return resourceOneandOneMonitoringPolicyRead(d, meta)
}

func resourceOneandOneMonitoringPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req := oneandone.MonitoringPolicy{}
	if d.HasChange("name") {
		_, n := d.GetChange("name")
		req.Name = n.(string)
	}

	if d.HasChange("description") {
		_, n := d.GetChange("description")
		req.Description = n.(string)
	}

	if d.HasChange("email") {
		_, n := d.GetChange("email")
		req.Email = n.(string)
	}

	if d.HasChange("agent") {
		_, n := d.GetChange("agent")
		req.Agent = n.(bool)
	}

	if d.HasChange("thresholds") {
		_, n := d.GetChange("thresholds")
		req.Thresholds = getThresholds(n)
	}

	mp, err := config.API.UpdateMonitoringPolicy(d.Id(), &req)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
	if err != nil {
		return err
	}

	if d.HasChange("ports") {
		o, n := d.GetChange("ports")
		oldValues := o.([]interface{})
		newValues := n.([]interface{})

		if len(newValues) > len(oldValues) {
			ports := getPorts(newValues)

			newports := []oneandone.MonitoringPort{}

			for _, p := range ports {
				if p.Id == "" {
					newports = append(newports, p)
				}
			}

			mp, err := config.API.AddMonitoringPolicyPorts(d.Id(), newports)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}
		} else if len(oldValues) > len(newValues) {
			diff := difference(oldValues, newValues)
			ports := getPorts(diff)

			for _, port := range ports {
				if port.Id == "" {
					continue
				}

				mp, err := config.API.DeleteMonitoringPolicyPort(d.Id(), port.Id)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		} else if len(oldValues) == len(newValues) {
			ports := getPorts(newValues)

			for _, port := range ports {
				mp, err := config.API.ModifyMonitoringPolicyPort(d.Id(), port.Id, &port)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		}
	}

	if d.HasChange("processes") {
		o, n := d.GetChange("processes")
		oldValues := o.([]interface{})
		newValues := n.([]interface{})
		if len(newValues) > len(oldValues) {
			processes := getProcesses(newValues)

			newprocesses := []oneandone.MonitoringProcess{}

			for _, p := range processes {
				if p.Id == "" {
					newprocesses = append(newprocesses, p)
				}
			}

			mp, err := config.API.AddMonitoringPolicyProcesses(d.Id(), newprocesses)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}
		} else if len(oldValues) > len(newValues) {
			diff := difference(oldValues, newValues)
			processes := getProcesses(diff)
			for _, process := range processes {
				if process.Id == "" {
					continue
				}

				mp, err := config.API.DeleteMonitoringPolicyProcess(d.Id(), process.Id)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		} else if len(oldValues) == len(newValues) {
			processes := getProcesses(newValues)

			for _, process := range processes {
				mp, err := config.API.ModifyMonitoringPolicyProcess(d.Id(), process.Id, &process)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		}
	}

	return resourceOneandOneMonitoringPolicyRead(d, meta)
}

func resourceOneandOneMonitoringPolicyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	mp, err := config.API.GetMonitoringPolicy(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	if len(mp.Servers) > 0 {
	}

	if len(mp.Ports) > 0 {
		pports := d.Get("ports").([]interface{})
		for i, raw_ports := range pports {
			port := raw_ports.(map[string]interface{})
			port["id"] = mp.Ports[i].Id
		}
		d.Set("ports", pports)
	}

	if len(mp.Processes) > 0 {
		pprocesses := d.Get("processes").([]interface{})
		for i, raw_processes := range pprocesses {
			process := raw_processes.(map[string]interface{})
			process["id"] = mp.Processes[i].Id
		}
		d.Set("processes", pprocesses)
	}

	return nil
}

func resourceOneandOneMonitoringPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	mp, err := config.API.DeleteMonitoringPolicy(d.Id())
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(mp)
	if err != nil {
		return err
	}

	return nil
}

func getThresholds(d interface{}) *oneandone.MonitoringThreshold {
	raw_thresholds := d.(*schema.Set).List()

	toReturn := &oneandone.MonitoringThreshold{}

	for _, thresholds := range raw_thresholds {
		th_set := thresholds.(map[string]interface{})

		//CPU
		cpu_raw := th_set["cpu"].(*schema.Set)
		toReturn.Cpu = &oneandone.MonitoringLevel{}
		for _, c := range cpu_raw.List() {
			int_k := c.(map[string]interface{})
			for _, w := range int_k["warning"].(*schema.Set).List() {
				toReturn.Cpu.Warning = &oneandone.MonitoringValue{
					Value: w.(map[string]interface{})["value"].(int),
					Alert: w.(map[string]interface{})["alert"].(bool),
				}
			}

			for _, c := range int_k["critical"].(*schema.Set).List() {
				toReturn.Cpu.Critical = &oneandone.MonitoringValue{
					Value: c.(map[string]interface{})["value"].(int),
					Alert: c.(map[string]interface{})["alert"].(bool),
				}
			}
		}
		//RAM
		ram_raw := th_set["ram"].(*schema.Set)
		toReturn.Ram = &oneandone.MonitoringLevel{}
		for _, c := range ram_raw.List() {
			int_k := c.(map[string]interface{})
			for _, w := range int_k["warning"].(*schema.Set).List() {
				toReturn.Ram.Warning = &oneandone.MonitoringValue{
					Value: w.(map[string]interface{})["value"].(int),
					Alert: w.(map[string]interface{})["alert"].(bool),
				}
			}

			for _, c := range int_k["critical"].(*schema.Set).List() {
				toReturn.Ram.Critical = &oneandone.MonitoringValue{
					Value: c.(map[string]interface{})["value"].(int),
					Alert: c.(map[string]interface{})["alert"].(bool),
				}
			}
		}

		//DISK
		disk_raw := th_set["disk"].(*schema.Set)
		toReturn.Disk = &oneandone.MonitoringLevel{}
		for _, c := range disk_raw.List() {
			int_k := c.(map[string]interface{})
			for _, w := range int_k["warning"].(*schema.Set).List() {
				toReturn.Disk.Warning = &oneandone.MonitoringValue{
					Value: w.(map[string]interface{})["value"].(int),
					Alert: w.(map[string]interface{})["alert"].(bool),
				}
			}

			for _, c := range int_k["critical"].(*schema.Set).List() {
				toReturn.Disk.Critical = &oneandone.MonitoringValue{
					Value: c.(map[string]interface{})["value"].(int),
					Alert: c.(map[string]interface{})["alert"].(bool),
				}
			}
		}

		//TRANSFER
		transfer_raw := th_set["transfer"].(*schema.Set)
		toReturn.Transfer = &oneandone.MonitoringLevel{}
		for _, c := range transfer_raw.List() {
			int_k := c.(map[string]interface{})
			for _, w := range int_k["warning"].(*schema.Set).List() {
				toReturn.Transfer.Warning = &oneandone.MonitoringValue{
					Value: w.(map[string]interface{})["value"].(int),
					Alert: w.(map[string]interface{})["alert"].(bool),
				}
			}

			for _, c := range int_k["critical"].(*schema.Set).List() {
				toReturn.Transfer.Critical = &oneandone.MonitoringValue{
					Value: c.(map[string]interface{})["value"].(int),
					Alert: c.(map[string]interface{})["alert"].(bool),
				}
			}
		}
		//internal ping
		ping_raw := th_set["internal_ping"].(*schema.Set)
		toReturn.InternalPing = &oneandone.MonitoringLevel{}
		for _, c := range ping_raw.List() {
			int_k := c.(map[string]interface{})
			for _, w := range int_k["warning"].(*schema.Set).List() {
				toReturn.InternalPing.Warning = &oneandone.MonitoringValue{
					Value: w.(map[string]interface{})["value"].(int),
					Alert: w.(map[string]interface{})["alert"].(bool),
				}
			}

			for _, c := range int_k["critical"].(*schema.Set).List() {
				toReturn.InternalPing.Critical = &oneandone.MonitoringValue{
					Value: c.(map[string]interface{})["value"].(int),
					Alert: c.(map[string]interface{})["alert"].(bool),
				}
			}
		}
	}

	return toReturn
}

func getProcesses(d interface{}) []oneandone.MonitoringProcess {
	toReturn := []oneandone.MonitoringProcess{}

	for _, raw := range d.([]interface{}) {
		port := raw.(map[string]interface{})
		m_port := oneandone.MonitoringProcess{
			EmailNotification: port["email_notification"].(bool),
		}

		if port["id"] != nil {
			m_port.Id = port["id"].(string)
		}

		if port["process"] != nil {
			m_port.Process = port["process"].(string)
		}

		if port["alert_if"] != nil {
			m_port.AlertIf = port["alert_if"].(string)
		}

		toReturn = append(toReturn, m_port)
	}

	return toReturn
}

func getPorts(d interface{}) []oneandone.MonitoringPort {
	toReturn := []oneandone.MonitoringPort{}

	for _, raw := range d.([]interface{}) {
		port := raw.(map[string]interface{})
		m_port := oneandone.MonitoringPort{
			EmailNotification: port["email_notification"].(bool),
			Port:              port["port"].(int),
		}

		if port["id"] != nil {
			m_port.Id = port["id"].(string)
		}

		if port["protocol"] != nil {
			m_port.Protocol = port["protocol"].(string)
		}

		if port["alert_if"] != nil {
			m_port.AlertIf = port["alert_if"].(string)
		}

		toReturn = append(toReturn, m_port)
	}

	return toReturn
}

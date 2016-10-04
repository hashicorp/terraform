package cloudstack

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackSecurityGroupCreate,
		Read:   resourceCloudStackSecurityGroupRead,
		Delete: resourceCloudStackSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"rules": &schema.Schema{
				Type:     schema.TypeList,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_list": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"ports": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"traffic_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "ingress",
						},
					},
				},
			},
		},
	}
}

func resourceCloudStackSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	name := d.Get("name").(string)

	// Create a new parameter struct
	p := cs.SecurityGroup.NewCreateSecurityGroupParams(name)

	// Set the description
	if description, ok := d.GetOk("description"); ok {
		p.SetDescription(description.(string))
	} else {
		p.SetDescription(name)
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating security group %s", name)
	r, err := cs.SecurityGroup.CreateSecurityGroup(p)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Security group %s successfully created with ID: %s", name, r.Id)
	d.SetId(r.Id)

	// Create Rules
	rules := d.Get("rules").([]interface{})
	for _, r := range rules {

		rule := r.(map[string]interface{})

		m := splitPorts.FindStringSubmatch(rule["ports"].(string))
		startPort, err := strconv.Atoi(m[1])
		if err != nil {
			return err
		}

		endPort := startPort
		if m[2] != "" {
			endPort, err = strconv.Atoi(m[2])
			if err != nil {
				return err
			}
		}

		traffic := rule["traffic_type"].(string)
		if traffic == "ingress" {
			param := cs.SecurityGroup.NewAuthorizeSecurityGroupIngressParams()

			param.SetSecuritygroupid(d.Id())
			if cidrlist := rule["cidr_list"]; cidrlist != "" {
				param.SetCidrlist([]string{cidrlist.(string)})
				log.Printf("[DEBUG] cidr = %v", cidrlist)
			}
			if securitygroup := rule["security_group"]; securitygroup != "" {
				// Get the security group details
				ag, count, err := cs.SecurityGroup.GetSecurityGroupByName(
					securitygroup.(string),
					cloudstack.WithProject(d.Get("project").(string)),
				)
				if err != nil {
					if count == 0 {
						log.Printf("[DEBUG] Security group %s does not longer exist", d.Get("name").(string))
						d.SetId("")
						return nil
					}

					log.Printf("[DEBUG] Found %v groups matching", count)

					return err
				}
				log.Printf("[DEBUG] ag = %v", ag)
				m := make(map[string]string)
				m[ag.Account] = ag.Name
				log.Printf("[DEBUG] m = %v", m)
				param.SetUsersecuritygrouplist(m)
			}
			param.SetStartport(startPort)
			param.SetEndport(endPort)
			param.SetProtocol(rule["protocol"].(string))

			log.Printf("[DEBUG] Authorizing Ingress Rule %#v", param)
			_, err := cs.SecurityGroup.AuthorizeSecurityGroupIngress(param)
			if err != nil {
				return err
			}
		} else if traffic == "egress" {
			param := cs.SecurityGroup.NewAuthorizeSecurityGroupEgressParams()

			param.SetSecuritygroupid(d.Id())
			if cidrlist := rule["cidr_list"]; cidrlist != "" {
				param.SetCidrlist([]string{cidrlist.(string)})
				log.Printf("[DEBUG] cidr = %v", cidrlist)
			}
			if securitygroup := rule["security_group"]; securitygroup != "" {
				// Get the security group details
				ag, count, err := cs.SecurityGroup.GetSecurityGroupByName(
					securitygroup.(string),
					cloudstack.WithProject(d.Get("project").(string)),
				)
				if err != nil {
					if count == 0 {
						log.Printf("[DEBUG] Security group %s does not longer exist", d.Get("name").(string))
						d.SetId("")
						return nil
					}

					log.Printf("[DEBUG] Found %v groups matching", count)

					return err
				}
				log.Printf("[DEBUG] ag = %v", ag)
				m := make(map[string]string)
				m[ag.Account] = ag.Name
				log.Printf("[DEBUG] m = %v", m)
				param.SetUsersecuritygrouplist(m)
			}
			param.SetStartport(startPort)
			param.SetEndport(endPort)
			param.SetProtocol(rule["protocol"].(string))

			log.Printf("[DEBUG] Authorizing Egress Rule %#v", param)
			_, err := cs.SecurityGroup.AuthorizeSecurityGroupEgress(param)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf(
				"Parameter traffic_type only accepts 'ingress' or 'egress' as values")
		}
	}

	return resourceCloudStackSecurityGroupRead(d, meta)
}

func resourceCloudStackSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	log.Printf("[DEBUG] Retrieving security group %s (ID=%s)", d.Get("name").(string), d.Id())

	// Get the security group details
	ag, count, err := cs.SecurityGroup.GetSecurityGroupByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Security group %s does not longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		log.Printf("[DEBUG] Found %v groups matching", count)

		return err
	}

	// Update the config
	d.Set("name", ag.Name)
	d.Set("description", ag.Description)

	var rules []interface{}
	for _, r := range ag.Ingressrule {
		var ports string
		if r.Startport == r.Endport {
			ports = strconv.Itoa(r.Startport)
		} else {
			ports = fmt.Sprintf("%v-%v", r.Startport, r.Endport)
		}
		rule := map[string]interface{}{
			"cidr_list":      r.Cidr,
			"ports":          ports,
			"protocol":       r.Protocol,
			"traffic_type":   "ingress",
			"security_group": r.Securitygroupname,
		}
		rules = append(rules, rule)
	}
	for _, r := range ag.Egressrule {
		var ports string
		if r.Startport == r.Endport {
			ports = strconv.Itoa(r.Startport)
		} else {
			ports = fmt.Sprintf("%s-%s", r.Startport, r.Endport)
		}
		rule := map[string]interface{}{
			"cidr_list":      r.Cidr,
			"ports":          ports,
			"protocol":       r.Protocol,
			"traffic_type":   "egress",
			"security_group": r.Securitygroupname,
		}
		rules = append(rules, rule)
	}
	d.Set("rules", rules)

	return nil
}

func resourceCloudStackSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.SecurityGroup.NewDeleteSecurityGroupParams()
	p.SetId(d.Id())

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// Delete the security group
	_, err := cs.SecurityGroup.DeleteSecurityGroup(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting security group: %s", err)
	}

	return nil
}

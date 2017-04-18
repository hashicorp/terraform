package alicloud

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceAliyunSlbAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSlbAttachmentCreate,
		Read:   resourceAliyunSlbAttachmentRead,
		Update: resourceAliyunSlbAttachmentUpdate,
		Delete: resourceAliyunSlbAttachmentDelete,

		Schema: map[string]*schema.Schema{

			"slb_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"instances": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				Set:      schema.HashString,
			},

			"backend_servers": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAliyunSlbAttachmentCreate(d *schema.ResourceData, meta interface{}) error {

	slbId := d.Get("slb_id").(string)

	slbconn := meta.(*AliyunClient).slbconn

	loadBalancer, err := slbconn.DescribeLoadBalancerAttribute(slbId)
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return fmt.Errorf("Special SLB Id not found: %#v", err)
		}

		return err
	}

	d.SetId(loadBalancer.LoadBalancerId)

	return resourceAliyunSlbAttachmentUpdate(d, meta)
}

func resourceAliyunSlbAttachmentRead(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn
	loadBalancer, err := slbconn.DescribeLoadBalancerAttribute(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Read special SLB Id not found: %#v", err)
	}

	if loadBalancer == nil {
		d.SetId("")
		return nil
	}

	backendServerType := loadBalancer.BackendServers
	servers := backendServerType.BackendServer
	instanceIds := make([]string, 0, len(servers))
	if len(servers) > 0 {
		for _, e := range servers {
			instanceIds = append(instanceIds, e.ServerId)
		}
		if err != nil {
			return err
		}
	}

	d.Set("slb_id", d.Id())
	d.Set("instances", instanceIds)
	d.Set("backend_servers", strings.Join(instanceIds, ","))

	return nil
}

func resourceAliyunSlbAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn
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

	}

	return resourceAliyunSlbAttachmentRead(d, meta)

}

func resourceAliyunSlbAttachmentDelete(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn
	o := d.Get("instances")
	os := o.(*schema.Set)
	remove := expandBackendServers(os.List())

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

	return nil
}

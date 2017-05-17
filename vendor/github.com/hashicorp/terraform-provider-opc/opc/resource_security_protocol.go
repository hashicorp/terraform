package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCSecurityProtocol() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCSecurityProtocolCreate,
		Read:   resourceOPCSecurityProtocolRead,
		Update: resourceOPCSecurityProtocolUpdate,
		Delete: resourceOPCSecurityProtocolDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"dst_ports": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ip_protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(compute.All),
				ValidateFunc: validateIPProtocol,
			},
			"src_ports": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsForceNewSchema(),
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCSecurityProtocolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityProtocols()
	input := compute.CreateSecurityProtocolInput{
		Name:       d.Get("name").(string),
		IPProtocol: d.Get("ip_protocol").(string),
	}
	dstPorts := getStringList(d, "dst_ports")
	if len(dstPorts) != 0 {
		input.DstPortSet = dstPorts
	}
	srcPorts := getStringList(d, "src_ports")
	if len(srcPorts) != 0 {
		input.SrcPortSet = srcPorts
	}
	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateSecurityProtocol(&input)
	if err != nil {
		return fmt.Errorf("Error creating Security Protocol: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCSecurityProtocolRead(d, meta)
}

func resourceOPCSecurityProtocolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityProtocols()
	getInput := compute.GetSecurityProtocolInput{
		Name: d.Id(),
	}
	result, err := client.GetSecurityProtocol(&getInput)
	if err != nil {
		// Security Protocol does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security protocol %s: %s", d.Id(), err)
	}

	d.Set("name", result.Name)
	d.Set("ip_protocol", result.IPProtocol)
	d.Set("description", result.Description)
	if err := setStringList(d, "dst_ports", result.DstPortSet); err != nil {
		return err
	}
	if err := setStringList(d, "src_ports", result.SrcPortSet); err != nil {
		return err
	}
	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCSecurityProtocolUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityProtocols()
	input := compute.UpdateSecurityProtocolInput{
		Name:       d.Get("name").(string),
		IPProtocol: d.Get("ip_protocol").(string),
	}
	dstPorts := getStringList(d, "dst_ports")
	if len(dstPorts) != 0 {
		input.DstPortSet = dstPorts
	}
	srcPorts := getStringList(d, "src_ports")
	if len(srcPorts) != 0 {
		input.SrcPortSet = srcPorts
	}
	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.UpdateSecurityProtocol(&input)
	if err != nil {
		return fmt.Errorf("Error updating Security Protocol: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCSecurityProtocolRead(d, meta)
}

func resourceOPCSecurityProtocolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityProtocols()
	name := d.Id()

	input := compute.DeleteSecurityProtocolInput{
		Name: name,
	}
	if err := client.DeleteSecurityProtocol(&input); err != nil {
		return fmt.Errorf("Error deleting Security Protocol: %s", err)
	}
	return nil
}

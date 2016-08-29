package occi

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceVirtualMachineCreate,
		Delete: resourceVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"x509": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"image_template": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"resource_template": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	var (
		cmdOut []byte
		err    error
	)
	endpoint := d.Get("endpoint").(string)
	image_template := d.Get("image_template").(string)
	resource_template := d.Get("resource_template").(string)
	proxy_file := d.Get("x509").(string)
	vm_name := d.Get("name").(string)

	cmd_name := "occi"
	cmd_args := []string{"-e", endpoint, "--auth", "x509", "--user-cred", proxy_file, "--voms", "-a", "create", "-r", "compute", "--mixin", image_template, "--mixin", resource_template, "--attribute", vm_name}

	if cmdOut, err = exec.Command(cmd_name, cmd_args...).Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return err
	}

	cmd_out_string := string(cmdOut)
	fmt.Println(cmd_out_string)

	return nil
}

func resourceVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

package occi

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceVirtualMachineCreate,
		Read:   resourceVirtualMachineRead,
		Update: resourceVirtualMachineUpdate,
		Delete: resourceVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"x509": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"image_template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"resource_template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"init_file": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"storage_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"vm_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	var (
		cmdOut []byte
		err    error
	)

	// arguments for VM creation
	endpoint := d.Get("endpoint").(string)
	image_template := d.Get("image_template").(string)
	resource_template := d.Get("resource_template").(string)
	proxy_file := d.Get("x509").(string)
	vm_name := d.Get("name").(string)
	init_file := d.Get("init_file").(string)
	network := d.Get("network").(string)

	// create VM
	cmd_name := "occi"
	cmd_args_create := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "create", "-r", "compute", "-M", image_template, "-M", resource_template, "-t", strings.Join([]string{"occi.core.title=", vm_name}, ""), "-T", strings.Join([]string{"user_data=file:///", init_file}, ""), "-w", "3600"}

	if len(network) > 0 {
		cmd_args_create = append(cmd_args_create, "-j")
		cmd_args_create = append(cmd_args_create, network)
	}

	log.Printf("[DEBUG] OCCI command args: %s", cmd_args_create)
	log.Printf("[INFO] Creating VM with image %s and resource template %s", image_template, resource_template)

	if cmdOut, err = exec.Command(cmd_name, cmd_args_create...).CombinedOutput(); err != nil {
		return fmt.Errorf("Error while creating virtual machine: %s", cmdOut)
	}
	compute := strings.Trim(string(cmdOut), "\n")
	d.Set("vm_id", compute)
	d.SetId(compute)

	log.Printf("[INFO] VM was created with ID %s", compute)
	// get IP address
	cmd_args_describe := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "describe", "-r", compute}

	if cmdOut, err = exec.Command(cmd_name, cmd_args_describe...).CombinedOutput(); err != nil {
		return fmt.Errorf("Error while trying to get IP address: %s", cmdOut)
	}
	byte_array := bytes.Fields(cmdOut)
	for i, line := range byte_array {
		if bytes.Contains(line, []byte("occi.networkinterface.address")) {
			ip_line := string(byte_array[i+2][:])
			d.Set("ip_address", ip_line)
			log.Printf("[INFO] IP address of VM: %s", ip_line)
			break
		}
	}
	// if storage variable is set, create storage
	storage_size := d.Get("storage_size").(int)
	if storage_size > 0 {
		log.Printf("[INFO] Linking storage with size %v", storage_size)
		storage_params := strings.Join([]string{"occi.storage.size=", "'num(", strconv.Itoa(storage_size), ")',occi.core.title=storage_terraform", "_", strings.Split(compute, "/")[len(strings.Split(compute, "/")) - 1]}, "")
		cmd_args_storage := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "create", "-r", "storage", "-t", storage_params, "-w", "3600"}
		if cmdOut, err = exec.Command(cmd_name, cmd_args_storage...).CombinedOutput(); err != nil {
			return fmt.Errorf("Error while creating storage: %s", cmdOut)
		}
		storage_id := strings.Trim(string(cmdOut), "\n")
		d.Set("storage_id", storage_id)

		// link storage to VM
		cmd_args_storage_link := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "link", "-r", compute, "-j", storage_id}
		if cmdOut, err = exec.Command(cmd_name, cmd_args_storage_link...).CombinedOutput(); err != nil {
			return fmt.Errorf("Error while linking storage %s to VM %s: %s", compute, storage_id, cmdOut)
		}
		d.Set("storage_link", strings.Trim(string(cmdOut), "\n"))
	}

	return nil
}

func resourceVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	var (
		cmdOut []byte
		err    error
	)
	endpoint := d.Get("endpoint").(string)
	proxy_file := d.Get("x509").(string)
	vm_id := d.Get("vm_id").(string)
	storage_id := d.Get("storage_id").(string)
	storage_link := d.Get("storage_link").(string)
	cmd_name := "occi"

	// if storage is provisioned, unlink from VM
	if storage_link != "" {
		cmd_args_unlink := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "unlink", "-r", storage_link}
		if cmdOut, err = exec.Command(cmd_name, cmd_args_unlink...).CombinedOutput(); err != nil {
			return fmt.Errorf("Error while unlinking storage %s: %s", storage_link, cmdOut)
		}
	}
	// destroy VM
	log.Printf("[INFO] Destroying VM with ID %s", vm_id)
	cmd_args := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "delete", "-r", vm_id}
	if cmdOut, err = exec.Command(cmd_name, cmd_args...).CombinedOutput(); err != nil {
		return fmt.Errorf("Error while destroying VM %s: %s", vm_id, cmdOut)
	}

	log.Printf("[INFO] Destroying storage with ID %s", storage_id)
	// if storage has been provisioned, destroy it too
	if storage_id != "" {
		cmd_args_storage := []string{"-e", endpoint, "-n", "x509", "-x", proxy_file, "-X", "-a", "delete", "-r", storage_id}
		if cmdOut, err = exec.Command(cmd_name, cmd_args_storage...).CombinedOutput(); err != nil {
			return fmt.Errorf("Error while destroying storage %s: %s", storage_id, cmdOut)
		}
	}
	return nil
}

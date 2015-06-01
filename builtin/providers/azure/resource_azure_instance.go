package azure

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/svanharmelen/azure-sdk-for-go/management"
	"github.com/svanharmelen/azure-sdk-for-go/management/hostedservice"
	"github.com/svanharmelen/azure-sdk-for-go/management/osimage"
	"github.com/svanharmelen/azure-sdk-for-go/management/virtualmachine"
	"github.com/svanharmelen/azure-sdk-for-go/management/virtualmachineimage"
	"github.com/svanharmelen/azure-sdk-for-go/management/vmutils"
)

const (
	linux                = "Linux"
	windows              = "Windows"
	osDiskBlobStorageURL = "http://%s.blob.core.windows.net/vhds/%s.vhd"
)

func resourceAzureInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureInstanceCreate,
		Read:   resourceAzureInstanceRead,
		Update: resourceAzureInstanceUpdate,
		Delete: resourceAzureInstanceDelete,

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

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"subnet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_network": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"storage": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"reverse_dns": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"automatic_updates": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ssh_key_thumbprint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "tcp",
						},

						"public_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"private_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceAzureEndpointHash,
			},

			"security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"vip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAzureInstanceCreate(d *schema.ResourceData, meta interface{}) (err error) {
	mc := meta.(*Client).mgmtClient

	name := d.Get("name").(string)

	// Compute/set the description
	description := d.Get("description").(string)
	if description == "" {
		description = name
	}

	// Retrieve the needed details of the image
	configureForImage, osType, err := retrieveImageDetails(
		mc,
		d.Get("image").(string),
		name,
		d.Get("storage").(string),
	)
	if err != nil {
		return err
	}

	// Verify if we have all required parameters
	if err := verifyInstanceParameters(d, osType); err != nil {
		return err
	}

	p := hostedservice.CreateHostedServiceParameters{
		ServiceName:    name,
		Label:          base64.StdEncoding.EncodeToString([]byte(name)),
		Description:    fmt.Sprintf("Cloud Service created automatically for instance %s", name),
		Location:       d.Get("location").(string),
		ReverseDNSFqdn: d.Get("reverse_dns").(string),
	}

	log.Printf("[DEBUG] Creating Cloud Service for instance: %s", name)
	err = hostedservice.NewClient(mc).CreateHostedService(p)
	if err != nil {
		return fmt.Errorf("Error creating Cloud Service for instance %s: %s", name, err)
	}

	// Put in this defer here, so we are sure to cleanup already created parts
	// when we exit with an error
	defer func(mc management.Client) {
		if err != nil {
			req, err := hostedservice.NewClient(mc).DeleteHostedService(name, true)
			if err != nil {
				log.Printf("[DEBUG] Error cleaning up Cloud Service of instance %s: %s", name, err)
			}

			// Wait until the Cloud Service is deleted
			if err := mc.WaitForOperation(req, nil); err != nil {
				log.Printf(
					"[DEBUG] Error waiting for Cloud Service of instance %s to be deleted: %s", name, err)
			}
		}
	}(mc)

	// Create a new role for the instance
	role := vmutils.NewVMConfiguration(name, d.Get("size").(string))

	log.Printf("[DEBUG] Configuring deployment from image...")
	err = configureForImage(&role)
	if err != nil {
		return fmt.Errorf("Error configuring the deployment for %s: %s", name, err)
	}

	if osType == linux {
		// This is pretty ugly, but the Azure SDK leaves me no other choice...
		if tp, ok := d.GetOk("ssh_key_thumbprint"); ok {
			err = vmutils.ConfigureForLinux(
				&role,
				name,
				d.Get("username").(string),
				d.Get("password").(string),
				tp.(string),
			)
		} else {
			err = vmutils.ConfigureForLinux(
				&role,
				name,
				d.Get("username").(string),
				d.Get("password").(string),
			)
		}
		if err != nil {
			return fmt.Errorf("Error configuring %s for Linux: %s", name, err)
		}
	}

	if osType == windows {
		err = vmutils.ConfigureForWindows(
			&role,
			name,
			d.Get("username").(string),
			d.Get("password").(string),
			d.Get("automatic_updates").(bool),
			d.Get("time_zone").(string),
		)
		if err != nil {
			return fmt.Errorf("Error configuring %s for Windows: %s", name, err)
		}
	}

	if s := d.Get("endpoint").(*schema.Set); s.Len() > 0 {
		for _, v := range s.List() {
			m := v.(map[string]interface{})
			err := vmutils.ConfigureWithExternalPort(
				&role,
				m["name"].(string),
				m["private_port"].(int),
				m["public_port"].(int),
				endpointProtocol(m["protocol"].(string)),
			)
			if err != nil {
				return fmt.Errorf(
					"Error adding endpoint %s for instance %s: %s", m["name"].(string), name, err)
			}
		}
	}

	if subnet, ok := d.GetOk("subnet"); ok {
		err = vmutils.ConfigureWithSubnet(&role, subnet.(string))
		if err != nil {
			return fmt.Errorf(
				"Error associating subnet %s with instance %s: %s", d.Get("subnet").(string), name, err)
		}
	}

	if sg, ok := d.GetOk("security_group"); ok {
		err = vmutils.ConfigureWithSecurityGroup(&role, sg.(string))
		if err != nil {
			return fmt.Errorf(
				"Error associating security group %s with instance %s: %s", sg.(string), name, err)
		}
	}

	options := virtualmachine.CreateDeploymentOptions{
		VirtualNetworkName: d.Get("virtual_network").(string),
	}

	log.Printf("[DEBUG] Creating the new instance...")
	req, err := virtualmachine.NewClient(mc).CreateDeployment(role, name, options)
	if err != nil {
		return fmt.Errorf("Error creating instance %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for the new instance to be created...")
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for instance %s to be created: %s", name, err)
	}

	d.SetId(name)

	return resourceAzureInstanceRead(d, meta)
}

func resourceAzureInstanceRead(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	log.Printf("[DEBUG] Retrieving Cloud Service for instance: %s", d.Id())
	cs, err := hostedservice.NewClient(mc).GetHostedService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving Cloud Service of instance %s: %s", d.Id(), err)
	}

	d.Set("reverse_dns", cs.ReverseDNSFqdn)
	d.Set("location", cs.Location)

	log.Printf("[DEBUG] Retrieving instance: %s", d.Id())
	dpmt, err := virtualmachine.NewClient(mc).GetDeployment(d.Id(), d.Id())
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving instance %s: %s", d.Id(), err)
	}

	if len(dpmt.RoleList) != 1 {
		return fmt.Errorf(
			"Instance %s has an unexpected number of roles: %d", d.Id(), len(dpmt.RoleList))
	}

	d.Set("size", dpmt.RoleList[0].RoleSize)

	if len(dpmt.RoleInstanceList) != 1 {
		return fmt.Errorf(
			"Instance %s has an unexpected number of role instances: %d",
			d.Id(), len(dpmt.RoleInstanceList))
	}
	d.Set("ip_address", dpmt.RoleInstanceList[0].IPAddress)

	if len(dpmt.RoleInstanceList[0].InstanceEndpoints) > 0 {
		d.Set("vip_address", dpmt.RoleInstanceList[0].InstanceEndpoints[0].Vip)
	}

	// Find the network configuration set
	for _, c := range dpmt.RoleList[0].ConfigurationSets {
		if c.ConfigurationSetType == virtualmachine.ConfigurationSetTypeNetwork {
			// Create a new set to hold all configured endpoints
			endpoints := &schema.Set{
				F: resourceAzureEndpointHash,
			}

			// Loop through all endpoints
			for _, ep := range c.InputEndpoints {
				endpoint := map[string]interface{}{}

				// Update the values
				endpoint["name"] = ep.Name
				endpoint["protocol"] = string(ep.Protocol)
				endpoint["public_port"] = ep.Port
				endpoint["private_port"] = ep.LocalPort
				endpoints.Add(endpoint)
			}
			d.Set("endpoint", endpoints)

			// Update the subnet
			switch len(c.SubnetNames) {
			case 1:
				d.Set("subnet", c.SubnetNames[0])
			case 0:
				d.Set("subnet", "")
			default:
				return fmt.Errorf(
					"Instance %s has an unexpected number of associated subnets %d",
					d.Id(), len(dpmt.RoleInstanceList))
			}

			// Update the security group
			d.Set("security_group", c.NetworkSecurityGroup)
		}
	}

	connType := "ssh"
	if dpmt.RoleList[0].OSVirtualHardDisk.OS == windows {
		connType = "winrm"
	}

	// Set the connection info for any configured provisioners
	d.SetConnInfo(map[string]string{
		"type":     connType,
		"host":     dpmt.VirtualIPs[0].Address,
		"user":     d.Get("username").(string),
		"password": d.Get("password").(string),
	})

	return nil
}

func resourceAzureInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	// First check if anything we can update changed, and if not just return
	if !d.HasChange("size") && !d.HasChange("endpoint") && !d.HasChange("security_group") {
		return nil
	}

	// Get the current role
	role, err := virtualmachine.NewClient(mc).GetRole(d.Id(), d.Id(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving role of instance %s: %s", d.Id(), err)
	}

	// Verify if we have all required parameters
	if err := verifyInstanceParameters(d, role.OSVirtualHardDisk.OS); err != nil {
		return err
	}

	if d.HasChange("size") {
		role.RoleSize = d.Get("size").(string)
	}

	if d.HasChange("endpoint") {
		_, n := d.GetChange("endpoint")

		// Delete the existing endpoints
		for i, c := range role.ConfigurationSets {
			if c.ConfigurationSetType == virtualmachine.ConfigurationSetTypeNetwork {
				c.InputEndpoints = nil
				role.ConfigurationSets[i] = c
			}
		}

		// And add the ones we still want
		if s := n.(*schema.Set); s.Len() > 0 {
			for _, v := range s.List() {
				m := v.(map[string]interface{})
				err := vmutils.ConfigureWithExternalPort(
					role,
					m["name"].(string),
					m["private_port"].(int),
					m["public_port"].(int),
					endpointProtocol(m["protocol"].(string)),
				)
				if err != nil {
					return fmt.Errorf(
						"Error adding endpoint %s for instance %s: %s", m["name"].(string), d.Id(), err)
				}
			}
		}
	}

	if d.HasChange("security_group") {
		sg := d.Get("security_group").(string)
		err := vmutils.ConfigureWithSecurityGroup(role, sg)
		if err != nil {
			return fmt.Errorf(
				"Error associating security group %s with instance %s: %s", sg, d.Id(), err)
		}
	}

	// Update the adjusted role
	req, err := virtualmachine.NewClient(mc).UpdateRole(d.Id(), d.Id(), d.Id(), *role)
	if err != nil {
		return fmt.Errorf("Error updating role of instance %s: %s", d.Id(), err)
	}

	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for role of instance %s to be updated: %s", d.Id(), err)
	}

	return resourceAzureInstanceRead(d, meta)
}

func resourceAzureInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	log.Printf("[DEBUG] Deleting instance: %s", d.Id())
	req, err := hostedservice.NewClient(mc).DeleteHostedService(d.Id(), true)
	if err != nil {
		return fmt.Errorf("Error deleting instance %s: %s", d.Id(), err)
	}

	// Wait until the instance is deleted
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf(
			"Error waiting for instance %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceAzureEndpointHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["public_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["private_port"].(int)))

	return hashcode.String(buf.String())
}

func retrieveImageDetails(
	mc management.Client,
	label string,
	name string,
	storage string) (func(*virtualmachine.Role) error, string, error) {
	configureForImage, osType, VMLabels, err := retrieveVMImageDetails(mc, label)
	if err == nil {
		return configureForImage, osType, nil
	}

	configureForImage, osType, OSLabels, err := retrieveOSImageDetails(mc, label, name, storage)
	if err == nil {
		return configureForImage, osType, nil
	}

	return nil, "", fmt.Errorf("Could not find image with label '%s'. Available images are: %s",
		label, strings.Join(append(VMLabels, OSLabels...), ", "))
}

func retrieveVMImageDetails(
	mc management.Client,
	label string) (func(*virtualmachine.Role) error, string, []string, error) {
	imgs, err := virtualmachineimage.NewClient(mc).ListVirtualMachineImages()
	if err != nil {
		return nil, "", nil, fmt.Errorf("Error retrieving image details: %s", err)
	}

	var labels []string
	for _, img := range imgs.VMImages {
		if img.Label == label {
			if img.OSDiskConfiguration.OS != linux && img.OSDiskConfiguration.OS != windows {
				return nil, "", nil, fmt.Errorf("Unsupported image OS: %s", img.OSDiskConfiguration.OS)
			}

			configureForImage := func(role *virtualmachine.Role) error {
				return vmutils.ConfigureDeploymentFromVMImage(
					role,
					img.Name,
					"",
					true,
				)
			}

			return configureForImage, img.OSDiskConfiguration.OS, nil, nil
		}

		labels = append(labels, img.Label)
	}

	return nil, "", labels, fmt.Errorf("Could not find image with label '%s'", label)
}

func retrieveOSImageDetails(
	mc management.Client,
	label string,
	name string,
	storage string) (func(*virtualmachine.Role) error, string, []string, error) {
	imgs, err := osimage.NewClient(mc).ListOSImages()
	if err != nil {
		return nil, "", nil, fmt.Errorf("Error retrieving image details: %s", err)
	}

	var labels []string
	for _, img := range imgs.OSImages {
		if img.Label == label {
			if img.OS != linux && img.OS != windows {
				return nil, "", nil, fmt.Errorf("Unsupported image OS: %s", img.OS)
			}
			if img.MediaLink == "" {
				if storage == "" {
					return nil, "", nil,
						fmt.Errorf("When using a platform image, the 'storage' parameter is required")
				}
				img.MediaLink = fmt.Sprintf(osDiskBlobStorageURL, storage, name)
			}

			configureForImage := func(role *virtualmachine.Role) error {
				return vmutils.ConfigureDeploymentFromPlatformImage(
					role,
					img.Name,
					img.MediaLink,
					label,
				)
			}

			return configureForImage, img.OS, nil, nil
		}

		labels = append(labels, img.Label)
	}

	return nil, "", labels, fmt.Errorf("Could not find image with label '%s'", label)
}

func endpointProtocol(p string) virtualmachine.InputEndpointProtocol {
	if p == "tcp" {
		return virtualmachine.InputEndpointProtocolTCP
	}

	return virtualmachine.InputEndpointProtocolUDP
}

func verifyInstanceParameters(d *schema.ResourceData, osType string) error {
	if osType == linux {
		_, pass := d.GetOk("password")
		_, key := d.GetOk("ssh_key_thumbprint")

		if !pass && !key {
			return fmt.Errorf(
				"You must supply a 'password' and/or a 'ssh_key_thumbprint' when using a Linux image")
		}
	}

	if osType == windows {
		if _, ok := d.GetOk("password"); !ok {
			return fmt.Errorf("You must supply a 'password' when using a Windows image")
		}

		if _, ok := d.GetOk("time_zone"); !ok {
			return fmt.Errorf("You must supply a 'time_zone' when using a Windows image")
		}
	}

	if _, ok := d.GetOk("subnet"); ok {
		if _, ok := d.GetOk("virtual_network"); !ok {
			return fmt.Errorf("You must also supply a 'virtual_network' when supplying a 'subnet'")
		}
	}

	if s := d.Get("endpoint").(*schema.Set); s.Len() > 0 {
		for _, v := range s.List() {
			protocol := v.(map[string]interface{})["protocol"].(string)

			if protocol != "tcp" && protocol != "udp" {
				return fmt.Errorf(
					"Invalid endpoint protocol %s! Valid options are 'tcp' and 'udp'.", protocol)
			}
		}
	}

	return nil
}

package azure

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/MSOpenTech/azure-sdk-for-go/management"
	"github.com/MSOpenTech/azure-sdk-for-go/management/hostedservice"
	"github.com/MSOpenTech/azure-sdk-for-go/management/osimage"
	"github.com/MSOpenTech/azure-sdk-for-go/management/virtualmachine"
	"github.com/MSOpenTech/azure-sdk-for-go/management/vmutils"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	linux   = "Linux"
	windows = "Windows"
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

			"network": &schema.Schema{
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
				ForceNew: true,
			},

			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"public_rdp": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"public_ssh": &schema.Schema{
				Type:     schema.TypeBool,
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
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"local_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceAzureEndpointHash,
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
	mc := meta.(*management.Client)

	name := d.Get("name").(string)

	// Compute/set the description
	description := d.Get("description").(string)
	if description == "" {
		description = name
	}

	// Retrieve the needed details of the image
	imageName, imageURL, osType, err := retrieveImageDetails(mc, d.Get("image").(string))
	if err != nil {
		return err
	}

	if imageURL == "" {
		storage, ok := d.GetOk("storage")
		if !ok {
			return fmt.Errorf("When using a platform image, the 'storage' parameter is required")
		}
		imageURL = fmt.Sprintf("http://%s.blob.core.windows.net/vhds/%s.vhd", storage, name)
	}

	// Verify if we have all parameters required for the image OS type
	if err := verifyParameters(d, osType); err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating Cloud Service for instance: %s", name)
	req, err := hostedservice.NewClient(*mc).
		CreateHostedService(
		name,
		d.Get("location").(string),
		d.Get("reverse_dns").(string),
		name,
		fmt.Sprintf("Cloud Service created automatically for instance %s", name),
	)
	if err != nil {
		return fmt.Errorf("Error creating Cloud Service for instance %s: %s", name, err)
	}

	// Wait until the Cloud Service is created
	if err := mc.WaitAsyncOperation(req); err != nil {
		return fmt.Errorf(
			"Error waiting for Cloud Service of instance %s to be created: %s", name, err)
	}

	// Put in this defer here, so we are sure to cleanup already created parts
	// when we exit with an error
	defer func(mc *management.Client) {
		if err != nil {
			req, err := hostedservice.NewClient(*mc).DeleteHostedService(name, true)
			if err != nil {
				log.Printf("[DEBUG] Error cleaning up Cloud Service of instance %s: %s", name, err)
			}

			// Wait until the Cloud Service is deleted
			if err := mc.WaitAsyncOperation(req); err != nil {
				log.Printf(
					"[DEBUG] Error waiting for Cloud Service of instance %s to be deleted : %s", name, err)
			}
		}
	}(mc)

	// Create a new role for the instance
	role := vmutils.NewVmConfiguration(name, d.Get("size").(string))

	log.Printf("[DEBUG] Configuring deployment from image...")
	err = vmutils.ConfigureDeploymentFromPlatformImage(
		&role,
		imageName,
		imageURL,
		d.Get("image").(string),
	)
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

		if d.Get("public_ssh").(bool) {
			if err := vmutils.ConfigureWithPublicSSH(&role); err != nil {
				return fmt.Errorf("Error configuring %s for public SSH: %s", name, err)
			}
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

		if d.Get("public_rdp").(bool) {
			if err := vmutils.ConfigureWithPublicRDP(&role); err != nil {
				return fmt.Errorf("Error configuring %s for public RDP: %s", name, err)
			}
		}
	}

	log.Printf("[DEBUG] Creating the new instance...")
	req, err = virtualmachine.NewClient(*mc).CreateDeployment(role, name)
	if err != nil {
		return fmt.Errorf("Error creating instance %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for the new instance to be created...")
	if err := mc.WaitAsyncOperation(req); err != nil {
		return fmt.Errorf(
			"Error waiting for instance %s to be created: %s", name, err)
	}

	/*
		if v := d.Get("endpoint").(*schema.Set); v.Len() > 0 {
			log.Printf("[DEBUG] Adding Endpoints to the Azure Virtual Machine...")
			endpoints := make([]vmClient.InputEndpoint, v.Len())
			for i, v := range v.List() {
				m := v.(map[string]interface{})
				endpoint := vmClient.InputEndpoint{}
				endpoint.Name = m["name"].(string)
				endpoint.Protocol = m["protocol"].(string)
				endpoint.Port = m["port"].(int)
				endpoint.LocalPort = m["local_port"].(int)
				endpoints[i] = endpoint
			}

			configSets := vmConfig.ConfigurationSets.ConfigurationSet
			if len(configSets) == 0 {
				return fmt.Errorf("Azure virtual machine does not have configuration sets")
			}
			for i := 0; i < len(configSets); i++ {
				if configSets[i].ConfigurationSetType != "NetworkConfiguration" {
					continue
				}
				configSets[i].InputEndpoints.InputEndpoint =
					append(configSets[i].InputEndpoints.InputEndpoint, endpoints...)
			}
		}

	*/

	d.SetId(name)

	return resourceAzureInstanceRead(d, meta)
}

func resourceAzureInstanceRead(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	log.Printf("[DEBUG] Retrieving Cloud Service for instance: %s", d.Id())
	cs, err := hostedservice.NewClient(*mc).GetHostedService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving Cloud Service of instance %s: %s", d.Id(), err)
	}

	d.Set("reverse_dns", cs.ReverseDnsFqdn)
	d.Set("location", cs.Location)

	log.Printf("[DEBUG] Retrieving instance: %s", d.Id())
	wi, err := virtualmachine.NewClient(*mc).GetDeployment(d.Id(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving instance %s: %s", d.Id(), err)
	}

	if len(wi.RoleList) == 0 {
		return fmt.Errorf("Instance %s does not have VIP addresses", d.Id())
	}
	role := wi.RoleList[0]

	d.Set("size", role.RoleSize)

	if len(wi.RoleInstanceList) == 0 {
		return fmt.Errorf("Instance %s does not have IP addresses", d.Id())
	}
	d.Set("ip_address", wi.RoleInstanceList[0].IpAddress)

	if len(wi.VirtualIPs) == 0 {
		return fmt.Errorf("Instance %s does not have VIP addresses", d.Id())
	}
	d.Set("vip_address", wi.VirtualIPs[0].Address)

	connType := "ssh"
	if role.OSVirtualHardDisk.OS == windows {
		connType = windows
	}

	// Set the connection info for any configured provisioners
	d.SetConnInfo(map[string]string{
		"type": connType,
		"host": wi.VirtualIPs[0].Address,
		"user": d.Get("username").(string),
	})

	return nil
}

func resourceAzureInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	if d.HasChange("size") {
		role, err := virtualmachine.NewClient(*mc).GetRole(d.Id(), d.Id(), d.Id())
		if err != nil {
			return fmt.Errorf("Error retrieving role of instance %s: %s", d.Id(), err)
		}

		role.RoleSize = d.Get("size").(string)

		req, err := virtualmachine.NewClient(*mc).UpdateRole(d.Id(), d.Id(), d.Id(), *role)
		if err != nil {
			return fmt.Errorf("Error updating role of instance %s: %s", d.Id(), err)
		}

		if err := mc.WaitAsyncOperation(req); err != nil {
			return fmt.Errorf(
				"Error waiting for role of instance %s to be updated: %s", d.Id(), err)
		}
	}

	if d.HasChange("endpoint") {

	}

	return resourceAzureInstanceRead(d, meta)
}

func resourceAzureInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	log.Printf("[DEBUG] Deleting instance: %s", d.Id())
	req, err := hostedservice.NewClient(*mc).DeleteHostedService(d.Id(), true)
	if err != nil {
		return fmt.Errorf("Error deleting instance %s: %s", d.Id(), err)
	}

	// Wait until the instance is deleted
	if err := mc.WaitAsyncOperation(req); err != nil {
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
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["local_port"].(int)))

	return hashcode.String(buf.String())
}

func retrieveImageDetails(mc *management.Client, label string) (string, string, string, error) {
	imgs, err := osimage.NewClient(*mc).GetImageList()
	if err != nil {
		return "", "", "", fmt.Errorf("Error retrieving image details: %s", err)
	}

	var labels []string
	for _, img := range imgs {
		if img.Label == label {
			if img.OS != linux && img.OS != windows {
				return "", "", "", fmt.Errorf("Unsupported image OS: %s", img.OS)
			}
			return img.Name, img.MediaLink, img.OS, nil
		}
		labels = append(labels, img.Label)
	}

	return "", "", "",
		fmt.Errorf("Could not find image with label '%s', available labels are: %s",
			label, strings.Join(labels, ","))
}

func verifyParameters(d *schema.ResourceData, osType string) error {
	if osType == linux {
		_, pass := d.GetOk("password")
		_, key := d.GetOk("ssh_key_thumbprint")

		if !pass && !key {
			return fmt.Errorf(
				"You must supply a 'password' and/or a 'ssh_key_thumbprint' when using a Linux image")
		}

		if key {
			// check if it's a file of a string containing the key
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

	return nil
}

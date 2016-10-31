package cobbler

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	cobbler "github.com/jtopjian/cobblerclient"
)

func resourceProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfileCreate,
		Read:   resourceProfileRead,
		Update: resourceProfileUpdate,
		Delete: resourceProfileDelete,

		Schema: map[string]*schema.Schema{
			"boot_files": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"comment": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"distro": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enable_gpxe": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_menu": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"fetchable_files": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kernel_options": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kernel_options_post": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kickstart": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ks_meta": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"mgmt_classes": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"mgmt_parameters": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name_servers_search": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"name_servers": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"owners": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"parent": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"proxy": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"redhat_management_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"redhat_management_server": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"repos": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"server": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"template_files": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"template_remote_kickstarts": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"virt_auto_boot": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_bridge": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_cpus": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_disk_driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_file_size": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_path": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_ram": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceProfileCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Create a cobblerclient.Profile struct
	profile := buildProfile(d, config)

	// Attempt to create the Profile
	log.Printf("[DEBUG] Cobbler Profile: Create Options: %#v", profile)
	newProfile, err := config.cobblerClient.CreateProfile(profile)
	if err != nil {
		return fmt.Errorf("Cobbler Profile: Error Creating: %s", err)
	}

	d.SetId(newProfile.Name)

	return resourceProfileRead(d, meta)
}

func resourceProfileRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Retrieve the profile entry from Cobbler
	profile, err := config.cobblerClient.GetProfile(d.Id())
	if err != nil {
		return fmt.Errorf("Cobbler Profile: Error Reading (%s): %s", d.Id(), err)
	}

	// Set all fields
	d.Set("boot_files", profile.BootFiles)
	d.Set("comment", profile.Comment)
	d.Set("distro", profile.Distro)
	d.Set("enable_gpxe", profile.EnableGPXE)
	d.Set("enable_menu", profile.EnableMenu)
	d.Set("fetchable_files", profile.FetchableFiles)
	d.Set("kernel_options", profile.KernelOptions)
	d.Set("kernel_options_post", profile.KernelOptionsPost)
	d.Set("kickstart", profile.Kickstart)
	d.Set("ks_meta", profile.KSMeta)
	d.Set("mgmt_classes", profile.MGMTClasses)
	d.Set("mgmt_parameters", profile.MGMTParameters)
	d.Set("name_servers_search", profile.NameServersSearch)
	d.Set("name_servers", profile.NameServers)
	d.Set("owners", profile.Owners)
	d.Set("proxy", profile.Proxy)
	d.Set("redhat_management_key", profile.RedHatManagementKey)
	d.Set("redhat_management_server", profile.RedHatManagementServer)
	d.Set("repos", profile.Repos)
	d.Set("template_files", profile.TemplateFiles)
	d.Set("template_remote_kickstarts", profile.TemplateRemoteKickstarts)
	d.Set("virt_auto_boot", profile.VirtAutoBoot)
	d.Set("virt_bridge", profile.VirtBridge)
	d.Set("virt_cpus", profile.VirtCPUs)
	d.Set("virt_disk_driver", profile.VirtDiskDriver)
	d.Set("virt_file_size", profile.VirtFileSize)
	d.Set("virt_path", profile.VirtPath)
	d.Set("virt_ram", profile.VirtRam)
	d.Set("virt_type", profile.VirtType)

	return nil
}

func resourceProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Create a cobblerclient.Profile struct
	profile := buildProfile(d, config)

	// Attempt to update the profile with new information
	log.Printf("[DEBUG] Cobbler Profile: Updating Profile (%s) with options: %+v", d.Id(), profile)
	err := config.cobblerClient.UpdateProfile(&profile)
	if err != nil {
		return fmt.Errorf("Cobbler Profile: Error Updating (%s): %s", d.Id(), err)
	}

	return resourceProfileRead(d, meta)
}

func resourceProfileDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Attempt to delete the profile
	if err := config.cobblerClient.DeleteProfile(d.Id()); err != nil {
		return fmt.Errorf("Cobbler Profile: Error Deleting (%s): %s", d.Id(), err)
	}

	return nil
}

// buildProfile builds a cobblerclient.Profile out of the Terraform attributes
func buildProfile(d *schema.ResourceData, meta interface{}) cobbler.Profile {
	mgmtClasses := []string{}
	for _, i := range d.Get("mgmt_classes").([]interface{}) {
		mgmtClasses = append(mgmtClasses, i.(string))
	}

	nameServersSearch := []string{}
	for _, i := range d.Get("name_servers_search").([]interface{}) {
		nameServersSearch = append(nameServersSearch, i.(string))
	}

	nameServers := []string{}
	for _, i := range d.Get("name_servers").([]interface{}) {
		nameServers = append(nameServers, i.(string))
	}

	owners := []string{}
	for _, i := range d.Get("owners").([]interface{}) {
		owners = append(owners, i.(string))
	}

	repos := []string{}
	for _, i := range d.Get("repos").([]interface{}) {
		repos = append(repos, i.(string))
	}

	profile := cobbler.Profile{
		BootFiles:              d.Get("boot_files").(string),
		Comment:                d.Get("comment").(string),
		Distro:                 d.Get("distro").(string),
		EnableGPXE:             d.Get("enable_gpxe").(bool),
		EnableMenu:             d.Get("enable_menu").(bool),
		FetchableFiles:         d.Get("fetchable_files").(string),
		KernelOptions:          d.Get("kernel_options").(string),
		KernelOptionsPost:      d.Get("kernel_options_post").(string),
		Kickstart:              d.Get("kickstart").(string),
		KSMeta:                 d.Get("ks_meta").(string),
		MGMTClasses:            mgmtClasses,
		MGMTParameters:         d.Get("mgmt_parameters").(string),
		Name:                   d.Get("name").(string),
		NameServersSearch:      nameServersSearch,
		NameServers:            nameServers,
		Owners:                 owners,
		Parent:                 d.Get("parent").(string),
		Proxy:                  d.Get("proxy").(string),
		RedHatManagementKey:    d.Get("redhat_management_key").(string),
		RedHatManagementServer: d.Get("redhat_management_server").(string),
		Repos:                    repos,
		Server:                   d.Get("server").(string),
		TemplateFiles:            d.Get("template_files").(string),
		TemplateRemoteKickstarts: d.Get("template_remote_kickstarts").(int),
		VirtAutoBoot:             d.Get("virt_auto_boot").(string),
		VirtBridge:               d.Get("virt_bridge").(string),
		VirtCPUs:                 d.Get("virt_cpus").(string),
		VirtDiskDriver:           d.Get("virt_disk_driver").(string),
		VirtFileSize:             d.Get("virt_file_size").(string),
		VirtPath:                 d.Get("virt_path").(string),
		VirtRam:                  d.Get("virt_ram").(string),
		VirtType:                 d.Get("virt_type").(string),
	}

	return profile
}

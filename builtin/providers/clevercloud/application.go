package clevercloud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/samber/go-clevercloud-api/clever"
)

func resourceCleverCloudApplication(runtime string, availableRegions []string, availableDeploymentMethods []string, availableInstanceSizes []string) *schema.Resource {
	instanceSizes := []string{"pico", "nano", "xs", "s", "m", "l", "xl"}
	sort.Strings(instanceSizes)
	validateInstanceSize := func(v interface{}, k string) (ws []string, es []error) {
		size := strings.ToLower(v.(string))
		if i := sort.SearchStrings(instanceSizes, size); i >= len(instanceSizes) {
			es = append(es, fmt.Errorf(size+" is not available as instance size"))
		}
		return
	}

	return &schema.Resource{
		Create: CreateApplication,
		Update: UpdateApplication,
		Delete: DeleteApplication,
		Exists: ApplicationExists,
		Read:   ReadApplication,

		Schema: map[string]*schema.Schema{

			// SET BY TERRAFORM sub-resource
			"runtime": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  runtime,
				ForceNew: true,
			},

			"deployment_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "git",
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					method := strings.ToLower(v.(string))
					sort.Strings(availableDeploymentMethods)
					if i := sort.SearchStrings(availableDeploymentMethods, method); i >= len(availableDeploymentMethods) {
						es = append(es, fmt.Errorf(method+" deployment method is not available for runtime "+runtime))
					}
					return
				},
			},

			// SET BY USER
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "par",
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					region := strings.ToLower(v.(string))
					sort.Strings(availableRegions)
					if i := sort.SearchStrings(availableRegions, region); i >= len(availableRegions) {
						es = append(es, fmt.Errorf(region+" region is not available for runtime "+runtime))
					}
					return
				},
			},

			"fqdns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"environment": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"cancel_on_push": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"separate_build": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"sticky_sessions": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"homogeneous": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"restart_on_change": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"min_size": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateInstanceSize,
			},
			"max_size": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateInstanceSize,
			},

			"min_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"max_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			// COMPUTED
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			// Include fqnds set in your .tf and externally
			"all_fqdns": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			// Include env vars set in your .tf and externally
			"all_environment": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},

			"runtime_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"git_ssh": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_http": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"branch": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"commit_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateApplication(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	// Create app
	applicationInput, err := resourceDataToApplication(d)
	if err != nil {
		return err
	}
	applicationOutput, err := client.CreateApplication(applicationInput)
	if err != nil {
		return err
	}

	// Set env vars
	envInput, err := resourceDataToEnv(d)
	if err != nil {
		return err
	}
	if _, err := client.CreateApplicationEnv(applicationOutput.Id, envInput); err != nil {
		return err
	}

	// Set dns
	fqdnInput, err := resourceDataToFqdn(d)
	if err != nil {
		return err
	}
	if _, err := client.CreateApplicationFqdn(applicationOutput.Id, fqdnInput); err != nil {
		return err
	}

	// @todo: provisioner git

	return applicationToResourceData(applicationOutput, client, d)
}

func UpdateApplication(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)
	restart_app := false

	// update app
	applicationInput, err := resourceDataToApplication(d)
	if err != nil {
		return err
	}
	applicationOutput, err := client.UpdateApplication(d.Get("id").(string), applicationInput)
	if err != nil {
		return err
	}

	// update env
	// We make a diff between old and new env vars.
	if d.HasChange("environment") {
		o, n := d.GetChange("environment")
		if o == nil {
			o = map[string]interface{}{}
		}
		if n == nil {
			n = map[string]interface{}{}
		}

		om := o.(map[string]interface{})
		nm := n.(map[string]interface{})

		// create or update env vars
		for k, v := range nm {
			if _, ok := om[k]; ok == true && om[k] == v.(string) {
				continue
			}
			if _, err := client.UpdateApplicationEnv(d.Get("id").(string), map[string]string{k: v.(string)}); err != nil {
				return err
			}
			restart_app = true
		}

		// remove env vars that no longer exist
		for k, _ := range om {
			if _, ok := nm[k]; ok == true {
				continue
			}
			if err := client.DeleteApplicationEnv(d.Get("id").(string), k); err != nil {
				return err
			}
			restart_app = true
		}
	}

	// Update fqdns.
	// We make a diff between old and new fqdns.
	// This will not erase domains set manually from the console.
	if d.HasChange("fqdns") {
		o, n := d.GetChange("fqdns")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		toRemove := os.Difference(ns).List()
		toAdd := ns.Difference(os).List()

		// create fqdns that was not set
		for _, fqdn := range toAdd {
			if _, err := client.CreateApplicationFqdn(d.Get("id").(string), []string{fqdn.(string)}); err != nil {
				return err
			}
		}
		// delete fqdns that does not exist anymore
		for _, fqdn := range toRemove {
			if err := client.DeleteApplicationFqdn(d.Get("id").(string), fqdn.(string)); err != nil {
				return err
			}
		}
	}

	// restart app only in case env vars have been modified
	if restart_app == true && d.Get("restart_on_change").(bool) == true && applicationOutput.CommitId != "" {
		if err := client.RestartApplication(d.Get("id").(string)); err != nil {
			return err
		}
	}

	return applicationToResourceData(applicationOutput, client, d)
}

func DeleteApplication(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	err := client.DeleteApplication(d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func ApplicationExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*clever.Client)

	_, err := client.GetApplicationById(d.Id())
	if err != nil {
		if _, ok := err.(clever.NotFoundError); ok {
			err = nil
		}
		return false, err
	}

	return true, nil
}

func ReadApplication(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clever.Client)

	applicationOutput, err := client.GetApplicationById(d.Id())
	if err != nil {
		return err
	}

	return applicationToResourceData(applicationOutput, client, d)
}

func resourceDataToApplication(d *schema.ResourceData) (*clever.ApplicationInput, error) {
	applicationInput := &clever.ApplicationInput{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Region:           d.Get("region").(string),
		Deploy:           d.Get("deployment_method").(string),
		CancelOnPush:     d.Get("cancel_on_push").(bool),
		SeparateBuild:    d.Get("separate_build").(bool),
		StickySessions:   d.Get("sticky_sessions").(bool),
		Homogeneous:      d.Get("homogeneous").(bool),
		InstanceRuntime:  d.Get("runtime").(string),
		InstanceSizeMin:  d.Get("min_size").(string),
		InstanceSizeMax:  d.Get("max_size").(string),
		InstanceCountMin: d.Get("min_count").(int),
		InstanceCountMax: d.Get("max_count").(int),
	}
	return applicationInput, nil
}

func resourceDataToEnv(d *schema.ResourceData) (map[string]string, error) {
	envInput := map[string]string{}
	for k, v := range d.Get("environment").(map[string]interface{}) {
		envInput[k] = v.(string)
	}
	return envInput, nil
}

func resourceDataToFqdn(d *schema.ResourceData) ([]string, error) {
	fqdnInput := []string{}
	for _, fqdn := range d.Get("fqdns").(*schema.Set).List() {
		fqdnInput = append(fqdnInput, fqdn.(string))
	}
	return fqdnInput, nil
}

func applicationToResourceData(applicationOutput *clever.ApplicationOutput, client *clever.Client, d *schema.ResourceData) error {
	d.SetId(applicationOutput.Id)
	d.Set("id", applicationOutput.Id)
	d.Set("name", applicationOutput.Name)
	d.Set("description", applicationOutput.Description)
	d.Set("region", applicationOutput.Region)
	d.Set("sticky_sessions", applicationOutput.StickySessions)
	d.Set("cancel_on_push", applicationOutput.CancelOnPush)
	d.Set("separate_build", applicationOutput.SeparateBuild)
	d.Set("homogeneous", applicationOutput.Homogeneous)
	d.Set("branch", applicationOutput.Branch)
	d.Set("commit_id", applicationOutput.CommitId)
	d.Set("min_count", applicationOutput.Instance.InstanceCountMin)
	d.Set("max_count", applicationOutput.Instance.InstanceCountMax)
	d.Set("min_size", applicationOutput.Instance.InstanceSizeMin)
	d.Set("max_size", applicationOutput.Instance.InstanceSizeMax)
	d.Set("git_ssh", applicationOutput.Deployment.SshUrl)
	d.Set("git_http", applicationOutput.Deployment.HttpUrl)
	d.Set("runtime_name", applicationOutput.Instance.InstanceRuntime.Name)

	// Output env vars and dns set by terraform and externally
	envOutput, err := client.GetApplicationEnvById(applicationOutput.Id)
	if err != nil {
		return err
	}
	d.Set("all_environment", envOutput)
	fqdnOutput, err := client.GetApplicationFqdnById(applicationOutput.Id)
	if err != nil {
		return err
	}
	d.Set("all_fqdns", fqdnOutput)

	return nil
}

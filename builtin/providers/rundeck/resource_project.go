package rundeck

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
)

var projectConfigAttributes = map[string]string{
	"project.name":                          "name",
	"project.description":                   "description",
	"service.FileCopier.default.provider":   "default_node_file_copier_plugin",
	"service.NodeExecutor.default.provider": "default_node_executor_plugin",
	"project.ssh-authentication":            "ssh_authentication_type",
	"project.ssh-key-storage-path":          "ssh_key_storage_path",
	"project.ssh-keypath":                   "ssh_key_file_path",
}

func resourceRundeckProject() *schema.Resource {
	return &schema.Resource{
		Create: CreateProject,
		Update: UpdateProject,
		Delete: DeleteProject,
		Exists: ProjectExists,
		Read:   ReadProject,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name for the project",
				ForceNew:    true,
			},

			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the project to be shown in the Rundeck UI",
				Default:     "Managed by Terraform",
			},

			"ui_url": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Computed: true,
			},

			"resource_model_source": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the resource model plugin to use",
						},
						"config": &schema.Schema{
							Type:        schema.TypeMap,
							Required:    true,
							Description: "Configuration parameters for the selected plugin",
						},
					},
				},
			},

			"default_node_file_copier_plugin": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "jsch-scp",
			},

			"default_node_executor_plugin": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "jsch-ssh",
			},

			"ssh_authentication_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "privateKey",
			},

			"ssh_key_storage_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"ssh_key_file_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"extra_config": &schema.Schema{
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional raw configuration parameters to include in the project configuration, with dots replaced with slashes in the key names due to limitations in Terraform's config language.",
			},
		},
	}
}

func CreateProject(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	// Rundeck's model is a little inconsistent in that we can create
	// a project via a high-level structure but yet we must update
	// the project via its raw config properties.
	// For simplicity's sake we create a bare minimum project here
	// and then delegate to UpdateProject to fill in the rest of the
	// configuration via the raw config properties.

	project, err := client.CreateProject(&rundeck.Project{
		Name: d.Get("name").(string),
	})

	if err != nil {
		return err
	}

	d.SetId(project.Name)
	d.Set("id", project.Name)

	return UpdateProject(d, meta)
}

func UpdateProject(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	// In Rundeck, updates are always in terms of the low-level config
	// properties map, so we need to transform our data structure
	// into the equivalent raw properties.

	projectName := d.Id()

	updateMap := map[string]string{}

	slashReplacer := strings.NewReplacer("/", ".")
	if extraConfig := d.Get("extra_config"); extraConfig != nil {
		for k, v := range extraConfig.(map[string]interface{}) {
			updateMap[slashReplacer.Replace(k)] = v.(string)
		}
	}

	for configKey, attrKey := range projectConfigAttributes {
		v := d.Get(attrKey).(string)
		if v != "" {
			updateMap[configKey] = v
		}
	}

	for i, rmsi := range d.Get("resource_model_source").([]interface{}) {
		rms := rmsi.(map[string]interface{})
		pluginType := rms["type"].(string)
		ci := rms["config"].(map[string]interface{})
		attrKeyPrefix := fmt.Sprintf("resources.source.%v.", i+1)
		typeKey := attrKeyPrefix + "type"
		configKeyPrefix := fmt.Sprintf("%vconfig.", attrKeyPrefix)
		updateMap[typeKey] = pluginType
		for k, v := range ci {
			updateMap[configKeyPrefix+k] = v.(string)
		}
	}

	err := client.SetProjectConfig(projectName, updateMap)

	if err != nil {
		return err
	}

	return ReadProject(d, meta)
}

func ReadProject(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	name := d.Id()
	project, err := client.GetProject(name)

	if err != nil {
		return err
	}

	for configKey, attrKey := range projectConfigAttributes {
		d.Set(projectConfigAttributes[configKey], nil)
		if v, ok := project.Config[configKey]; ok {
			d.Set(attrKey, v)
			// Remove this key so it won't get included in extra_config
			// later.
			delete(project.Config, configKey)
		}
	}

	resourceSourceMap := map[int]interface{}{}
	configMaps := map[int]interface{}{}
	for configKey, v := range project.Config {
		if strings.HasPrefix(configKey, "resources.source.") {
			nameParts := strings.Split(configKey, ".")

			if len(nameParts) < 4 {
				continue
			}

			index, err := strconv.Atoi(nameParts[2])
			if err != nil {
				continue
			}

			if _, ok := resourceSourceMap[index]; !ok {
				configMap := map[string]interface{}{}
				configMaps[index] = configMap
				resourceSourceMap[index] = map[string]interface{}{
					"config": configMap,
				}
			}

			switch nameParts[3] {
			case "type":
				if len(nameParts) != 4 {
					continue
				}
				m := resourceSourceMap[index].(map[string]interface{})
				m["type"] = v
			case "config":
				if len(nameParts) != 5 {
					continue
				}
				m := configMaps[index].(map[string]interface{})
				m[nameParts[4]] = v
			default:
				continue
			}

			// Remove this key so it won't get included in extra_config
			// later.
			delete(project.Config, configKey)
		}
	}

	resourceSources := []map[string]interface{}{}
	resourceSourceIndices := []int{}
	for k := range resourceSourceMap {
		resourceSourceIndices = append(resourceSourceIndices, k)
	}
	sort.Ints(resourceSourceIndices)

	for _, index := range resourceSourceIndices {
		resourceSources = append(resourceSources, resourceSourceMap[index].(map[string]interface{}))
	}
	d.Set("resource_model_source", resourceSources)

	extraConfig := map[string]string{}
	dotReplacer := strings.NewReplacer(".", "/")
	for k, v := range project.Config {
		extraConfig[dotReplacer.Replace(k)] = v
	}
	d.Set("extra_config", extraConfig)

	d.Set("name", project.Name)
	d.Set("ui_url", project.URL)

	return nil
}

func ProjectExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.Client)

	name := d.Id()
	_, err := client.GetProject(name)

	if _, ok := err.(rundeck.NotFoundError); ok {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func DeleteProject(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.Client)

	name := d.Id()
	return client.DeleteProject(name)
}

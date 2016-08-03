package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	dc "github.com/fsouza/go-dockerclient"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:///var/run/docker.sock"),
				Description: "The Docker daemon address",
			},

			"cert_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_PATH", ""),
				Description: "Path to directory with Docker TLS config",
			},

			"registry_auth": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Address of the registry",
						},

						"username": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"registry_auth.config_file"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
							Description:   "Username for the registry",
						},

						"password": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"registry_auth.config_file"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Description:   "Password for the registry",
						},

						"config_file": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"registry_auth.username", "registry_auth.password"},
							DefaultFunc:   schema.EnvDefaultFunc("DOCKER_CONFIG", "~/.docker/config.json"),
							Description:   "Path to docker json file for registry auth",
						},
					},
				},
				Set: providerRegistryAuthHash,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"docker_container": resourceDockerContainer(),
			"docker_image":     resourceDockerImage(),
			"docker_network":   resourceDockerNetwork(),
			"docker_volume":    resourceDockerVolume(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"docker_registry_image": dataSourceDockerRegistryImage(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := DockerConfig{
		Host:     d.Get("host").(string),
		CertPath: d.Get("cert_path").(string),
	}

	client, err := config.NewClient()
	if err != nil {
		return nil, fmt.Errorf("Error initializing Docker client: %s", err)
	}

	err = client.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error pinging Docker server: %s", err)
	}

	authConfigs := &dc.AuthConfigurations{}

	if v, ok := d.GetOk("registry_auth"); ok {
		authConfigs, err = providerSetToRegistryAuth(v.(*schema.Set))

		if err != nil {
			return nil, fmt.Errorf("Error loading registry auth config: %s", err)
		}
	}

	providerConfig := ProviderConfig{
		DockerClient: client,
		AuthConfigs:  authConfigs,
	}

	return &providerConfig, nil
}

func providerRegistryAuthHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%v-", m["address"].(string)))

	if v, ok := m["username"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["password"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["config_file"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

// Take the given registry_auth schemas and return a map of registry auth configurations
func providerSetToRegistryAuth(authSet *schema.Set) (*dc.AuthConfigurations, error) {
	authConfigs := dc.AuthConfigurations{
		Configs: make(map[string]dc.AuthConfiguration),
	}

	for _, authInt := range authSet.List() {
		auth := authInt.(map[string]interface{})
		authConfig := dc.AuthConfiguration{}
		authConfig.ServerAddress = normalizeRegistryAddress(auth["address"].(string))

		// For each registry_auth block, generate an AuthConfiguration using either
		// username/password or the given config file
		if username, ok := auth["username"].(string); ok && username != "" {
			authConfig.Username = auth["username"].(string)
			authConfig.Password = auth["password"].(string)
		} else if configFile, ok := auth["config_file"].(string); ok && configFile != "" {
			if strings.HasPrefix(configFile, "~/") {
				usr, err := user.Current()
				if err != nil {
					return nil, err
				}
				configFile = strings.Replace(configFile, "~", usr.HomeDir, 1)
			}

			r, err := os.Open(configFile)
			if err != nil {
				return nil, fmt.Errorf("Error opening docker registry config file: %v", err)
			}

			auths, err := dc.NewAuthConfigurations(r)
			if err != nil {
				return nil, fmt.Errorf("Error parsing docker registry config json: %v", err)
			}

			foundRegistry := false
			for registry, authFileConfig := range auths.Configs {
				if authConfig.ServerAddress == normalizeRegistryAddress(registry) {
					authConfig.Username = authFileConfig.Username
					authConfig.Password = authFileConfig.Password
					foundRegistry = true
				}
			}

			if !foundRegistry {
				return nil, fmt.Errorf("Couldn't find registry config for '%s' in file: %s",
					authConfig.ServerAddress, configFile)
			}
		}

		authConfigs.Configs[authConfig.ServerAddress] = authConfig
	}

	return &authConfigs, nil
}

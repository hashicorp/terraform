// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

// Modified from github.com/terraform-providers/terraform-provider-kubernetes

const (
	noConfigError = `

[Kubernetes backend] Neither service_account nor load_config_file were set to true, 
this could cause issues connecting to your Kubernetes cluster.
`
)

var (
	secretResource = k8sSchema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)

// New creates a new backend for kubernetes remote state.
func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"secret_suffix": {
						Type:        cty.String,
						Required:    true,
						Description: "Suffix used when creating the secret. The secret will be named in the format: `tfstate-{workspace}-{secret_suffix}`. Note that the backend may append its own numeric index to the secret name when chunking large state files into multiple secrets. In this case, there will be multiple secrets named in the format: `tfstate-{workspace}-{secret_suffix}-{index}`.",
					},
					"labels": {
						Type:        cty.Map(cty.String),
						Optional:    true,
						Description: "Map of additional labels to be applied to the secret.",
					},
					"namespace": {
						Type:        cty.String,
						Optional:    true,
						Description: "Namespace to store the secret in.",
					},
					"in_cluster_config": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Used to authenticate to the cluster from inside a pod.",
					},
					"load_config_file": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Load local kubeconfig.",
					},
					"host": {
						Type:        cty.String,
						Optional:    true,
						Description: "The hostname (in form of URI) of Kubernetes master.",
					},
					"username": {
						Type:        cty.String,
						Optional:    true,
						Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"password": {
						Type:        cty.String,
						Optional:    true,
						Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"insecure": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether server should be accessed without verifying the TLS certificate.",
					},
					"client_certificate": {
						Type:        cty.String,
						Optional:    true,
						Description: "PEM-encoded client certificate for TLS authentication.",
					},
					"client_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "PEM-encoded client certificate key for TLS authentication.",
					},
					"cluster_ca_certificate": {
						Type:        cty.String,
						Optional:    true,
						Description: "PEM-encoded root certificates bundle for TLS authentication.",
					},
					"config_paths": {
						Type:        cty.List(cty.String),
						Optional:    true,
						Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
					},
					"config_path": {
						Type:        cty.String,
						Optional:    true,
						Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH environment variable.",
					},
					"config_context": {
						Type:     cty.String,
						Optional: true,
					},
					"config_context_auth_info": {
						Type:        cty.String,
						Optional:    true,
						Description: "",
					},
					"config_context_cluster": {
						Type:        cty.String,
						Optional:    true,
						Description: "",
					},
					"token": {
						Type:        cty.String,
						Optional:    true,
						Description: "Token to authentifcate a service account.",
					},
					"proxy_url": {
						Type:        cty.String,
						Optional:    true,
						Description: "URL to the proxy to be used for all API requests",
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"exec": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"api_version": {
									Type:     cty.String,
									Required: true,
								},
								"command": {
									Type:     cty.String,
									Required: true,
								},
								"env": {
									Type:     cty.Map(cty.String),
									Optional: true,
								},
								"args": {
									Type:     cty.List(cty.String),
									Optional: true,
								},
							},
						},
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"namespace": {
					EnvVars:  []string{"KUBE_NAMESPACE"},
					Fallback: "default",
				},
				"in_cluster_config": {
					EnvVars:  []string{"KUBE_IN_CLUSTER_CONFIG"},
					Fallback: "false",
				},
				"load_config_file": {
					EnvVars:  []string{"KUBE_LOAD_CONFIG_FILE"},
					Fallback: "true",
				},
				"host": {
					EnvVars: []string{"KUBE_HOST"},
				},
				"username": {
					EnvVars: []string{"KUBE_USER"},
				},
				"password": {
					EnvVars: []string{"KUBE_PASSWORD"},
				},
				"insecure": {
					EnvVars:  []string{"KUBE_INSECURE"},
					Fallback: "false",
				},
				"client_certificate": {
					EnvVars: []string{"KUBE_CLIENT_CERT_DATA"},
				},
				"client_key": {
					EnvVars: []string{"KUBE_CLIENT_KEY_DATA"},
				},
				"cluster_ca_certificate": {
					EnvVars: []string{"KUBE_CLUSTER_CA_CERT_DATA"},
				},
				"config_path": {
					EnvVars: []string{"KUBE_CONFIG_PATH"},
				},
				"config_context": {
					EnvVars: []string{"KUBE_CTX"},
				},
				"config_context_auth_info": {
					EnvVars: []string{"KUBE_CTX_AUTH_INFO"},
				},
				"config_context_cluster": {
					EnvVars: []string{"KUBE_CTX_CLUSTER"},
				},
				"token": {
					EnvVars: []string{"KUBE_TOKEN"},
				},
				"proxy_url": {
					EnvVars: []string{"KUBE_PROXY_URL"},
				},
			},
		},
	}
}

type Backend struct {
	backendbase.Base

	// The fields below are set from configure
	kubernetesSecretClient dynamic.ResourceInterface
	kubernetesLeaseClient  coordinationv1.LeaseInterface
	config                 *restclient.Config
	namespace              string
	labels                 map[string]string
	nameSuffix             string
}

func (b Backend) KubernetesSecretClient() (dynamic.ResourceInterface, error) {
	if b.kubernetesSecretClient != nil {
		return b.kubernetesSecretClient, nil
	}

	client, err := dynamic.NewForConfig(b.config)
	if err != nil {
		return nil, fmt.Errorf("Failed to configure: %s", err)
	}

	b.kubernetesSecretClient = client.Resource(secretResource).Namespace(b.namespace)
	return b.kubernetesSecretClient, nil
}

func (b Backend) KubernetesLeaseClient() (coordinationv1.LeaseInterface, error) {
	if b.kubernetesLeaseClient != nil {
		return b.kubernetesLeaseClient, nil
	}

	client, err := kubernetes.NewForConfig(b.config)
	if err != nil {
		return nil, err
	}

	b.kubernetesLeaseClient = client.CoordinationV1().Leases(b.namespace)
	return b.kubernetesLeaseClient, nil
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	if b.config != nil {
		return nil
	}

	data := backendbase.NewSDKLikeData(configVal)

	cfg, err := getInitialConfig(data)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	// Overriding with static configuration
	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", version.String())

	if v := data.String("host"); v != "" {
		cfg.Host = v
	}
	if v := data.String("username"); v != "" {
		cfg.Username = v
	}
	if v := data.String("password"); v != "" {
		cfg.Password = v
	}
	cfg.Insecure = data.Bool("insecure")
	if v := data.String("cluster_ca_certificate"); v != "" {
		cfg.CAData = bytes.NewBufferString(v).Bytes()
	}
	if v := data.String("client_certificate"); v != "" {
		cfg.CertData = bytes.NewBufferString(v).Bytes()
	}
	if v := data.String("client_key"); v != "" {
		cfg.KeyData = bytes.NewBufferString(v).Bytes()
	}
	if v := data.String("token"); v != "" {
		cfg.BearerToken = v
	}

	if v := data.GetAttr("labels", cty.Map(cty.String)); !v.IsNull() {
		labels := map[string]string{}
		for it := v.ElementIterator(); it.Next(); {
			kV, vV := it.Element()
			if vV.IsNull() {
				vV = cty.StringVal("")
			}
			labels[kV.AsString()] = vV.AsString()
		}
		b.labels = labels
	}

	ns := data.String("namespace")
	b.namespace = ns

	b.nameSuffix = data.String("secret_suffix")
	if hasNumericSuffix(b.nameSuffix, "-") {
		// If the last segment is a number, it's considered invalid.
		// The backend automatically appends its own numeric suffix when chunking large state files into multiple secrets.
		// Allowing a user-defined numeric suffix could cause conflicts with this mechanism.
		return backendbase.ErrorAsDiagnostics(
			fmt.Errorf("secret_suffix must not end with '-<number>', got %q", b.nameSuffix),
		)
	}

	b.config = cfg

	return nil
}

func getInitialConfig(data backendbase.SDKLikeData) (*restclient.Config, error) {
	var cfg *restclient.Config
	var err error

	inCluster := data.Bool("in_cluster_config")
	if inCluster {
		cfg, err = restclient.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = tryLoadingConfigFile(data)
		if err != nil {
			return nil, err
		}
	}

	if cfg == nil {
		cfg = &restclient.Config{}
	}
	return cfg, err
}

func tryLoadingConfigFile(d backendbase.SDKLikeData) (*restclient.Config, error) {
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if v := d.String("config_path"); v != "" {
		configPaths = []string{v}
	} else if v := d.GetAttr("config_paths", cty.List(cty.String)); !v.IsNull() {
		configPaths = append(configPaths, decodeListOfString(v)...)
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}

	expandedPaths := []string{}
	for _, p := range configPaths {
		path, err := homedir.Expand(p)
		if err != nil {
			log.Printf("[DEBUG] Could not expand path: %s", err)
			return nil, err
		}
		log.Printf("[DEBUG] Using kubeconfig: %s", path)
		expandedPaths = append(expandedPaths, path)
	}

	if len(expandedPaths) == 1 {
		loader.ExplicitPath = expandedPaths[0]
	} else {
		loader.Precedence = expandedPaths
	}

	overrides := &clientcmd.ConfigOverrides{}
	ctxSuffix := "; default context"

	configCtx := d.String("config_context")
	authInfo := d.String("config_context_auth_info")
	cluster := d.String("config_context_cluster")
	if configCtx != "" || authInfo != "" || cluster != "" {
		ctxSuffix = "; overriden context"
		if configCtx != "" {
			overrides.CurrentContext = configCtx
			ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
		}

		overrides.Context = clientcmdapi.Context{}
		if authInfo != "" {
			overrides.Context.AuthInfo = authInfo
			ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
		}
		if cluster != "" {
			overrides.Context.Cluster = cluster
			ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
		}
		log.Printf("[DEBUG] Using overidden context: %#v", overrides.Context)
	}

	// exec is a nested block with nesting mode NestingSingle, so GetAttr
	// will return a value of an object type that will be null if the block
	// isn't present at all.
	if v := d.GetAttr("exec", cty.DynamicPseudoType); !v.IsNull() {
		spec := backendbase.NewSDKLikeData(v)
		exec := &clientcmdapi.ExecConfig{
			APIVersion: spec.String("api_version"),
			Command:    spec.String("command"),
			Args:       decodeListOfString(spec.GetAttr("args", cty.List(cty.String))),
		}
		if envMap := spec.GetAttr("env", cty.Map(cty.String)); !envMap.IsNull() {
			for it := envMap.ElementIterator(); it.Next(); {
				kV, vV := it.Element()
				if vV.IsNull() {
					vV = cty.StringVal("")
				}
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{
					Name:  kV.AsString(),
					Value: vV.AsString(),
				})
			}
		}
		overrides.AuthInfo.Exec = exec
	}

	if v := d.String("proxy_url"); v != "" {
		overrides.ClusterDefaults.ProxyURL = v
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		if pathErr, ok := err.(*os.PathError); ok && os.IsNotExist(pathErr.Err) {
			log.Printf("[INFO] Unable to load config file as it doesn't exist at %q", pathErr.Path)
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to initialize kubernetes configuration: %s", err)
	}

	log.Printf("[INFO] Successfully initialized config")
	return cfg, nil
}

func decodeListOfString(v cty.Value) []string {
	if v.IsNull() {
		return nil
	}
	ret := make([]string, 0, v.LengthInt())
	for it := v.ElementIterator(); it.Next(); {
		_, vV := it.Element()
		if vV.IsNull() {
			ret = append(ret, "")
		} else {
			ret = append(ret, vV.AsString())
		}
	}
	return ret
}

func hasNumericSuffix(value, substr string) bool {
	// Find the last occurrence of '-' and get the part after it
	if idx := strings.LastIndex(value, substr); idx != -1 {
		lastPart := value[idx+1:]
		// Try to convert the last part to an integer.
		if _, err := strconv.Atoi(lastPart); err == nil {
			return true
		}
	}
	// Return false if no '-' is found or if the last part isn't numeric
	return false
}

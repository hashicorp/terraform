package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesService() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesServiceCreate,
		Read:   resourceKubernetesServiceRead,
		Exists: resourceKubernetesServiceExists,
		Update: resourceKubernetesServiceUpdate,
		Delete: resourceKubernetesServiceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of a service. http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_ip": {
							Type:        schema.TypeString,
							Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Optional:    true,
							ForceNew:    true,
							Computed:    true,
						},
						"external_ips": {
							Type:        schema.TypeSet,
							Description: "A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"external_name": {
							Type:        schema.TypeString,
							Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.",
							Optional:    true,
						},
						"load_balancer_ip": {
							Type:        schema.TypeString,
							Description: "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.",
							Optional:    true,
						},
						"load_balancer_source_ranges": {
							Type:        schema.TypeSet,
							Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. More info: http://kubernetes.io/docs/user-guide/services-firewalls",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"port": {
							Type:        schema.TypeList,
							Description: "The list of ports that are exposed by this service. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Required:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.",
										Optional:    true,
									},
									"node_port": {
										Type:        schema.TypeInt,
										Description: "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. More info: http://kubernetes.io/docs/user-guide/services#type--nodeport",
										Computed:    true,
										Optional:    true,
									},
									"port": {
										Type:        schema.TypeInt,
										Description: "The port that will be exposed by this service.",
										Required:    true,
									},
									"protocol": {
										Type:        schema.TypeString,
										Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
										Optional:    true,
										Default:     "TCP",
									},
									"target_port": {
										Type:        schema.TypeInt,
										Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. More info: http://kubernetes.io/docs/user-guide/services#defining-a-service",
										Required:    true,
									},
								},
							},
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. More info: http://kubernetes.io/docs/user-guide/services#overview",
							Optional:    true,
						},
						"session_affinity": {
							Type:        schema.TypeString,
							Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Optional:    true,
							Default:     "None",
						},
						"type": {
							Type:        schema.TypeString,
							Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. More info: http://kubernetes.io/docs/user-guide/services#overview",
							Optional:    true,
							Default:     "ClusterIP",
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesServiceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svc := api.Service{
		ObjectMeta: metadata,
		Spec:       expandServiceSpec(d.Get("spec").([]interface{})),
	}
	log.Printf("[INFO] Creating new service: %#v", svc)
	out, err := conn.CoreV1().Services(metadata.Namespace).Create(&svc)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceRead(d, meta)
}

func resourceKubernetesServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading service %s", name)
	svc, err := conn.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received service: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta))
	if err != nil {
		return err
	}

	flattened := flattenServiceSpec(svc.Spec)
	log.Printf("[DEBUG] Flattened service spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps := patchServiceSpec("spec.0.", "/spec/", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service %q: %v", name, string(data))
	out, err := conn.CoreV1().Services(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update service: %s", err)
	}
	log.Printf("[INFO] Submitted updated service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceRead(d, meta)
}

func resourceKubernetesServiceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting service: %#v", name)
	err := conn.CoreV1().Services(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Service %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking service %s", name)
	_, err := conn.CoreV1().Services(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/apis/autoscaling/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesHorizontalPodAutoscaler() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesHorizontalPodAutoscalerCreate,
		Read:   resourceKubernetesHorizontalPodAutoscalerRead,
		Exists: resourceKubernetesHorizontalPodAutoscalerExists,
		Update: resourceKubernetesHorizontalPodAutoscalerUpdate,
		Delete: resourceKubernetesHorizontalPodAutoscalerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("horizontal pod autoscaler", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Behaviour of the autoscaler. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status.",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_replicas": {
							Type:        schema.TypeInt,
							Description: "Upper limit for the number of pods that can be set by the autoscaler.",
							Required:    true,
						},
						"min_replicas": {
							Type:        schema.TypeInt,
							Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
							Optional:    true,
							Default:     1,
						},
						"scale_target_ref": {
							Type:        schema.TypeList,
							Description: "Reference to scaled resource. e.g. Replication Controller",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"api_version": {
										Type:        schema.TypeString,
										Description: "API version of the referent",
										Optional:    true,
									},
									"kind": {
										Type:        schema.TypeString,
										Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
										Required:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
										Required:    true,
									},
								},
							},
						},
						"target_cpu_utilization_percentage": {
							Type:        schema.TypeInt,
							Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesHorizontalPodAutoscalerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svc := api.HorizontalPodAutoscaler{
		ObjectMeta: metadata,
		Spec:       expandHorizontalPodAutoscalerSpec(d.Get("spec").([]interface{})),
	}
	log.Printf("[INFO] Creating new horizontal pod autoscaler: %#v", svc)
	out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(metadata.Namespace).Create(&svc)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted new horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerRead(d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading horizontal pod autoscaler %s", name)
	svc, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received horizontal pod autoscaler: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta))
	if err != nil {
		return err
	}

	flattened := flattenHorizontalPodAutoscalerSpec(svc.Spec)
	log.Printf("[DEBUG] Flattened horizontal pod autoscaler spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesHorizontalPodAutoscalerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps := patchHorizontalPodAutoscalerSpec("spec.0.", "/spec", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating horizontal pod autoscaler %q: %v", name, string(data))
	out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update horizontal pod autoscaler: %s", err)
	}
	log.Printf("[INFO] Submitted updated horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerRead(d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting horizontal pod autoscaler: %#v", name)
	err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Horizontal Pod Autoscaler %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesHorizontalPodAutoscalerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking horizontal pod autoscaler %s", name)
	_, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

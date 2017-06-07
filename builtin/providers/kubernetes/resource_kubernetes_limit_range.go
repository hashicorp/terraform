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

func resourceKubernetesLimitRange() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesLimitRangeCreate,
		Read:   resourceKubernetesLimitRangeRead,
		Exists: resourceKubernetesLimitRangeExists,
		Update: resourceKubernetesLimitRangeUpdate,
		Delete: resourceKubernetesLimitRangeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("limit range", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the limits enforced. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit": {
							Type:        schema.TypeList,
							Description: "Limits is the list of objects that are enforced.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"default": {
										Type:        schema.TypeMap,
										Description: "Default resource requirement limit value by resource name if resource limit is omitted.",
										Optional:    true,
									},
									"default_request": {
										Type:        schema.TypeMap,
										Description: "The default resource requirement request value by resource name if resource request is omitted.",
										Optional:    true,
										Computed:    true,
									},
									"max": {
										Type:        schema.TypeMap,
										Description: "Max usage constraints on this kind by resource name.",
										Optional:    true,
									},
									"max_limit_request_ratio": {
										Type:        schema.TypeMap,
										Description: "The named resource must have a request and limit that are both non-zero where limit divided by request is less than or equal to the enumerated value; this represents the max burst for the named resource.",
										Optional:    true,
									},
									"min": {
										Type:        schema.TypeMap,
										Description: "Min usage constraints on this kind by resource name.",
										Optional:    true,
									},
									"type": {
										Type:        schema.TypeString,
										Description: "Type of resource that this limit applies to.",
										Optional:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesLimitRangeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandLimitRangeSpec(d.Get("spec").([]interface{}), d.IsNewResource())
	if err != nil {
		return err
	}
	limitRange := api.LimitRange{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	log.Printf("[INFO] Creating new limit range: %#v", limitRange)
	out, err := conn.CoreV1().LimitRanges(metadata.Namespace).Create(&limitRange)
	if err != nil {
		return fmt.Errorf("Failed to create limit range: %s", err)
	}
	log.Printf("[INFO] Submitted new limit range: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesLimitRangeRead(d, meta)
}

func resourceKubernetesLimitRangeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading limit range %s", name)
	limitRange, err := conn.CoreV1().LimitRanges(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received limit range: %#v", limitRange)

	err = d.Set("metadata", flattenMetadata(limitRange.ObjectMeta))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenLimitRangeSpec(limitRange.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesLimitRangeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		spec, err := expandLimitRangeSpec(d.Get("spec").([]interface{}), d.IsNewResource())
		if err != nil {
			return err
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating limit range %q: %v", name, string(data))
	out, err := conn.CoreV1().LimitRanges(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update limit range: %s", err)
	}
	log.Printf("[INFO] Submitted updated limit range: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesLimitRangeRead(d, meta)
}

func resourceKubernetesLimitRangeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting limit range: %#v", name)
	err := conn.CoreV1().LimitRanges(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Limit range %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesLimitRangeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking limit range %s", name)
	_, err := conn.CoreV1().LimitRanges(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

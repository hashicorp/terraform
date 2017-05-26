package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesResourceQuota() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesResourceQuotaCreate,
		Read:   resourceKubernetesResourceQuotaRead,
		Exists: resourceKubernetesResourceQuotaExists,
		Update: resourceKubernetesResourceQuotaUpdate,
		Delete: resourceKubernetesResourceQuotaDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("resource quota", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired quota. http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hard": {
							Type:         schema.TypeMap,
							Description:  "The set of desired hard limits for each named resource. More info: http://releases.k8s.io/HEAD/docs/design/admission_control_resource_quota.md#admissioncontrol-plugin-resourcequota",
							Optional:     true,
							Elem:         schema.TypeString,
							ValidateFunc: validateResourceList,
						},
						"scopes": {
							Type:        schema.TypeSet,
							Description: "A collection of filters that must match each object tracked by a quota. If not specified, the quota matches all objects.",
							Optional:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesResourceQuotaCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandResourceQuotaSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	resQuota := api.ResourceQuota{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	log.Printf("[INFO] Creating new resource quota: %#v", resQuota)
	out, err := conn.CoreV1().ResourceQuotas(metadata.Namespace).Create(&resQuota)
	if err != nil {
		return fmt.Errorf("Failed to create resource quota: %s", err)
	}
	log.Printf("[INFO] Submitted new resource quota: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		quota, err := conn.CoreV1().ResourceQuotas(out.Namespace).Get(out.Name, meta_v1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if resourceListEquals(spec.Hard, quota.Status.Hard) {
			return nil
		}
		err = fmt.Errorf("Quotas don't match after creation.\nExpected: %#v\nGiven: %#v",
			spec.Hard, quota.Status.Hard)
		return resource.RetryableError(err)
	})
	if err != nil {
		return err
	}

	return resourceKubernetesResourceQuotaRead(d, meta)
}

func resourceKubernetesResourceQuotaRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading resource quota %s", name)
	resQuota, err := conn.CoreV1().ResourceQuotas(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received resource quota: %#v", resQuota)

	// This is to work around K8S bug
	// See https://github.com/kubernetes/kubernetes/issues/44539
	if resQuota.ObjectMeta.GenerateName == "" {
		if v, ok := d.GetOk("metadata.0.generate_name"); ok {
			resQuota.ObjectMeta.GenerateName = v.(string)
		}
	}

	err = d.Set("metadata", flattenMetadata(resQuota.ObjectMeta))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenResourceQuotaSpec(resQuota.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesResourceQuotaUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	var spec api.ResourceQuotaSpec
	waitForChangedSpec := false
	if d.HasChange("spec") {
		var err error
		spec, err = expandResourceQuotaSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return err
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
		waitForChangedSpec = true
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating resource quota %q: %v", name, string(data))
	out, err := conn.CoreV1().ResourceQuotas(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update resource quota: %s", err)
	}
	log.Printf("[INFO] Submitted updated resource quota: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if waitForChangedSpec {
		err = resource.Retry(1*time.Minute, func() *resource.RetryError {
			quota, err := conn.CoreV1().ResourceQuotas(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				return resource.NonRetryableError(err)
			}
			if resourceListEquals(spec.Hard, quota.Status.Hard) {
				return nil
			}
			err = fmt.Errorf("Quotas don't match after update.\nExpected: %#v\nGiven: %#v",
				spec.Hard, quota.Status.Hard)
			return resource.RetryableError(err)
		})
		if err != nil {
			return err
		}
	}

	return resourceKubernetesResourceQuotaRead(d, meta)
}

func resourceKubernetesResourceQuotaDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting resource quota: %#v", name)
	err := conn.CoreV1().ResourceQuotas(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Resource quota %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesResourceQuotaExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking resource quota %s", name)
	_, err := conn.CoreV1().ResourceQuotas(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

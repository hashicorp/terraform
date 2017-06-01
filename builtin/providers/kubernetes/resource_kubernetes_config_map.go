package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesConfigMapCreate,
		Read:   resourceKubernetesConfigMapRead,
		Exists: resourceKubernetesConfigMapExists,
		Update: resourceKubernetesConfigMapUpdate,
		Delete: resourceKubernetesConfigMapDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("config map", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the configuration data.",
				Optional:    true,
			},
		},
	}
}

func resourceKubernetesConfigMapCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cfgMap := api.ConfigMap{
		ObjectMeta: metadata,
		Data:       expandStringMap(d.Get("data").(map[string]interface{})),
	}
	log.Printf("[INFO] Creating new config map: %#v", cfgMap)
	out, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Create(&cfgMap)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(d, meta)
}

func resourceKubernetesConfigMapRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading config map %s", name)
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)
	err = d.Set("metadata", flattenMetadata(cfgMap.ObjectMeta))
	if err != nil {
		return err
	}
	d.Set("data", cfgMap.Data)

	return nil
}

func resourceKubernetesConfigMapUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")
		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating config map %q: %v", name, string(data))
	out, err := conn.CoreV1().ConfigMaps(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update Config Map: %s", err)
	}
	log.Printf("[INFO] Submitted updated config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(d, meta)
}

func resourceKubernetesConfigMapDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting config map: %#v", name)
	err := conn.CoreV1().ConfigMaps(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Config map %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesConfigMapExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking config map %s", name)
	_, err := conn.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

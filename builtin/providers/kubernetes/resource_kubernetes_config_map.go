package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api/errors"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
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
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(name)
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

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	namespace, name := idParts(d.Id())
	// This is necessary in case the name is generated
	metadata.Name = name

	cfgMap := api.ConfigMap{
		ObjectMeta: metadata,
		Data:       expandStringMap(d.Get("data").(map[string]interface{})),
	}
	log.Printf("[INFO] Updating config map: %#v", cfgMap)
	out, err := conn.CoreV1().ConfigMaps(namespace).Update(&cfgMap)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(d, meta)
}

func resourceKubernetesConfigMapDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting config map: %#v", name)
	err := conn.CoreV1().ConfigMaps(namespace).Delete(name, &api.DeleteOptions{})
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
	_, err := conn.CoreV1().ConfigMaps(namespace).Get(name)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

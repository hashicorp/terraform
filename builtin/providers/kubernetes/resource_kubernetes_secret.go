package kubernetes

import (
	"log"

	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesSecretCreate,
		Read:   resourceKubernetesSecretRead,
		Exists: resourceKubernetesSecretExists,
		Update: resourceKubernetesSecretUpdate,
		Delete: resourceKubernetesSecretDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("secret", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Optional:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Default:     "Opaque",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceKubernetesSecretCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	secret := api.Secret{
		ObjectMeta: metadata,
		StringData: expandStringMap(d.Get("data").(map[string]interface{})),
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = api.SecretType(v.(string))
	}

	log.Printf("[INFO] Creating new secret: %#v", secret)
	out, err := conn.CoreV1().Secrets(metadata.Namespace).Create(&secret)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitting new secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesSecretRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	log.Printf("[INFO] Reading secret %s", name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received secret: %#v", secret)
	err = d.Set("metadata", flattenMetadata(secret.ObjectMeta))
	if err != nil {
		return err
	}

	d.Set("data", byteMapToStringMap(secret.Data))
	d.Set("type", secret.Type)

	return nil
}

func resourceKubernetesSecretUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")

		oldV = base64EncodeStringMap(oldV.(map[string]interface{}))
		newV = base64EncodeStringMap(newV.(map[string]interface{}))

		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))

		ops = append(ops, diffOps...)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating secret %q: %v", name, data)
	out, err := conn.CoreV1().Secrets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update secret: %s", err)
	}

	log.Printf("[INFO] Submitting updated secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesSecretDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	log.Printf("[INFO] Deleting secret: %q", name)
	err := conn.CoreV1().Secrets(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Secret %s deleted", name)

	d.SetId("")

	return nil
}

func resourceKubernetesSecretExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	log.Printf("[INFO] Checking secret %s", name)
	_, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}

	return true, err
}

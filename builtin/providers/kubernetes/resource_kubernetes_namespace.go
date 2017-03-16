package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api/errors"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
)

func resourceKubernetesNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesNamespaceCreate,
		Read:   resourceKubernetesNamespaceRead,
		Exists: resourceKubernetesNamespaceExists,
		Update: resourceKubernetesNamespaceUpdate,
		Delete: resourceKubernetesNamespaceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("namespace"),
		},
	}
}

func resourceKubernetesNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	namespace := api.Namespace{
		ObjectMeta: metadata,
	}
	log.Printf("[INFO] Creating new namespace: %#v", namespace)
	out, err := conn.CoreV1().Namespaces().Create(&namespace)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Reading namespace %s", name)
	namespace, err := conn.CoreV1().Namespaces().Get(name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received namespace: %#v", namespace)
	err = d.Set("metadata", flattenMetadata(namespace.ObjectMeta))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	// This is necessary in case the name is generated
	metadata.Name = d.Id()

	namespace := api.Namespace{
		ObjectMeta: metadata,
	}
	log.Printf("[INFO] Updating namespace: %#v", namespace)
	out, err := conn.CoreV1().Namespaces().Update(&namespace)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Deleting namespace: %#v", name)
	err := conn.CoreV1().Namespaces().Delete(name, &api.DeleteOptions{})
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Target:  []string{},
		Pending: []string{"Terminating"},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Namespaces().Get(name)
			if err != nil {
				if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
					return nil, "", nil
				}
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Namespace %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Namespace %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Checking namespace %s", name)
	_, err := conn.CoreV1().Namespaces().Get(name)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	log.Printf("[INFO] Namespace %s exists", name)
	return true, err
}

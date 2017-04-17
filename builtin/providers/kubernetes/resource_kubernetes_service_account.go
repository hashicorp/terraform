package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	pkgApi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
)

func resourceKubernetesServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesServiceAccountCreate,
		Read:   resourceKubernetesServiceAccountRead,
		Exists: resourceKubernetesServiceAccountExists,
		Update: resourceKubernetesServiceAccountUpdate,
		Delete: resourceKubernetesServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account", true),
			"image_pull_secrets": {
				Type:        schema.TypeList,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Optional:    true,
						},
					},
				},
			},
			"secrets": {
				Type:        schema.TypeList,
				Description: "The list of secrets allowed to be used by pods running using this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_version": {
							Type:        schema.TypeString,
							Description: "API version of the referent.",
							Optional:    true,
						},
						"field_path": {
							Type:        schema.TypeString,
							Description: "If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: \"spec.containers{name}\" (where \"name\" refers to the name of the container that triggered the event) or if no container name is specified \"spec.containers[2]\" (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object.",
							Optional:    true,
						},
						"kind": {
							Type:        schema.TypeString,
							Description: "Kind of the referent. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
							Optional:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Optional:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "Namespace of the referent. More info: http://kubernetes.io/docs/user-guide/namespaces",
							Optional:    true,
						},
						"resource_version": {
							Type:        schema.TypeString,
							Description: "Specific resourceVersion to which this reference is made, if any. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#concurrency-control-and-consistency",
							Optional:    true,
						},
						"uid": {
							Type:        schema.TypeString,
							Description: "UID of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#uids",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	serviceAcc := api.ServiceAccount{
		ObjectMeta:       metadata,
		ImagePullSecrets: expandLocalObjectReferences(d.Get("image_pull_secrets").([]interface{})),
		Secrets:          expandObjectReferences(d.Get("secrets").([]interface{})),
	}
	log.Printf("[INFO] Creating new service account: %#v", serviceAcc)
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Create(&serviceAcc)
	if err != nil {
		return fmt.Errorf("Failed to create service account: %s", err)
	}
	log.Printf("[INFO] Submitted new service account: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceAccountRead(d, meta)
}

func resourceKubernetesServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading service account %s", name)
	serviceAcc, err := conn.CoreV1().ServiceAccounts(namespace).Get(name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received service account: %#v", serviceAcc)

	err = d.Set("metadata", flattenMetadata(serviceAcc.ObjectMeta))
	if err != nil {
		return err
	}
	err = d.Set("image_pull_secrets", flattenLocalObjectReferences(serviceAcc.ImagePullSecrets))
	if err != nil {
		return err
	}

	secrets, err := removeDefaultSecrets(conn, serviceAcc)
	if err != nil {
		return err
	}
	err = d.Set("secrets", secrets)
	if err != nil {
		return err
	}

	return nil
}

// The kubelet is automatically adding default secrets
// so we detect and filter these out to avoid spurious diffs
func removeDefaultSecrets(conn *kubernetes.Clientset, sa *api.ServiceAccount) ([]api.ObjectReference, error) {
	for i, secret := range sa.Secrets {
		out, err := conn.CoreV1().Secrets(sa.Namespace).Get(secret.Name)
		if err != nil {
			return nil, err
		}
		if isDefaultSecret(out, sa) {
			// remove
			log.Printf("[DEBUG] Removing default secret: %#v", secret)
			sa.Secrets = append(sa.Secrets[:i], sa.Secrets[i+1:]...)
		}
	}
	return sa.Secrets, nil
}

func isDefaultSecret(secret *api.Secret, sa *api.ServiceAccount) bool {
	// TODO: Figure out a more reliable way to detect a default secret
	uid, ok := secret.Annotations["kubernetes.io/service-account.uid"]
	if !ok || uid != string(sa.UID) {
		return false
	}
	if secret.Type != api.SecretTypeServiceAccountToken {
		return false
	}
	return true
}

func resourceKubernetesServiceAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	// TODO - add/remove ops
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("image_pull_secrets") {
		pullSecrets := expandLocalObjectReferences(d.Get("image_pull_secrets").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/imagePullSecrets",
			Value: pullSecrets,
		})
	}
	// TODO - add/remove ops
	if d.HasChange("secrets") {
		secrets := expandLocalObjectReferences(d.Get("secrets").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/secrets",
			Value: secrets,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service account %q: %v", name, string(data))
	out, err := conn.CoreV1().ServiceAccounts(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update service account: %s", err)
	}
	log.Printf("[INFO] Submitted updated service account: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceAccountRead(d, meta)
}

func resourceKubernetesServiceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting service account: %#v", name)
	err := conn.CoreV1().ServiceAccounts(namespace).Delete(name, &api.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Service account %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceAccountExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking service account %s", name)
	_, err := conn.CoreV1().ServiceAccounts(namespace).Get(name)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

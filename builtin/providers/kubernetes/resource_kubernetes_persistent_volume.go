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

func resourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeCreate,
		Read:   resourceKubernetesPersistentVolumeRead,
		Exists: resourceKubernetesPersistentVolumeExists,
		Update: resourceKubernetesPersistentVolumeUpdate,
		Delete: resourceKubernetesPersistentVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("persistent volume", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the persistent volume owned by the cluster",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "Contains all ways the volume can be mounted. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"capacity": {
							Type:         schema.TypeMap,
							Description:  "A description of the persistent volume's resources and capacity. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#capacity",
							Required:     true,
							Elem:         schema.TypeString,
							ValidateFunc: validateResourceList,
						},
						"persistent_volume_reclaim_policy": {
							Type:        schema.TypeString,
							Description: "What happens to a persistent volume when released from its claim. Valid options are Retain (default) and Recycle. Recycling must be supported by the volume plugin underlying this persistent volume. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#recycling-policy",
							Optional:    true,
							Default:     "Retain",
						},
						"persistent_volume_source": {
							Type:        schema.TypeList,
							Description: "The specification of a persistent volume.",
							Required:    true,
							MaxItems:    1,
							Elem:        persistentVolumeSourceSchema(),
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPersistentVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPersistentVolumeSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	volume := api.PersistentVolume{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new persistent volume: %#v", volume)
	out, err := conn.CoreV1().PersistentVolumes().Create(&volume)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume: %#v", out)

	stateConf := &resource.StateChangeConf{
		Target:  []string{"Available", "Bound"},
		Pending: []string{"Pending"},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().PersistentVolumes().Get(metadata.Name, meta_v1.GetOptions{})
			if err != nil {
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Persistent volume %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Persistent volume %s created", out.Name)

	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Reading persistent volume %s", name)
	volume, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received persistent volume: %#v", volume)
	err = d.Set("metadata", flattenMetadata(volume.ObjectMeta))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenPersistentVolumeSpec(volume.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPersistentVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		specOps, err := patchPersistentVolumeSpec("/spec", "spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating persistent volume %s: %s", d.Id(), ops)
	out, err := conn.CoreV1().PersistentVolumes().Patch(d.Id(), pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated persistent volume: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Deleting persistent volume: %#v", name)
	err := conn.CoreV1().PersistentVolumes().Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Persistent volume %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Checking persistent volume %s", name)
	_, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

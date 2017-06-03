package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	pkgApi "k8s.io/apimachinery/pkg/types"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesPersistentVolumeClaim() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeClaimCreate,
		Read:   resourceKubernetesPersistentVolumeClaimRead,
		Exists: resourceKubernetesPersistentVolumeClaimExists,
		Update: resourceKubernetesPersistentVolumeClaimUpdate,
		Delete: resourceKubernetesPersistentVolumeClaimDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("wait_until_bound", true)
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("persistent volume claim", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#persistentvolumeclaims",
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "A set of the desired access modes the volume should have. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes-1",
							Required:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"resources": {
							Type:        schema.TypeList,
							Description: "A list of the minimum resources the volume should have. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#resources",
							Required:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"limits": {
										Type:        schema.TypeMap,
										Description: "Map describing the maximum amount of compute resources allowed. More info: http://kubernetes.io/docs/user-guide/compute-resources/",
										Optional:    true,
										ForceNew:    true,
									},
									"requests": {
										Type:        schema.TypeMap,
										Description: "Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: http://kubernetes.io/docs/user-guide/compute-resources/",
										Optional:    true,
										ForceNew:    true,
									},
								},
							},
						},
						"selector": {
							Type:        schema.TypeList,
							Description: "A label query over volumes to consider for binding.",
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"match_expressions": {
										Type:        schema.TypeList,
										Description: "A list of label selector requirements. The requirements are ANDed.",
										Optional:    true,
										ForceNew:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:        schema.TypeString,
													Description: "The label key that the selector applies to.",
													Optional:    true,
													ForceNew:    true,
												},
												"operator": {
													Type:        schema.TypeString,
													Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
													Optional:    true,
													ForceNew:    true,
												},
												"values": {
													Type:        schema.TypeSet,
													Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
													Optional:    true,
													ForceNew:    true,
													Elem:        &schema.Schema{Type: schema.TypeString},
													Set:         schema.HashString,
												},
											},
										},
									},
									"match_labels": {
										Type:        schema.TypeMap,
										Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
										Optional:    true,
										ForceNew:    true,
									},
								},
							},
						},
						"volume_name": {
							Type:        schema.TypeString,
							Description: "The binding reference to the PersistentVolume backing this claim.",
							Optional:    true,
							ForceNew:    true,
							Computed:    true,
						},
					},
				},
			},
			"wait_until_bound": {
				Type:        schema.TypeBool,
				Description: "Whether to wait for the claim to reach `Bound` state (to find volume in which to claim the space)",
				Optional:    true,
				Default:     true,
			},
		},
	}
}

func resourceKubernetesPersistentVolumeClaimCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPersistentVolumeClaimSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	claim := api.PersistentVolumeClaim{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new persistent volume claim: %#v", claim)
	out, err := conn.CoreV1().PersistentVolumeClaims(metadata.Namespace).Create(&claim)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume claim: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	name := out.ObjectMeta.Name

	if d.Get("wait_until_bound").(bool) {
		var lastEvent api.Event
		stateConf := &resource.StateChangeConf{
			Target:  []string{"Bound"},
			Pending: []string{"Pending"},
			Timeout: d.Timeout(schema.TimeoutCreate),
			Refresh: func() (interface{}, string, error) {
				out, err := conn.CoreV1().PersistentVolumeClaims(metadata.Namespace).Get(name, meta_v1.GetOptions{})
				if err != nil {
					log.Printf("[ERROR] Received error: %#v", err)
					return out, "", err
				}

				events, err := conn.CoreV1().Events(metadata.Namespace).List(meta_v1.ListOptions{
					FieldSelector: fields.Set(map[string]string{
						"involvedObject.name":      metadata.Name,
						"involvedObject.namespace": metadata.Namespace,
						"involvedObject.kind":      "PersistentVolumeClaim",
					}).String(),
				})
				if err != nil {
					return out, "", err
				}
				if len(events.Items) > 0 {
					lastEvent = events.Items[0]
				}

				statusPhase := fmt.Sprintf("%v", out.Status.Phase)
				log.Printf("[DEBUG] Persistent volume claim %s status received: %#v", out.Name, statusPhase)
				return out, statusPhase, nil
			},
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			reason := ""
			if lastEvent.Reason != "" {
				reason = fmt.Sprintf(". Reason: %s: %s", lastEvent.Reason, lastEvent.Message)
			}
			return fmt.Errorf("%s%s", err, reason)
		}
	}
	log.Printf("[INFO] Persistent volume claim %s created", out.Name)

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading persistent volume claim %s", name)
	claim, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received persistent volume claim: %#v", claim)
	err = d.Set("metadata", flattenMetadata(claim.ObjectMeta))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenPersistentVolumeClaimSpec(claim.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPersistentVolumeClaimUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)
	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	// The whole spec is ForceNew = nothing to update there
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating persistent volume claim: %s", ops)
	out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated persistent volume claim: %#v", out)

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting persistent volume claim: %#v", name)
	err := conn.CoreV1().PersistentVolumeClaims(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Persistent volume claim %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeClaimExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking persistent volume claim %s", name)
	_, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

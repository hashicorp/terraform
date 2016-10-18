package maas

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"net/url"
	"time"
)

// This function doesn't really *create* a new node but, power an already registered
// node.
func resourceMAASInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	log.Println("[DEBUG] [resourceMAASInstanceCreate] Launching new maas_instance")

	/*
		According to the MAAS API documentation here: https://maas.ubuntu.com/docs/api.html
		We need to acquire or allocate a node before we start it.  We pass (as url.Values)
		some parameters that could be used to narrow down our selection (cpu_count, memory, etc.)
	*/

	constraints, err := parseConstraints(d)
	if err != nil {
		log.Println("[ERROR] [resourceMAASInstanceCreate] Unable to parse constraints.")
		return err
	}

	nodeObj, err := nodesAllocate(meta.(*Config).MAASObject, constraints)
	if err != nil {
		log.Printf("[ERROR] [resourceMAASInstanceCreate] Unable to allocate nodes")
		return err
	}

	if err := nodeDo(meta.(*Config).MAASObject, nodeObj.system_id, "start", url.Values{}); err != nil {
		log.Printf("[ERROR] [resourceMAASInstanceCreate] Unable to power up node: %s\n", nodeObj.system_id)
		return err
	}

	log.Printf("[DEBUG] [resourceMAASInstanceCreate] Waiting for instance (%s) to become active\n", nodeObj.system_id)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"6:9"},
		Target:     "6:6",
		Refresh:    getNodeStatus(meta.(*Config).MAASObject, nodeObj.system_id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"[ERROR] [resourceMAASInstanceCreate] Error waiting for instance (%s) to become ready: %s",
			nodeObj.system_id, err)
	}

	d.SetId(nodeObj.system_id)
	return resourceMAASInstanceUpdate(d, meta)
}

func resourceMAASInstanceRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading instance (%s) information.\n", d.Id())
	return nil
}

func resourceMAASInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] [resourceMAASInstanceUpdate] Modifying instance %s\n", d.Id())

	d.Partial(true)

	d.Partial(false)

	log.Printf("[DEBUG] Done Modifying instance %s", d.Id())
	return resourceMAASInstanceRead(d, meta)
}

// This function doesn't really *delete* a maas managed instance but releases (read, turns off) the node.
func resourceMAASInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Deleting instance %s\n", d.Id())

	if err := nodeRelease(meta.(*Config).MAASObject, d.Id()); err != nil {
		return err
	}

	log.Printf("[DEBUG] [resourceMAASInstanceDelete] Node (%s) released", d.Id())

	d.SetId("")

	return nil
}

func resourceMAASInstanceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("[DEBUG] Checking for the existence of %s", d.Id())
	if _, err := getSingleNode(meta.(*Config).MAASObject, d.Id()); err != nil {
		log.Printf("[DEBUG] Unable to get information on instance %s\n", d.Id())
		return false, err
	}
	return true, nil
}

func resourceMAASInstance() *schema.Resource {
	log.Println("[DEBUG] [resourceMAASInstance] Initializing data structure")
	return &schema.Resource{
		Create: resourceMAASInstanceCreate,
		Read:   resourceMAASInstanceRead,
		Update: resourceMAASInstanceUpdate,
		Delete: resourceMAASInstanceDelete,

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"architecture": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"boot_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"cpu_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"disable_ipv4": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"distro_series": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip_addresses": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"macaddress_set": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"resource_uri": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"netboot": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"osystem": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"physicalblockdevice_set": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"block_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"id": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"id_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"model": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"serial": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"tags": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"power_state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"power_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"pxe_mac": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"resource_uri": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"resource_uri": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"routers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"status": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"storage": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"substatus": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"swap_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"system_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"tag_names": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"zone": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"resource_uri": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},

			"hwe_kernel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

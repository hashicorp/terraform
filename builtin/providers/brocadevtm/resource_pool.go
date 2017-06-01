package brocadevtm

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/go-brocade-vtm"
	"github.com/sky-uk/go-brocade-vtm/api/pool"

	"fmt"
)

func getSinglePool(poolName string, vtmClient *brocadevtm.VTMClient) (*pool.Pool, error) {

	getSinglePoolAPI := pool.NewGetSingle(poolName)
	getSinglePoolErr := vtmClient.Do(getSinglePoolAPI)
	if getSinglePoolErr != nil {
		return getSinglePoolErr
	}

	if getSinglePoolAPI.StatusCode() != 200 {
		return nil, fmt.Errorf("Status code : %d , Response: %s ", getSinglePoolAPI.StatusCode(), getSinglePoolAPI.GetResponse())
	}
	thisPool := getSinglePoolAPI.GetResponse()

	return thisPool, nil
}

func resourcePool() *schema.Resource {
	return &schema.Resource{
		Create: resourcePoolCreate,
		Read:   resourcePoolRead,
		Delete: resourcePoolDelete,
		Update: resourcePoolUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"nodelist": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem:     *schema.TypeString,
			},
			"monitorlist": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem:     *schema.TypeString,
			},
		},
	}

}

// resourcePoolCreate - Creates a  pool resource object
func resourcePoolCreate() {

}

// resourcePoolRead - Reads a  pool resource
func resourcePoolRead() {

}

// resourcePoolDelete - Deletes a pool resource
func resourcePoolDelete() {

}

// resourcePoolUpdate - Updates an existing pool resource
func resourcePoolUpdate() {

}

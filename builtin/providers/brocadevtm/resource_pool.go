package brocadevtm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/go-brocade-vtm"
	"github.com/sky-uk/go-brocade-vtm/api/pool"
	"log"
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
				Type:     *schema.TypeString,
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
func resourcePoolCreate(d *schema.ResourceData, m interface{}) error {

	vtmClient := m.(*brocadevtm.VTMClient)
	var createPool pool.Pool
	var poolName string
	if v, ok := d.GetOk("name"); ok {
		poolName = v.(string)
	} else {
		return fmt.Errorf("Pool name argument required")
	}
	if v, ok := d.GetOk("nodelist"); ok {
		createPool.Properties.Basic.NodesTable = v.([]interface{})
	}
	if v, ok := d.GetOk("max_connection_attempts"); ok {
		createPool.Properties.Basic.MaxConnectionAttempts = v.(int)
	}
	if v, ok := d.GetOk("max_idle_connections_pernode"); ok {
		createPool.Properties.Basic.MaxIdleConnectionsPerNode = v.(int)
	}
	if v, ok := d.GetOk("max_timed_out_connection_attempts"); ok {
		createPool.Properties.Basic.MaxTimeoutConnectionAttempts = v.(int)
	}
	if v, ok := d.GetOk("monitorlist"); ok {
		createPool.Properties.Basic.Monitors = v.([]interface{})
	}
	if v, ok := d.GetOk("node_close_with_rst"); ok {
		createPool.Properties.Basic.NodeCloseWithReset = v.(*bool)
	}
	if v, ok := d.GetOk("max_connection_timeout"); ok {
		createPool.Properties.Connection.MaxConnectTime = v.(int)
	}
	if v, ok := d.GetOk("max_connections_per_node"); ok {
		createPool.Properties.Connection.MaxConnectionsPerNode = v.(int)
	}
	if v, ok := d.GetOk("max_queue_size"); ok {
		createPool.Properties.Connection.MaxQueueSize = v.(int)
	}
	if v, ok := d.GetOk("max_reply_time"); ok {
		createPool.Properties.Connection.MaxReplyTime = v.(int)
	}
	if v, ok := d.GetOk("queue_timeout"); ok {
		createPool.Properties.Connection.QueueTimeout = v.(int)
	}
	if v, ok := d.GetOk("http_keepalive"); ok {
		createPool.Properties.HTTP.HTTPKeepAlive = v.(*bool)
	}
	if v, ok := d.GetOk("http_keepalive_non_idempotent"); ok {
		createPool.Properties.HTTP.HTTPKeepAliveNonIdempotent = v.(*bool)
	}
	if v, ok := d.GetOk("load_balancing_priority_enabled"); ok {
		1
		createPool.Properties.LoadBalancing.PriorityEnabled = v.(*bool)
	}
	if v, ok := d.GetOk("load_balancing_priority_nodes"); ok {
		createPool.Properties.LoadBalancing.PriorityNodes = v.(int)
	}
	if v, ok := d.GetOk("tcp_nagle"); ok {
		createPool.Properties.TCP.Nagle = v.(bool)
	}

	createAPI := pool.NewCreate(poolName, createPool)
	err := vtmClient.Do(createAPI)
	if err != nil {
		return fmt.Errorf("Could not create pool: %+v", err)
	}
	if createAPI.StatusCode() != 201 && createAPI.StatusCode() != 200 {
		return fmt.Errorf("Invalid HTTP response code %+v returned. Response object was %+v", createAPI.StatusCode(), createAPI.ResponseObject())
	}

	d.SetId(poolName)
	return resourcePoolRead(d, m)

}

// resourcePoolRead - Reads a  pool resource
func resourcePoolRead(d *schema.ResourceData, m interface{}) error {
	/*vtmClient := m.(*brocadevtm.VTMClient)
	var readPool pool.Pool
	var poolName string
	if v, ok := d.GetOk("name"); ok {
		poolName = v.(string)
	} else {
		return fmt.Errorf("Pool name argument required")
	}
	if v, ok := d.GetOk("nodelist"); ok {
		readPool.Properties.Basic.NodesTable = v.([]interface{})
	}
	if v, ok := d.GetOk("monitorlist"); ok {
		readPool.Properties.Basic.Monitors = v.([]interface{)
	}*/

}

// resourcePoolDelete - Deletes a pool resource
func resourcePoolDelete(d *schema.ResourceData, m interface{}) error {
	vtmClient := m.(*brocadevtm.VTMClient)
	var poolName string
	if v, ok := d.GetOk("name"); ok {
		poolName = v.(string)
	} else {
		return fmt.Errorf("Pool name argument required")
	}
	deleteAPI := pool.NewDelete(poolName)
	deleteErr := vtmClient.Do(deleteAPI)
	if deleteErr != nil {
		log.Println("Error Deleting the pool:", deleteErr)
	}
	d.SetId("")
	return nil

}

// resourcePoolUpdate - Updates an existing pool resource
func resourcePoolUpdate(d *schema.ResourceData, m interface{}) error {

}

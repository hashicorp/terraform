package rabbitmq

import (
	"fmt"
	"log"
	"strings"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceQueue() *schema.Resource {
	return &schema.Resource{
		Create: CreateQueue,
		Read:   ReadQueue,
		Delete: DeleteQueue,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vhost": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},

			"settings": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"durable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},

						"auto_delete": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},

						"arguments": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func CreateQueue(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)
	vhost := d.Get("vhost").(string)
	settingsList := d.Get("settings").([]interface{})

	settingsMap, ok := settingsList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to parse settings")
	}

	if err := declareQueue(rmqc, vhost, name, settingsMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", name, vhost)
	d.SetId(id)

	return ReadQueue(d, meta)
}

func ReadQueue(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	queueId := strings.Split(d.Id(), "@")
	if len(queueId) < 2 {
		return fmt.Errorf("Unable to determine Queue ID")
	}

	user := queueId[0]
	vhost := queueId[1]

	queueSettings, err := rmqc.GetQueue(vhost, user)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Queue retrieved for %s: %#v", d.Id(), queueSettings)

	d.Set("name", queueSettings.Name)
	d.Set("vhost", queueSettings.Vhost)

	queue := make([]map[string]interface{}, 1)
	e := make(map[string]interface{})
	e["durable"] = queueSettings.Durable
	e["auto_delete"] = queueSettings.AutoDelete
	e["arguments"] = queueSettings.Arguments
	queue[0] = e

	d.Set("settings", queue)

	return nil
}

func DeleteQueue(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	queueId := strings.Split(d.Id(), "@")
	if len(queueId) < 2 {
		return fmt.Errorf("Unable to determine Queue ID")
	}

	user := queueId[0]
	vhost := queueId[1]

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete queue for %s", d.Id())

	resp, err := rmqc.DeleteQueue(vhost, user)
	log.Printf("[DEBUG] RabbitMQ: Queue delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// the queue was automatically deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ queue: %s", resp.Status)
	}

	return nil
}

func declareQueue(rmqc *rabbithole.Client, vhost string, name string, settingsMap map[string]interface{}) error {
	queueSettings := rabbithole.QueueSettings{}

	if v, ok := settingsMap["durable"].(bool); ok {
		queueSettings.Durable = v
	}

	if v, ok := settingsMap["auto_delete"].(bool); ok {
		queueSettings.AutoDelete = v
	}

	if v, ok := settingsMap["arguments"].(map[string]interface{}); ok {
		queueSettings.Arguments = v
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to declare queue for %s@%s: %#v", name, vhost, queueSettings)

	resp, err := rmqc.DeclareQueue(vhost, name, queueSettings)
	log.Printf("[DEBUG] RabbitMQ: Queue declare response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error declaring RabbitMQ queue: %s", resp.Status)
	}

	return nil
}

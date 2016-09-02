package rabbitmq

import (
	"fmt"
	"log"
	"strings"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBinding() *schema.Resource {
	return &schema.Resource{
		Create: CreateBinding,
		Read:   ReadBinding,
		Delete: DeleteBinding,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"source": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vhost": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"properties_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"routing_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"arguments": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func CreateBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	vhost := d.Get("vhost").(string)
	bindingInfo := rabbithole.BindingInfo{
		Source:          d.Get("source").(string),
		Destination:     d.Get("destination").(string),
		DestinationType: d.Get("destination_type").(string),
		RoutingKey:      d.Get("routing_key").(string),
		PropertiesKey:   d.Get("properties_key").(string),
		Arguments:       d.Get("arguments").(map[string]interface{}),
	}

	if err := declareBinding(rmqc, vhost, bindingInfo); err != nil {
		return err
	}

	name := fmt.Sprintf("%s/%s/%s/%s/%s", vhost, bindingInfo.Source, bindingInfo.Destination, bindingInfo.DestinationType, bindingInfo.PropertiesKey)
	d.SetId(name)

	return ReadBinding(d, meta)
}

func ReadBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	bindingId := strings.Split(d.Id(), "/")
	if len(bindingId) < 5 {
		return fmt.Errorf("Unable to determine binding ID")
	}

	vhost := bindingId[0]
	source := bindingId[1]
	destination := bindingId[2]
	destinationType := bindingId[3]
	propertiesKey := bindingId[4]

	bindings, err := rmqc.ListBindingsIn(vhost)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Bindings retrieved: %#v", bindings)
	bindingFound := false
	for _, binding := range bindings {
		if binding.Source == source && binding.Destination == destination && binding.DestinationType == destinationType && binding.PropertiesKey == propertiesKey {
			log.Printf("[DEBUG] RabbitMQ: Found Binding: %#v", binding)
			bindingFound = true

			d.Set("vhost", binding.Vhost)
			d.Set("source", binding.Source)
			d.Set("destination", binding.Destination)
			d.Set("destination_type", binding.DestinationType)
			d.Set("routing_key", binding.RoutingKey)
			d.Set("properties_key", binding.PropertiesKey)
			d.Set("arguments", binding.Arguments)
		}
	}

	// The binding could not be found,
	// so consider it deleted and remove from state
	if !bindingFound {
		d.SetId("")
	}

	return nil
}

func DeleteBinding(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	bindingId := strings.Split(d.Id(), "/")
	if len(bindingId) < 5 {
		return fmt.Errorf("Unable to determine binding ID")
	}

	vhost := bindingId[0]
	source := bindingId[1]
	destination := bindingId[2]
	destinationType := bindingId[3]
	propertiesKey := bindingId[4]

	bindingInfo := rabbithole.BindingInfo{
		Vhost:           vhost,
		Source:          source,
		Destination:     destination,
		DestinationType: destinationType,
		PropertiesKey:   propertiesKey,
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete binding for %s/%s/%s/%s/%s",
		vhost, source, destination, destinationType, propertiesKey)

	resp, err := rmqc.DeleteBinding(vhost, bindingInfo)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] RabbitMQ: Binding delete response: %#v", resp)

	if resp.StatusCode == 404 {
		// The binding was already deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ binding: %s", resp.Status)
	}

	return nil
}

func declareBinding(rmqc *rabbithole.Client, vhost string, bindingInfo rabbithole.BindingInfo) error {
	log.Printf("[DEBUG] RabbitMQ: Attempting to declare binding for %s/%s/%s/%s/%s",
		vhost, bindingInfo.Source, bindingInfo.Destination, bindingInfo.DestinationType, bindingInfo.PropertiesKey)

	resp, err := rmqc.DeclareBinding(vhost, bindingInfo)
	log.Printf("[DEBUG] RabbitMQ: Binding declare response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error declaring RabbitMQ binding: %s", resp.Status)
	}

	return nil
}

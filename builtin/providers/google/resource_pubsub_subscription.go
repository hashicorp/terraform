package google

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/pubsub/v1"
)

func resourcePubsubSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourcePubsubSubscriptionCreate,
		Read:   resourcePubsubSubscriptionRead,
		Delete: resourcePubsubSubscriptionDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"topic": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ack_deadline_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"push_config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attributes": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
							Elem:     schema.TypeString,
						},

						"push_endpoint": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func cleanAdditionalArgs(args map[string]interface{}) map[string]string {
	cleaned_args := make(map[string]string)
	for k, v := range args {
		cleaned_args[k] = v.(string)
	}
	return cleaned_args
}

func resourcePubsubSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("projects/%s/subscriptions/%s", project, d.Get("name").(string))
	computed_topic_name := fmt.Sprintf("projects/%s/topics/%s", project, d.Get("topic").(string))

	//  process optional parameters
	var ackDeadlineSeconds int64
	ackDeadlineSeconds = 10
	if v, ok := d.GetOk("ack_deadline_seconds"); ok {
		ackDeadlineSeconds = int64(v.(int))
	}

	var subscription *pubsub.Subscription
	if v, ok := d.GetOk("push_config"); ok {
		push_configs := v.([]interface{})

		if len(push_configs) > 1 {
			return fmt.Errorf("At most one PushConfig is allowed per subscription!")
		}

		push_config := push_configs[0].(map[string]interface{})
		attributes := push_config["attributes"].(map[string]interface{})
		attributesClean := cleanAdditionalArgs(attributes)
		pushConfig := &pubsub.PushConfig{Attributes: attributesClean, PushEndpoint: push_config["push_endpoint"].(string)}
		subscription = &pubsub.Subscription{AckDeadlineSeconds: ackDeadlineSeconds, Topic: computed_topic_name, PushConfig: pushConfig}
	} else {
		subscription = &pubsub.Subscription{AckDeadlineSeconds: ackDeadlineSeconds, Topic: computed_topic_name}
	}

	call := config.clientPubsub.Projects.Subscriptions.Create(name, subscription)
	res, err := call.Do()
	if err != nil {
		return err
	}

	d.SetId(res.Name)

	return nil
}

func resourcePubsubSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Id()
	call := config.clientPubsub.Projects.Subscriptions.Get(name)
	_, err := call.Do()
	if err != nil {
		return err
	}

	return nil
}

func resourcePubsubSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Id()
	call := config.clientPubsub.Projects.Subscriptions.Delete(name)
	_, err := call.Do()
	if err != nil {
		return err
	}

	return nil
}

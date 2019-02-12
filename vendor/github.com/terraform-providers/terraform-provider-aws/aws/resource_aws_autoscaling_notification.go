package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingNotification() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingNotificationCreate,
		Read:   resourceAwsAutoscalingNotificationRead,
		Update: resourceAwsAutoscalingNotificationUpdate,
		Delete: resourceAwsAutoscalingNotificationDelete,

		Schema: map[string]*schema.Schema{
			"topic_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"group_names": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"notifications": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsAutoscalingNotificationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := convertSetToList(d.Get("group_names").(*schema.Set))
	nl := convertSetToList(d.Get("notifications").(*schema.Set))

	topic := d.Get("topic_arn").(string)
	if err := addNotificationConfigToGroupsWithTopic(conn, gl, nl, topic); err != nil {
		return err
	}

	// ARNs are unique, and these notifications are per ARN, so we re-use the ARN
	// here as the ID
	d.SetId(topic)
	return resourceAwsAutoscalingNotificationRead(d, meta)
}

func resourceAwsAutoscalingNotificationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := convertSetToList(d.Get("group_names").(*schema.Set))

	opts := &autoscaling.DescribeNotificationConfigurationsInput{
		AutoScalingGroupNames: gl,
	}

	topic := d.Get("topic_arn").(string)
	// Grab all applicable notifcation configurations for this Topic.
	// Each NotificationType will have a record, so 1 Group with 3 Types results
	// in 3 records, all with the same Group name
	gRaw := make(map[string]bool)
	nRaw := make(map[string]bool)

	i := 0
	err := conn.DescribeNotificationConfigurationsPages(opts, func(resp *autoscaling.DescribeNotificationConfigurationsOutput, lastPage bool) bool {
		if resp != nil {
			i++
			log.Printf("[DEBUG] Paging DescribeNotificationConfigurations for (%s), page: %d", d.Id(), i)
		} else {
			log.Printf("[DEBUG] Paging finished for DescribeNotificationConfigurations (%s)", d.Id())
		}

		for _, n := range resp.NotificationConfigurations {
			if *n.TopicARN == topic {
				gRaw[*n.AutoScalingGroupName] = true
				nRaw[*n.NotificationType] = true
			}
		}
		return true // return false to stop paging
	})
	if err != nil {
		return err
	}

	// Grab the keys here as the list of Groups
	var gList []string
	for k := range gRaw {
		gList = append(gList, k)
	}

	// Grab the keys here as the list of Types
	var nList []string
	for k := range nRaw {
		nList = append(nList, k)
	}

	if err := d.Set("group_names", gList); err != nil {
		return err
	}
	if err := d.Set("notifications", nList); err != nil {
		return err
	}

	return nil
}

func resourceAwsAutoscalingNotificationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	// Notifications API call is a PUT, so we don't need to diff the list, just
	// push whatever it is and AWS sorts it out
	nl := convertSetToList(d.Get("notifications").(*schema.Set))

	o, n := d.GetChange("group_names")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)
	remove := convertSetToList(os.Difference(ns))
	add := convertSetToList(ns.Difference(os))

	topic := d.Get("topic_arn").(string)

	if err := removeNotificationConfigToGroupsWithTopic(conn, remove, topic); err != nil {
		return err
	}

	var update []*string
	if d.HasChange("notifications") {
		update = convertSetToList(d.Get("group_names").(*schema.Set))
	} else {
		update = add
	}

	if err := addNotificationConfigToGroupsWithTopic(conn, update, nl, topic); err != nil {
		return err
	}

	return resourceAwsAutoscalingNotificationRead(d, meta)
}

func addNotificationConfigToGroupsWithTopic(conn *autoscaling.AutoScaling, groups []*string, nl []*string, topic string) error {
	for _, a := range groups {
		opts := &autoscaling.PutNotificationConfigurationInput{
			AutoScalingGroupName: a,
			NotificationTypes:    nl,
			TopicARN:             aws.String(topic),
		}

		_, err := conn.PutNotificationConfiguration(opts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				return fmt.Errorf("Error creating Autoscaling Group Notification for Group %s, error: \"%s\", code: \"%s\"", *a, awsErr.Message(), awsErr.Code())
			}
			return err
		}
	}
	return nil
}

func removeNotificationConfigToGroupsWithTopic(conn *autoscaling.AutoScaling, groups []*string, topic string) error {
	for _, r := range groups {
		opts := &autoscaling.DeleteNotificationConfigurationInput{
			AutoScalingGroupName: r,
			TopicARN:             aws.String(topic),
		}

		_, err := conn.DeleteNotificationConfiguration(opts)
		if err != nil {
			return fmt.Errorf("Error deleting notification configuration for ASG \"%s\", Topic ARN \"%s\"", *r, topic)
		}
	}
	return nil
}

func resourceAwsAutoscalingNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := convertSetToList(d.Get("group_names").(*schema.Set))

	topic := d.Get("topic_arn").(string)
	err := removeNotificationConfigToGroupsWithTopic(conn, gl, topic)
	return err
}

func convertSetToList(s *schema.Set) (nl []*string) {
	l := s.List()
	for _, n := range l {
		nl = append(nl, aws.String(n.(string)))
	}

	return nl
}

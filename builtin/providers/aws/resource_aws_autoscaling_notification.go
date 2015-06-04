package aws

import (
	"fmt"
	"sort"
	"strings"

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
			"topic_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"notifications": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsAutoscalingNotificationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := d.Get("group_names").(*schema.Set).List()
	var groups []interface{}
	for _, g := range gl {
		groups = append(groups, g)
	}

	nl := getNofiticationList(d.Get("notifications").([]interface{}))

	topic := d.Get("topic_arn").(string)
	if err := addNotificationConfigToGroupsWithTopic(conn, groups, nl, topic); err != nil {
		return err
	}

	// ARNs are unique, and these notifications are per ARN, so we re-use the ARN
	// here as the ID
	d.SetId(topic)
	return resourceAwsAutoscalingNotificationRead(d, meta)
}

func resourceAwsAutoscalingNotificationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := d.Get("group_names").(*schema.Set).List()
	var groups []*string
	for _, g := range gl {
		groups = append(groups, aws.String(g.(string)))
	}

	opts := &autoscaling.DescribeNotificationConfigurationsInput{
		AutoScalingGroupNames: groups,
	}

	resp, err := conn.DescribeNotificationConfigurations(opts)
	if err != nil {
		return fmt.Errorf("Error describing notifications")
	}

	// grab all applicable notifcation configurations for this Topic
	gRaw := make(map[string]bool)
	nRaw := make(map[string]bool)
	topic := d.Get("topic_arn").(string)
	for _, n := range resp.NotificationConfigurations {
		if *n.TopicARN == topic {
			gRaw[*n.AutoScalingGroupName] = true
			nRaw[*n.NotificationType] = true
		}
	}

	var gList []string
	for k, _ := range gRaw {
		gList = append(gList, k)
	}
	var nList []string
	for k, _ := range nRaw {
		nList = append(nList, k)
	}

	sort.Strings(gList)
	sort.Strings(nList)

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

	nl := getNofiticationList(d.Get("notifications").([]interface{}))

	o, n := d.GetChange("group_names")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)
	remove := os.Difference(ns).List()
	add := ns.Difference(os).List()

	topic := d.Get("topic_arn").(string)

	if err := removeNotificationConfigToGroupsWithTopic(conn, remove, topic); err != nil {
		return err
	}

	var update []interface{}
	if d.HasChange("notifications") {
		for _, g := range d.Get("group_names").(*schema.Set).List() {
			update = append(update, g)
		}
	} else {
		update = add
	}

	if err := addNotificationConfigToGroupsWithTopic(conn, update, nl, topic); err != nil {
		return err
	}

	return resourceAwsAutoscalingNotificationRead(d, meta)
}

func addNotificationConfigToGroupsWithTopic(conn *autoscaling.AutoScaling, groups []interface{}, nl []*string, topic string) error {
	for _, a := range groups {
		opts := &autoscaling.PutNotificationConfigurationInput{
			AutoScalingGroupName: aws.String(a.(string)),
			NotificationTypes:    nl,
			TopicARN:             aws.String(topic),
		}

		_, err := conn.PutNotificationConfiguration(opts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				return fmt.Errorf("[WARN] Error creating Autoscaling Group Notification for Group %s, error: \"%s\", code: \"%s\"", a.(string), awsErr.Message(), awsErr.Code())
			}
			return err
		}
	}
	return nil
}

func removeNotificationConfigToGroupsWithTopic(conn *autoscaling.AutoScaling, groups []interface{}, topic string) error {
	for _, r := range groups {
		opts := &autoscaling.DeleteNotificationConfigurationInput{
			AutoScalingGroupName: aws.String(r.(string)),
			TopicARN:             aws.String(topic),
		}

		_, err := conn.DeleteNotificationConfiguration(opts)
		if err != nil {
			return fmt.Errorf("[WARN] Error deleting notification configuration for ASG \"%s\", Topic ARN \"%s\"", r.(string), topic)
		}
	}
	return nil
}

func resourceAwsAutoscalingNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	gl := d.Get("group_names").(*schema.Set).List()
	var groups []interface{}
	for _, g := range gl {
		groups = append(groups, g)
	}

	topic := d.Get("topic_arn").(string)
	if err := removeNotificationConfigToGroupsWithTopic(conn, groups, topic); err != nil {
		return err
	}

	return nil
}

func buildNotificationTypesSlice(l []string) (nl []*string) {
	for _, n := range l {
		if !strings.HasPrefix(n, "autoscaling:") {
			nl = append(nl, aws.String("autoscaling:"+n))
		} else {
			nl = append(nl, aws.String(n))
		}
	}
	return nl
}

func getNofiticationList(l []interface{}) (nl []*string) {
	var notifications []string
	for _, n := range l {
		notifications = append(notifications, n.(string))
	}

	return buildNotificationTypesSlice(notifications)
}

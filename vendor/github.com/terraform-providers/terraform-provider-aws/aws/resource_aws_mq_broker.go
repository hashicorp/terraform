package aws

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/copystructure"
)

func resourceAwsMqBroker() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsMqBrokerCreate,
		Read:   resourceAwsMqBrokerRead,
		Update: resourceAwsMqBrokerUpdate,
		Delete: resourceAwsMqBrokerDelete,

		Schema: map[string]*schema.Schema{
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"broker_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"revision": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"deployment_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "SINGLE_INSTANCE",
				ForceNew: true,
			},
			"engine_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"host_instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"logs": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				// Ignore missing configuration block
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"general": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"audit": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"maintenance_window_start_time": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"day_of_week": {
							Type:     schema.TypeString,
							Required: true,
						},
						"time_of_day": {
							Type:     schema.TypeString,
							Required: true,
						},
						"time_zone": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"user": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      resourceAwsMqUserHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_access": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"groups": {
							Type:     schema.TypeSet,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
							Optional: true,
						},
						"password": {
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							ValidateFunc: validateMqBrokerPassword,
						},
						"username": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_url": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"endpoints": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsMqBrokerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn

	name := d.Get("broker_name").(string)
	requestId := resource.PrefixedUniqueId(fmt.Sprintf("tf-%s", name))
	input := mq.CreateBrokerRequest{
		AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		BrokerName:              aws.String(name),
		CreatorRequestId:        aws.String(requestId),
		EngineType:              aws.String(d.Get("engine_type").(string)),
		EngineVersion:           aws.String(d.Get("engine_version").(string)),
		HostInstanceType:        aws.String(d.Get("host_instance_type").(string)),
		PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
		SecurityGroups:          expandStringSet(d.Get("security_groups").(*schema.Set)),
		Users:                   expandMqUsers(d.Get("user").(*schema.Set).List()),
		Logs:                    expandMqLogs(d.Get("logs").([]interface{})),
	}

	if v, ok := d.GetOk("configuration"); ok {
		input.Configuration = expandMqConfigurationId(v.([]interface{}))
	}
	if v, ok := d.GetOk("deployment_mode"); ok {
		input.DeploymentMode = aws.String(v.(string))
	}
	if v, ok := d.GetOk("maintenance_window_start_time"); ok {
		input.MaintenanceWindowStartTime = expandMqWeeklyStartTime(v.([]interface{}))
	}
	if v, ok := d.GetOk("subnet_ids"); ok {
		input.SubnetIds = expandStringList(v.(*schema.Set).List())
	}
	if v, ok := d.GetOk("tags"); ok {
		input.Tags = tagsFromMapGeneric(v.(map[string]interface{}))
	}

	log.Printf("[INFO] Creating MQ Broker: %s", input)
	out, err := conn.CreateBroker(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.BrokerId)
	d.Set("arn", out.BrokerArn)

	stateConf := resource.StateChangeConf{
		Pending: []string{
			mq.BrokerStateCreationInProgress,
			mq.BrokerStateRebootInProgress,
		},
		Target:  []string{mq.BrokerStateRunning},
		Timeout: 30 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeBroker(&mq.DescribeBrokerInput{
				BrokerId: aws.String(d.Id()),
			})
			if err != nil {
				return 42, "", err
			}

			return out, *out.BrokerState, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsMqBrokerRead(d, meta)
}

func resourceAwsMqBrokerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn

	log.Printf("[INFO] Reading MQ Broker: %s", d.Id())
	out, err := conn.DescribeBroker(&mq.DescribeBrokerInput{
		BrokerId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "NotFoundException", "") {
			log.Printf("[WARN] MQ Broker %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		// API docs say a 404 can also return a 403
		if isAWSErr(err, "ForbiddenException", "Forbidden") {
			log.Printf("[WARN] MQ Broker %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("auto_minor_version_upgrade", out.AutoMinorVersionUpgrade)
	d.Set("arn", out.BrokerArn)
	d.Set("instances", flattenMqBrokerInstances(out.BrokerInstances))
	d.Set("broker_name", out.BrokerName)
	d.Set("deployment_mode", out.DeploymentMode)
	d.Set("engine_type", out.EngineType)
	d.Set("engine_version", out.EngineVersion)
	d.Set("host_instance_type", out.HostInstanceType)
	d.Set("publicly_accessible", out.PubliclyAccessible)
	err = d.Set("maintenance_window_start_time", flattenMqWeeklyStartTime(out.MaintenanceWindowStartTime))
	if err != nil {
		return err
	}
	d.Set("security_groups", aws.StringValueSlice(out.SecurityGroups))
	d.Set("subnet_ids", aws.StringValueSlice(out.SubnetIds))

	if err := d.Set("logs", flattenMqLogs(out.Logs)); err != nil {
		return fmt.Errorf("error setting logs: %s", err)
	}

	err = d.Set("configuration", flattenMqConfigurationId(out.Configurations.Current))
	if err != nil {
		return err
	}

	rawUsers := make([]*mq.User, len(out.Users))
	for i, u := range out.Users {
		uOut, err := conn.DescribeUser(&mq.DescribeUserInput{
			BrokerId: aws.String(d.Id()),
			Username: u.Username,
		})
		if err != nil {
			return err
		}

		rawUsers[i] = &mq.User{
			ConsoleAccess: uOut.ConsoleAccess,
			Groups:        uOut.Groups,
			Username:      uOut.Username,
		}
	}

	users := flattenMqUsers(rawUsers, d.Get("user").(*schema.Set).List())
	if err = d.Set("user", users); err != nil {
		return err
	}

	return getTagsMQ(conn, d, aws.StringValue(out.BrokerArn))
}

func resourceAwsMqBrokerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn

	if d.HasChange("configuration") || d.HasChange("logs") {
		_, err := conn.UpdateBroker(&mq.UpdateBrokerRequest{
			BrokerId:      aws.String(d.Id()),
			Configuration: expandMqConfigurationId(d.Get("configuration").([]interface{})),
			Logs:          expandMqLogs(d.Get("logs").([]interface{})),
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("user") {
		o, n := d.GetChange("user")
		err := updateAwsMqBrokerUsers(conn, d.Id(),
			o.(*schema.Set).List(), n.(*schema.Set).List())
		if err != nil {
			return err
		}
	}

	if d.Get("apply_immediately").(bool) {
		_, err := conn.RebootBroker(&mq.RebootBrokerInput{
			BrokerId: aws.String(d.Id()),
		})
		if err != nil {
			return err
		}

		stateConf := resource.StateChangeConf{
			Pending: []string{
				mq.BrokerStateRunning,
				mq.BrokerStateRebootInProgress,
			},
			Target:  []string{mq.BrokerStateRunning},
			Timeout: 30 * time.Minute,
			Refresh: func() (interface{}, string, error) {
				out, err := conn.DescribeBroker(&mq.DescribeBrokerInput{
					BrokerId: aws.String(d.Id()),
				})
				if err != nil {
					return 42, "", err
				}

				return out, *out.BrokerState, nil
			},
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}
	}

	if tagErr := setTagsMQ(conn, d, d.Get("arn").(string)); tagErr != nil {
		return fmt.Errorf("error setting mq broker tags: %s", tagErr)
	}

	return nil
}

func resourceAwsMqBrokerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn

	log.Printf("[INFO] Deleting MQ Broker: %s", d.Id())
	_, err := conn.DeleteBroker(&mq.DeleteBrokerInput{
		BrokerId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	return waitForMqBrokerDeletion(conn, d.Id())
}

func resourceAwsMqUserHash(v interface{}) int {
	var buf bytes.Buffer

	m := v.(map[string]interface{})
	if ca, ok := m["console_access"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", ca.(bool)))
	} else {
		buf.WriteString("false-")
	}
	if g, ok := m["groups"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", g.(*schema.Set).List()))
	}
	if p, ok := m["password"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", p.(string)))
	}
	buf.WriteString(fmt.Sprintf("%s-", m["username"].(string)))

	return hashcode.String(buf.String())
}

func waitForMqBrokerDeletion(conn *mq.MQ, id string) error {
	stateConf := resource.StateChangeConf{
		Pending: []string{
			mq.BrokerStateRunning,
			mq.BrokerStateRebootInProgress,
			mq.BrokerStateDeletionInProgress,
		},
		Target:  []string{""},
		Timeout: 30 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeBroker(&mq.DescribeBrokerInput{
				BrokerId: aws.String(id),
			})
			if err != nil {
				if isAWSErr(err, "NotFoundException", "") {
					return 42, "", nil
				}
				return 42, "", err
			}

			return out, *out.BrokerState, nil
		},
	}
	_, err := stateConf.WaitForState()
	return err
}

func updateAwsMqBrokerUsers(conn *mq.MQ, bId string, oldUsers, newUsers []interface{}) error {
	createL, deleteL, updateL, err := diffAwsMqBrokerUsers(bId, oldUsers, newUsers)
	if err != nil {
		return err
	}

	for _, c := range createL {
		_, err := conn.CreateUser(c)
		if err != nil {
			return err
		}
	}
	for _, d := range deleteL {
		_, err := conn.DeleteUser(d)
		if err != nil {
			return err
		}
	}
	for _, u := range updateL {
		_, err := conn.UpdateUser(u)
		if err != nil {
			return err
		}
	}

	return nil
}

func diffAwsMqBrokerUsers(bId string, oldUsers, newUsers []interface{}) (
	cr []*mq.CreateUserRequest, di []*mq.DeleteUserInput, ur []*mq.UpdateUserRequest, e error) {

	existingUsers := make(map[string]interface{})
	for _, ou := range oldUsers {
		u := ou.(map[string]interface{})
		username := u["username"].(string)
		// Convert Set to slice to allow easier comparison
		if g, ok := u["groups"]; ok {
			groups := g.(*schema.Set).List()
			u["groups"] = groups
		}

		existingUsers[username] = u
	}

	for _, nu := range newUsers {
		// Still need access to the original map
		// because Set contents doesn't get copied
		// Likely related to https://github.com/mitchellh/copystructure/issues/17
		nuOriginal := nu.(map[string]interface{})

		// Create a mutable copy
		newUser, err := copystructure.Copy(nu)
		if err != nil {
			e = err
			return
		}

		newUserMap := newUser.(map[string]interface{})
		username := newUserMap["username"].(string)

		// Convert Set to slice to allow easier comparison
		var ng []interface{}
		if g, ok := nuOriginal["groups"]; ok {
			ng = g.(*schema.Set).List()
			newUserMap["groups"] = ng
		}

		if eu, ok := existingUsers[username]; ok {

			existingUserMap := eu.(map[string]interface{})

			if !reflect.DeepEqual(existingUserMap, newUserMap) {
				ur = append(ur, &mq.UpdateUserRequest{
					BrokerId:      aws.String(bId),
					ConsoleAccess: aws.Bool(newUserMap["console_access"].(bool)),
					Groups:        expandStringList(ng),
					Password:      aws.String(newUserMap["password"].(string)),
					Username:      aws.String(username),
				})
			}

			// Delete after processing, so we know what's left for deletion
			delete(existingUsers, username)
		} else {
			cur := &mq.CreateUserRequest{
				BrokerId:      aws.String(bId),
				ConsoleAccess: aws.Bool(newUserMap["console_access"].(bool)),
				Password:      aws.String(newUserMap["password"].(string)),
				Username:      aws.String(username),
			}
			if len(ng) > 0 {
				cur.Groups = expandStringList(ng)
			}
			cr = append(cr, cur)
		}
	}

	for username := range existingUsers {
		di = append(di, &mq.DeleteUserInput{
			BrokerId: aws.String(bId),
			Username: aws.String(username),
		})
	}

	return
}

func validateMqBrokerPassword(v interface{}, k string) (ws []string, errors []error) {
	min := 12
	max := 250
	value := v.(string)
	unique := make(map[string]bool)

	for _, v := range value {
		if _, ok := unique[string(v)]; ok {
			continue
		}
		if string(v) == "," {
			errors = append(errors, fmt.Errorf("%q must not contain commas", k))
		}
		unique[string(v)] = true
	}
	if len(unique) < 4 {
		errors = append(errors, fmt.Errorf("%q must contain at least 4 unique characters", k))
	}
	if len(value) < min || len(value) > max {
		errors = append(errors, fmt.Errorf(
			"%q must be %d to %d characters long. provided string length: %d", k, min, max, len(value)))
	}
	return
}

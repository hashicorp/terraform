package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/glacier"
)

func resourceAwsGlacierVault() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlacierVaultCreate,
		Read:   resourceAwsGlacierVaultRead,
		Update: resourceAwsGlacierVaultUpdate,
		Delete: resourceAwsGlacierVaultDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[.0-9A-Za-z-_]+$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"only alphanumeric characters, hyphens, underscores, and periods are allowed in %q", k))
					}
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"access_policy": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := normalizeJsonString(v)
					return json
				},
			},

			"notification": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"events": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"sns_topic": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsGlacierVaultCreate(d *schema.ResourceData, meta interface{}) error {
	glacierconn := meta.(*AWSClient).glacierconn

	input := &glacier.CreateVaultInput{
		VaultName: aws.String(d.Get("name").(string)),
	}

	out, err := glacierconn.CreateVault(input)
	if err != nil {
		return fmt.Errorf("Error creating Glacier Vault: %s", err)
	}

	d.SetId(d.Get("name").(string))
	d.Set("location", *out.Location)

	return resourceAwsGlacierVaultUpdate(d, meta)
}

func resourceAwsGlacierVaultUpdate(d *schema.ResourceData, meta interface{}) error {
	glacierconn := meta.(*AWSClient).glacierconn

	if err := setGlacierVaultTags(glacierconn, d); err != nil {
		return err
	}

	if d.HasChange("access_policy") {
		if err := resourceAwsGlacierVaultPolicyUpdate(glacierconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("notification") {
		if err := resourceAwsGlacierVaultNotificationUpdate(glacierconn, d); err != nil {
			return err
		}
	}

	return resourceAwsGlacierVaultRead(d, meta)
}

func resourceAwsGlacierVaultRead(d *schema.ResourceData, meta interface{}) error {
	glacierconn := meta.(*AWSClient).glacierconn

	input := &glacier.DescribeVaultInput{
		VaultName: aws.String(d.Id()),
	}

	out, err := glacierconn.DescribeVault(input)
	if err != nil {
		return fmt.Errorf("Error reading Glacier Vault: %s", err.Error())
	}

	awsClient := meta.(*AWSClient)
	d.Set("name", out.VaultName)
	d.Set("arn", out.VaultARN)

	location, err := buildGlacierVaultLocation(awsClient.accountid, d.Id())
	if err != nil {
		return err
	}
	d.Set("location", location)

	tags, err := getGlacierVaultTags(glacierconn, d.Id())
	if err != nil {
		return err
	}
	d.Set("tags", tags)

	log.Printf("[DEBUG] Getting the access_policy for Vault %s", d.Id())
	pol, err := glacierconn.GetVaultAccessPolicy(&glacier.GetVaultAccessPolicyInput{
		VaultName: aws.String(d.Id()),
	})

	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "ResourceNotFoundException" {
		d.Set("access_policy", "")
	} else if pol != nil {
		policy, err := normalizeJsonString(*pol.Policy.Policy)
		if err != nil {
			return errwrap.Wrapf("access policy contains an invalid JSON: {{err}}", err)
		}
		d.Set("access_policy", policy)
	} else {
		return err
	}

	notifications, err := getGlacierVaultNotification(glacierconn, d.Id())
	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "ResourceNotFoundException" {
		d.Set("notification", "")
	} else if pol != nil {
		d.Set("notification", notifications)
	} else {
		return err
	}

	return nil
}

func resourceAwsGlacierVaultDelete(d *schema.ResourceData, meta interface{}) error {
	glacierconn := meta.(*AWSClient).glacierconn

	log.Printf("[DEBUG] Glacier Delete Vault: %s", d.Id())
	_, err := glacierconn.DeleteVault(&glacier.DeleteVaultInput{
		VaultName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting Glacier Vault: %s", err.Error())
	}
	return nil
}

func resourceAwsGlacierVaultNotificationUpdate(glacierconn *glacier.Glacier, d *schema.ResourceData) error {

	if v, ok := d.GetOk("notification"); ok {
		settings := v.([]interface{})

		if len(settings) > 1 {
			return fmt.Errorf("Only a single Notification Block is allowed for Glacier Vault")
		} else if len(settings) == 1 {
			s := settings[0].(map[string]interface{})
			var events []*string
			for _, id := range s["events"].(*schema.Set).List() {
				events = append(events, aws.String(id.(string)))
			}

			_, err := glacierconn.SetVaultNotifications(&glacier.SetVaultNotificationsInput{
				VaultName: aws.String(d.Id()),
				VaultNotificationConfig: &glacier.VaultNotificationConfig{
					SNSTopic: aws.String(s["sns_topic"].(string)),
					Events:   events,
				},
			})

			if err != nil {
				return fmt.Errorf("Error Updating Glacier Vault Notifications: %s", err.Error())
			}
		}
	} else {
		_, err := glacierconn.DeleteVaultNotifications(&glacier.DeleteVaultNotificationsInput{
			VaultName: aws.String(d.Id()),
		})

		if err != nil {
			return fmt.Errorf("Error Removing Glacier Vault Notifications: %s", err.Error())
		}

	}

	return nil
}

func resourceAwsGlacierVaultPolicyUpdate(glacierconn *glacier.Glacier, d *schema.ResourceData) error {
	vaultName := d.Id()
	policyContents := d.Get("access_policy").(string)

	policy := &glacier.VaultAccessPolicy{
		Policy: aws.String(policyContents),
	}

	if policyContents != "" {
		log.Printf("[DEBUG] Glacier Vault: %s, put policy", vaultName)

		_, err := glacierconn.SetVaultAccessPolicy(&glacier.SetVaultAccessPolicyInput{
			VaultName: aws.String(d.Id()),
			Policy:    policy,
		})

		if err != nil {
			return fmt.Errorf("Error putting Glacier Vault policy: %s", err.Error())
		}
	} else {
		log.Printf("[DEBUG] Glacier Vault: %s, delete policy: %s", vaultName, policy)
		_, err := glacierconn.DeleteVaultAccessPolicy(&glacier.DeleteVaultAccessPolicyInput{
			VaultName: aws.String(d.Id()),
		})

		if err != nil {
			return fmt.Errorf("Error deleting Glacier Vault policy: %s", err.Error())
		}
	}

	return nil
}

func setGlacierVaultTags(conn *glacier.Glacier, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffGlacierVaultTags(mapGlacierVaultTags(o), mapGlacierVaultTags(n))

		// Set tags
		if len(remove) > 0 {
			tagsToRemove := &glacier.RemoveTagsFromVaultInput{
				VaultName: aws.String(d.Id()),
				TagKeys:   glacierStringsToPointyString(remove),
			}

			log.Printf("[DEBUG] Removing tags: from %s", d.Id())
			_, err := conn.RemoveTagsFromVault(tagsToRemove)
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			tagsToAdd := &glacier.AddTagsToVaultInput{
				VaultName: aws.String(d.Id()),
				Tags:      glacierVaultTagsFromMap(create),
			}

			log.Printf("[DEBUG] Creating tags: for %s", d.Id())
			_, err := conn.AddTagsToVault(tagsToAdd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func mapGlacierVaultTags(m map[string]interface{}) map[string]string {
	results := make(map[string]string)
	for k, v := range m {
		results[k] = v.(string)
	}

	return results
}

func diffGlacierVaultTags(oldTags, newTags map[string]string) (map[string]string, []string) {

	create := make(map[string]string)
	for k, v := range newTags {
		create[k] = v
	}

	// Build the list of what to remove
	var remove []string
	for k, v := range oldTags {
		old, ok := create[k]
		if !ok || old != v {
			// Delete it!
			remove = append(remove, k)
		}
	}

	return create, remove
}

func getGlacierVaultTags(glacierconn *glacier.Glacier, vaultName string) (map[string]string, error) {
	request := &glacier.ListTagsForVaultInput{
		VaultName: aws.String(vaultName),
	}

	log.Printf("[DEBUG] Getting the tags: for %s", vaultName)
	response, err := glacierconn.ListTagsForVault(request)
	if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "NoSuchTagSet" {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}

	return glacierVaultTagsToMap(response.Tags), nil
}

func glacierVaultTagsToMap(responseTags map[string]*string) map[string]string {
	results := make(map[string]string, len(responseTags))
	for k, v := range responseTags {
		results[k] = *v
	}

	return results
}

func glacierVaultTagsFromMap(responseTags map[string]string) map[string]*string {
	results := make(map[string]*string, len(responseTags))
	for k, v := range responseTags {
		results[k] = aws.String(v)
	}

	return results
}

func glacierStringsToPointyString(s []string) []*string {
	results := make([]*string, len(s))
	for i, x := range s {
		results[i] = aws.String(x)
	}

	return results
}

func glacierPointersToStringList(pointers []*string) []interface{} {
	list := make([]interface{}, len(pointers))
	for i, v := range pointers {
		list[i] = *v
	}
	return list
}

func buildGlacierVaultLocation(accountId, vaultName string) (string, error) {
	if accountId == "" {
		return "", errors.New("AWS account ID unavailable - failed to construct Vault location")
	}
	return fmt.Sprintf("/" + accountId + "/vaults/" + vaultName), nil
}

func getGlacierVaultNotification(glacierconn *glacier.Glacier, vaultName string) ([]map[string]interface{}, error) {
	request := &glacier.GetVaultNotificationsInput{
		VaultName: aws.String(vaultName),
	}

	response, err := glacierconn.GetVaultNotifications(request)
	if err != nil {
		return nil, fmt.Errorf("Error reading Glacier Vault Notifications: %s", err.Error())
	}

	notifications := make(map[string]interface{}, 0)

	log.Print("[DEBUG] Flattening Glacier Vault Notifications")

	notifications["events"] = schema.NewSet(schema.HashString, glacierPointersToStringList(response.VaultNotificationConfig.Events))
	notifications["sns_topic"] = *response.VaultNotificationConfig.SNSTopic

	return []map[string]interface{}{notifications}, nil
}

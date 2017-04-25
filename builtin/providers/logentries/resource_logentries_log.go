package logentries

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	logentries "github.com/logentries/le_goclient"
)

func resourceLogentriesLog() *schema.Resource {

	return &schema.Resource{
		Create: resourceLogentriesLogCreate,
		Read:   resourceLogentriesLogRead,
		Update: resourceLogentriesLogUpdate,
		Delete: resourceLogentriesLogDelete,

		Schema: map[string]*schema.Schema{
			"token": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"logset_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"filename": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"retention_period": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ACCOUNT_DEFAULT",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					allowed_values := []string{"1W", "2W", "1M", "2M", "6M", "1Y", "2Y", "UNLIMITED", "ACCOUNT_DEFAULT"}
					if !sliceContains(value, allowed_values) {
						errors = append(errors, fmt.Errorf("Invalid retention period: %s (must be one of: %s)", value, allowed_values))
					}
					return
				},
			},
			"source": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "token",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					allowed_values := []string{"token", "syslog", "agent", "api"}
					if !sliceContains(value, allowed_values) {
						errors = append(errors, fmt.Errorf("Invalid log source option: %s (must be one of: %s)", value, allowed_values))
					}
					return
				},
			},
			"type": {
				Type:     schema.TypeString,
				Default:  "",
				Optional: true,
			},
		},
	}
}

func resourceLogentriesLogCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	retentionPeriod, err := retentionPeriodForEnum(d.Get("retention_period").(string))
	if err != nil {
		return err
	}
	res, err := client.Log.Create(logentries.LogCreateRequest{
		LogSetKey: d.Get("logset_id").(string),
		Name:      d.Get("name").(string),
		Retention: strconv.FormatInt(retentionPeriod, 10),
		Type:      d.Get("type").(string),
		Source:    d.Get("source").(string),
		Filename:  d.Get("filename").(string),
	})

	if err != nil {
		return err
	}

	d.SetId(res.Key)

	return mapLogToSchema(client, res, d)
}

func resourceLogentriesLogRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	res, err := client.Log.Read(logentries.LogReadRequest{
		LogSetKey: d.Get("logset_id").(string),
		Key:       d.Id(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("Logentries Log Not Found - Refreshing from State")
			d.SetId("")
			return nil
		}
		return err
	}

	if res == nil {
		d.SetId("")
		return nil
	}

	return mapLogToSchema(client, res, d)
}

func resourceLogentriesLogUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	_, err := client.Log.Update(logentries.LogUpdateRequest{
		Key:       d.Id(),
		Name:      d.Get("name").(string),
		Retention: d.Get("retention_period").(string),
		Type:      d.Get("type").(string),
		Source:    d.Get("source").(string),
		Filename:  d.Get("filename").(string),
	})
	if err != nil {
		return err
	}

	return resourceLogentriesLogRead(d, meta)
}

func resourceLogentriesLogDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*logentries.Client)
	err := client.Log.Delete(logentries.LogDeleteRequest{
		LogSetKey: d.Get("logset_id").(string),
		Key:       d.Id(),
	})
	return err
}

func mapLogToSchema(client *logentries.Client, log *logentries.Log, d *schema.ResourceData) error {
	d.Set("token", log.Token)
	d.Set("name", log.Name)
	d.Set("filename", log.Filename)
	retentionEnum, err := enumForRetentionPeriod(log.Retention)
	if err != nil {
		return err
	}
	d.Set("retention_period", retentionEnum)
	d.Set("source", log.Source)
	if log.Type != "" {
		logTypes, err := client.LogType.ReadDefault(logentries.LogTypeListRequest{})
		if err != nil {
			return err
		}
		logType := lookupTypeShortcut(log.Type, logTypes)
		if logType == "" {
			logTypes, err = client.LogType.Read(logentries.LogTypeListRequest{})
			if err != nil {
				return err
			}
			logType = lookupTypeShortcut(log.Type, logTypes)
		}
		d.Set("type", logType)
	}

	return nil
}

func enumForRetentionPeriod(retentionPeriod int64) (string, error) {
	switch retentionPeriod {
	case 604800000:
		return "1W", nil
	case 1209600000:
		return "2W", nil
	case 2678400000:
		return "1M", nil
	case 5356800000:
		return "2M", nil
	case 16070400000:
		return "6M", nil
	case 31536000000:
		return "1Y", nil
	case 63072000000:
		return "2Y", nil
	case 0:
		return "UNLIMITED", nil
	case -1:
		return "ACCOUNT_DEFAULT", nil
	}

	return "", fmt.Errorf("Unknown retention period: %d", retentionPeriod)
}

func retentionPeriodForEnum(retentionPeriodEnum string) (int64, error) {
	switch retentionPeriodEnum {
	case "1W":
		return 604800000, nil
	case "2W":
		return 1209600000, nil
	case "1M":
		return 2678400000, nil
	case "2M":
		return 5356800000, nil
	case "6M":
		return 16070400000, nil
	case "1Y":
		return 31536000000, nil
	case "2Y":
		return 63072000000, nil
	case "UNLIMITED":
		return 0, nil
	case "ACCOUNT_DEFAULT":
		return -1, nil
	}

	return 0, fmt.Errorf("Unknown retention period: %s", retentionPeriodEnum)
}

func lookupTypeShortcut(currentLogKey string, logTypes []logentries.LogType) string {
	for _, logType := range logTypes {
		if logType.Key == currentLogKey {
			return logType.Shortcut
		}
	}
	return ""
}

func sliceContains(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

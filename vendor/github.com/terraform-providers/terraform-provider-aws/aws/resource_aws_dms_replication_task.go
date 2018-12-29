package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDmsReplicationTask() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDmsReplicationTaskCreate,
		Read:   resourceAwsDmsReplicationTaskRead,
		Update: resourceAwsDmsReplicationTaskUpdate,
		Delete: resourceAwsDmsReplicationTaskDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"cdc_start_time": {
				Type:     schema.TypeString,
				Optional: true,
				// Requires a Unix timestamp in seconds. Example 1484346880
			},
			"migration_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					dms.MigrationTypeValueFullLoad,
					dms.MigrationTypeValueCdc,
					dms.MigrationTypeValueFullLoadAndCdc,
				}, false),
			},
			"replication_instance_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"replication_task_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"replication_task_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDmsReplicationTaskId,
			},
			"replication_task_settings": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
			},
			"source_endpoint_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"table_mappings": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"target_endpoint_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
		},
	}
}

func resourceAwsDmsReplicationTaskCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.CreateReplicationTaskInput{
		MigrationType:             aws.String(d.Get("migration_type").(string)),
		ReplicationInstanceArn:    aws.String(d.Get("replication_instance_arn").(string)),
		ReplicationTaskIdentifier: aws.String(d.Get("replication_task_id").(string)),
		SourceEndpointArn:         aws.String(d.Get("source_endpoint_arn").(string)),
		TableMappings:             aws.String(d.Get("table_mappings").(string)),
		Tags:                      dmsTagsFromMap(d.Get("tags").(map[string]interface{})),
		TargetEndpointArn:         aws.String(d.Get("target_endpoint_arn").(string)),
	}

	if v, ok := d.GetOk("cdc_start_time"); ok {
		seconds, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			return fmt.Errorf("DMS create replication task. Invalid CDC Unix timestamp: %s", err)
		}
		request.CdcStartTime = aws.Time(time.Unix(seconds, 0))
	}

	if v, ok := d.GetOk("replication_task_settings"); ok {
		request.ReplicationTaskSettings = aws.String(v.(string))
	}

	log.Println("[DEBUG] DMS create replication task:", request)

	_, err := conn.CreateReplicationTask(request)
	if err != nil {
		return err
	}

	taskId := d.Get("replication_task_id").(string)
	d.SetId(taskId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"ready"},
		Refresh:    resourceAwsDmsReplicationTaskStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDmsReplicationTaskRead(d, meta)
}

func resourceAwsDmsReplicationTaskRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	response, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-task-id"),
				Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
			},
		},
	})
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Task %q Not Found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = resourceAwsDmsReplicationTaskSetState(d, response.ReplicationTasks[0])
	if err != nil {
		return err
	}

	tagsResp, err := conn.ListTagsForResource(&dms.ListTagsForResourceInput{
		ResourceArn: aws.String(d.Get("replication_task_arn").(string)),
	})
	if err != nil {
		return err
	}
	d.Set("tags", dmsTagsToMap(tagsResp.TagList))

	return nil
}

func resourceAwsDmsReplicationTaskUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.ModifyReplicationTaskInput{
		ReplicationTaskArn: aws.String(d.Get("replication_task_arn").(string)),
	}
	hasChanges := false

	if d.HasChange("cdc_start_time") {
		seconds, err := strconv.ParseInt(d.Get("cdc_start_time").(string), 10, 64)
		if err != nil {
			return fmt.Errorf("DMS update replication task. Invalid CRC Unix timestamp: %s", err)
		}
		request.CdcStartTime = aws.Time(time.Unix(seconds, 0))
		hasChanges = true
	}

	if d.HasChange("migration_type") {
		request.MigrationType = aws.String(d.Get("migration_type").(string))
		hasChanges = true
	}

	if d.HasChange("replication_task_settings") {
		request.ReplicationTaskSettings = aws.String(d.Get("replication_task_settings").(string))
		hasChanges = true
	}

	if d.HasChange("table_mappings") {
		request.TableMappings = aws.String(d.Get("table_mappings").(string))
		hasChanges = true
	}

	if d.HasChange("tags") {
		err := dmsSetTags(d.Get("replication_task_arn").(string), d, meta)
		if err != nil {
			return err
		}
	}

	if hasChanges {
		log.Println("[DEBUG] DMS update replication task:", request)

		_, err := conn.ModifyReplicationTask(request)
		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"ready", "stopped", "failed"},
			Refresh:    resourceAwsDmsReplicationTaskStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

		return resourceAwsDmsReplicationTaskRead(d, meta)
	}

	return nil
}

func resourceAwsDmsReplicationTaskDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.DeleteReplicationTaskInput{
		ReplicationTaskArn: aws.String(d.Get("replication_task_arn").(string)),
	}

	log.Printf("[DEBUG] DMS delete replication task: %#v", request)

	_, err := conn.DeleteReplicationTask(request)
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Task %q Not Found", d.Id())
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDmsReplicationTaskStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsDmsReplicationTaskSetState(d *schema.ResourceData, task *dms.ReplicationTask) error {
	d.SetId(*task.ReplicationTaskIdentifier)

	d.Set("migration_type", task.MigrationType)
	d.Set("replication_instance_arn", task.ReplicationInstanceArn)
	d.Set("replication_task_arn", task.ReplicationTaskArn)
	d.Set("replication_task_id", task.ReplicationTaskIdentifier)
	d.Set("replication_task_settings", task.ReplicationTaskSettings)
	d.Set("source_endpoint_arn", task.SourceEndpointArn)
	d.Set("table_mappings", task.TableMappings)
	d.Set("target_endpoint_arn", task.TargetEndpointArn)

	return nil
}

func resourceAwsDmsReplicationTaskStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).dmsconn

		v, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-task-id"),
					Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
				},
			},
		})
		if err != nil {
			if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
				return nil, "", nil
			}
			log.Printf("Error on retrieving DMS Replication Task when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.ReplicationTasks != nil {
			log.Printf("[DEBUG] DMS Replication Task status for instance %s: %s", d.Id(), *v.ReplicationTasks[0].Status)
		}

		return v, *v.ReplicationTasks[0].Status, nil
	}
}

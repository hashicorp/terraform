package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/private/waiter"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
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
				Type:     schema.TypeInt,
				Optional: true,
				// Requires a Unix timestamp in seconds. Example 1484346880
			},
			"migration_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"full-load",
					"cdc",
					"full-load-and-cdc",
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
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
			},
			"source_endpoint_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"table_mappings": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateJsonString,
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
			return fmt.Errorf("[ERROR] DMS create replication task. Invalid CDC Unix timestamp: %s", err)
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

	err = waitForTaskCreated(conn, taskId, 30, 10)
	if err != nil {
		return err
	}

	d.SetId(taskId)
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
			return fmt.Errorf("[ERROR] DMS update replication task. Invalid CRC Unix timestamp: %s", err)
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

		err = waitForTaskUpdated(conn, d.Get("replication_task_id").(string), 30, 10)
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
		return err
	}

	waitErr := waitForTaskDeleted(conn, d.Get("replication_task_id").(string), 30, 10)
	if waitErr != nil {
		return waitErr
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

func waitForTaskCreated(client *dms.DatabaseMigrationService, id string, delay int, maxAttempts int) error {
	input := &dms.DescribeReplicationTasksInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-task-id"),
				Values: []*string{aws.String(id)},
			},
		},
	}

	config := waiter.Config{
		Operation:   "DescribeReplicationTasks",
		Delay:       delay,
		MaxAttempts: maxAttempts,
		Acceptors: []waiter.WaitAcceptor{
			{
				State:    "retry",
				Matcher:  "pathAll",
				Argument: "ReplicationTasks[].Status",
				Expected: "creating",
			},
			{
				State:    "success",
				Matcher:  "pathAll",
				Argument: "ReplicationTasks[].Status",
				Expected: "ready",
			},
		},
	}

	w := waiter.Waiter{
		Client: client,
		Input:  input,
		Config: config,
	}

	return w.Wait()
}

func waitForTaskUpdated(client *dms.DatabaseMigrationService, id string, delay int, maxAttempts int) error {
	input := &dms.DescribeReplicationTasksInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-task-id"),
				Values: []*string{aws.String(id)},
			},
		},
	}

	config := waiter.Config{
		Operation:   "DescribeReplicationTasks",
		Delay:       delay,
		MaxAttempts: maxAttempts,
		Acceptors: []waiter.WaitAcceptor{
			{
				State:    "retry",
				Matcher:  "pathAll",
				Argument: "ReplicationTasks[].Status",
				Expected: "modifying",
			},
			{
				State:    "success",
				Matcher:  "pathAll",
				Argument: "ReplicationTasks[].Status",
				Expected: "ready",
			},
		},
	}

	w := waiter.Waiter{
		Client: client,
		Input:  input,
		Config: config,
	}

	return w.Wait()
}

func waitForTaskDeleted(client *dms.DatabaseMigrationService, id string, delay int, maxAttempts int) error {
	input := &dms.DescribeReplicationTasksInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-task-id"),
				Values: []*string{aws.String(id)},
			},
		},
	}

	config := waiter.Config{
		Operation:   "DescribeReplicationTasks",
		Delay:       delay,
		MaxAttempts: maxAttempts,
		Acceptors: []waiter.WaitAcceptor{
			{
				State:    "retry",
				Matcher:  "pathAll",
				Argument: "ReplicationTasks[].Status",
				Expected: "deleting",
			},
			{
				State:    "success",
				Matcher:  "path",
				Argument: "length(ReplicationTasks[]) > `0`",
				Expected: false,
			},
		},
	}

	w := waiter.Waiter{
		Client: client,
		Input:  input,
		Config: config,
	}

	return w.Wait()
}

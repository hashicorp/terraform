package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDynamoDbGlobalTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDynamoDbGlobalTableCreate,
		Read:   resourceAwsDynamoDbGlobalTableRead,
		Update: resourceAwsDynamoDbGlobalTableUpdate,
		Delete: resourceAwsDynamoDbGlobalTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsDynamoDbGlobalTableName,
			},

			"replica": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region_name": {
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
		},
	}
}

func resourceAwsDynamoDbGlobalTableCreate(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	globalTableName := d.Get("name").(string)

	input := &dynamodb.CreateGlobalTableInput{
		GlobalTableName:  aws.String(globalTableName),
		ReplicationGroup: expandAwsDynamoDbReplicas(d.Get("replica").(*schema.Set).List()),
	}

	log.Printf("[DEBUG] Creating DynamoDB Global Table: %#v", input)
	_, err := dynamodbconn.CreateGlobalTable(input)
	if err != nil {
		return err
	}

	d.SetId(globalTableName)

	log.Println("[INFO] Waiting for DynamoDB Global Table to be created")
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			dynamodb.GlobalTableStatusCreating,
			dynamodb.GlobalTableStatusDeleting,
			dynamodb.GlobalTableStatusUpdating,
		},
		Target: []string{
			dynamodb.GlobalTableStatusActive,
		},
		Refresh:    resourceAwsDynamoDbGlobalTableStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDynamoDbGlobalTableRead(d, meta)
}

func resourceAwsDynamoDbGlobalTableRead(d *schema.ResourceData, meta interface{}) error {
	globalTableDescription, err := resourceAwsDynamoDbGlobalTableRetrieve(d, meta)

	if err != nil {
		return err
	}
	if globalTableDescription == nil {
		log.Printf("[WARN] DynamoDB Global Table %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return flattenAwsDynamoDbGlobalTable(d, globalTableDescription)
}

func resourceAwsDynamoDbGlobalTableUpdate(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	if d.HasChange("replica") {
		o, n := d.GetChange("replica")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		replicaUpdateCreateReplicas := expandAwsDynamoDbReplicaUpdateCreateReplicas(ns.Difference(os).List())
		replicaUpdateDeleteReplicas := expandAwsDynamoDbReplicaUpdateDeleteReplicas(os.Difference(ns).List())

		replicaUpdates := make([]*dynamodb.ReplicaUpdate, 0, (len(replicaUpdateCreateReplicas) + len(replicaUpdateDeleteReplicas)))
		replicaUpdates = append(replicaUpdates, replicaUpdateCreateReplicas...)
		replicaUpdates = append(replicaUpdates, replicaUpdateDeleteReplicas...)

		input := &dynamodb.UpdateGlobalTableInput{
			GlobalTableName: aws.String(d.Id()),
			ReplicaUpdates:  replicaUpdates,
		}
		log.Printf("[DEBUG] Updating DynamoDB Global Table: %#v", input)
		if _, err := dynamodbconn.UpdateGlobalTable(input); err != nil {
			return err
		}

		log.Println("[INFO] Waiting for DynamoDB Global Table to be updated")
		stateConf := &resource.StateChangeConf{
			Pending: []string{
				dynamodb.GlobalTableStatusCreating,
				dynamodb.GlobalTableStatusDeleting,
				dynamodb.GlobalTableStatusUpdating,
			},
			Target: []string{
				dynamodb.GlobalTableStatusActive,
			},
			Refresh:    resourceAwsDynamoDbGlobalTableStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
		}
		_, err := stateConf.WaitForState()
		if err != nil {
			return err
		}
	}

	return nil
}

// Deleting a DynamoDB Global Table is represented by removing all replicas.
func resourceAwsDynamoDbGlobalTableDelete(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	input := &dynamodb.UpdateGlobalTableInput{
		GlobalTableName: aws.String(d.Id()),
		ReplicaUpdates:  expandAwsDynamoDbReplicaUpdateDeleteReplicas(d.Get("replica").(*schema.Set).List()),
	}
	log.Printf("[DEBUG] Deleting DynamoDB Global Table: %#v", input)
	if _, err := dynamodbconn.UpdateGlobalTable(input); err != nil {
		return err
	}

	log.Println("[INFO] Waiting for DynamoDB Global Table to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			dynamodb.GlobalTableStatusActive,
			dynamodb.GlobalTableStatusCreating,
			dynamodb.GlobalTableStatusDeleting,
			dynamodb.GlobalTableStatusUpdating,
		},
		Target:     []string{},
		Refresh:    resourceAwsDynamoDbGlobalTableStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsDynamoDbGlobalTableRetrieve(d *schema.ResourceData, meta interface{}) (*dynamodb.GlobalTableDescription, error) {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	input := &dynamodb.DescribeGlobalTableInput{
		GlobalTableName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Retrieving DynamoDB Global Table: %#v", input)

	output, err := dynamodbconn.DescribeGlobalTable(input)
	if err != nil {
		if isAWSErr(err, dynamodb.ErrCodeGlobalTableNotFoundException, "") {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DynamoDB Global Table: %s", err)
	}

	return output.GlobalTableDescription, nil
}

func resourceAwsDynamoDbGlobalTableStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		gtd, err := resourceAwsDynamoDbGlobalTableRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving DynamoDB Global Table when waiting: %s", err)
			return nil, "", err
		}

		if gtd == nil {
			return nil, "", nil
		}

		if gtd.GlobalTableStatus != nil {
			log.Printf("[DEBUG] Status for DynamoDB Global Table %s: %s", d.Id(), *gtd.GlobalTableStatus)
		}

		return gtd, *gtd.GlobalTableStatus, nil
	}
}

func flattenAwsDynamoDbGlobalTable(d *schema.ResourceData, globalTableDescription *dynamodb.GlobalTableDescription) error {
	var err error

	d.Set("arn", globalTableDescription.GlobalTableArn)
	d.Set("name", globalTableDescription.GlobalTableName)

	err = d.Set("replica", flattenAwsDynamoDbReplicas(globalTableDescription.ReplicationGroup))
	return err
}

func expandAwsDynamoDbReplicaUpdateCreateReplicas(configuredReplicas []interface{}) []*dynamodb.ReplicaUpdate {
	replicaUpdates := make([]*dynamodb.ReplicaUpdate, 0, len(configuredReplicas))
	for _, replicaRaw := range configuredReplicas {
		replica := replicaRaw.(map[string]interface{})
		replicaUpdates = append(replicaUpdates, expandAwsDynamoDbReplicaUpdateCreateReplica(replica))
	}
	return replicaUpdates
}

func expandAwsDynamoDbReplicaUpdateCreateReplica(configuredReplica map[string]interface{}) *dynamodb.ReplicaUpdate {
	replicaUpdate := &dynamodb.ReplicaUpdate{
		Create: &dynamodb.CreateReplicaAction{
			RegionName: aws.String(configuredReplica["region_name"].(string)),
		},
	}
	return replicaUpdate
}

func expandAwsDynamoDbReplicaUpdateDeleteReplicas(configuredReplicas []interface{}) []*dynamodb.ReplicaUpdate {
	replicaUpdates := make([]*dynamodb.ReplicaUpdate, 0, len(configuredReplicas))
	for _, replicaRaw := range configuredReplicas {
		replica := replicaRaw.(map[string]interface{})
		replicaUpdates = append(replicaUpdates, expandAwsDynamoDbReplicaUpdateDeleteReplica(replica))
	}
	return replicaUpdates
}

func expandAwsDynamoDbReplicaUpdateDeleteReplica(configuredReplica map[string]interface{}) *dynamodb.ReplicaUpdate {
	replicaUpdate := &dynamodb.ReplicaUpdate{
		Delete: &dynamodb.DeleteReplicaAction{
			RegionName: aws.String(configuredReplica["region_name"].(string)),
		},
	}
	return replicaUpdate
}

func expandAwsDynamoDbReplicas(configuredReplicas []interface{}) []*dynamodb.Replica {
	replicas := make([]*dynamodb.Replica, 0, len(configuredReplicas))
	for _, replicaRaw := range configuredReplicas {
		replica := replicaRaw.(map[string]interface{})
		replicas = append(replicas, expandAwsDynamoDbReplica(replica))
	}
	return replicas
}

func expandAwsDynamoDbReplica(configuredReplica map[string]interface{}) *dynamodb.Replica {
	replica := &dynamodb.Replica{
		RegionName: aws.String(configuredReplica["region_name"].(string)),
	}
	return replica
}

func flattenAwsDynamoDbReplicas(replicaDescriptions []*dynamodb.ReplicaDescription) []interface{} {
	replicas := []interface{}{}
	for _, replicaDescription := range replicaDescriptions {
		replicas = append(replicas, flattenAwsDynamoDbReplica(replicaDescription))
	}
	return replicas
}

func flattenAwsDynamoDbReplica(replicaDescription *dynamodb.ReplicaDescription) map[string]interface{} {
	replica := make(map[string]interface{})
	replica["region_name"] = *replicaDescription.RegionName
	return replica
}

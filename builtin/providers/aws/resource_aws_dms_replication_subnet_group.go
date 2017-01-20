package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDmsReplicationSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDmsReplicationSubnetGroupCreate,
		Read:   resourceAwsDmsReplicationSubnetGroupRead,
		Update: resourceAwsDmsReplicationSubnetGroupUpdate,
		Delete: resourceAwsDmsReplicationSubnetGroupDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"replication_subnet_group_description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"replication_subnet_group_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateDmsReplicationSubnetGroupId,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Required: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDmsReplicationSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	// NOTE: Even though "tags" is an allowed parameter there is no way to retrieve the tags currently.
	request := &dms.CreateReplicationSubnetGroupInput{
		ReplicationSubnetGroupIdentifier:  aws.String(d.Get("replication_subnet_group_id").(string)),
		ReplicationSubnetGroupDescription: aws.String(d.Get("replication_subnet_group_description").(string)),
		SubnetIds:                         expandStringList(d.Get("subnet_ids").(*schema.Set).List()),
	}

	log.Println("[DEBUG] DMS create replication subnet group:", request)

	response, err := conn.CreateReplicationSubnetGroup(request)
	if err != nil {
		return err
	}

	return resourceAwsDmsReplicationSubnetGroupSetState(d, response.ReplicationSubnetGroup)
}

func resourceAwsDmsReplicationSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	response, err := conn.DescribeReplicationSubnetGroups(&dms.DescribeReplicationSubnetGroupsInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-subnet-group-id"),
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

	return resourceAwsDmsReplicationSubnetGroupSetState(d, response.ReplicationSubnetGroups[0])
}

func resourceAwsDmsReplicationSubnetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.ModifyReplicationSubnetGroupInput{
		ReplicationSubnetGroupIdentifier:  aws.String(d.Get("replication_subnet_group_id").(string)),
		ReplicationSubnetGroupDescription: aws.String(d.Get("replication_subnet_group_description").(string)),
		SubnetIds:                         expandStringList(d.Get("subnet_ids").(*schema.Set).List()),
	}

	response, err := conn.ModifyReplicationSubnetGroup(request)
	if err != nil {
		return err
	}

	return resourceAwsDmsReplicationSubnetGroupSetState(d, response.ReplicationSubnetGroup)
}

func resourceAwsDmsReplicationSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.DeleteReplicationSubnetGroupInput{
		ReplicationSubnetGroupIdentifier: aws.String(d.Get("replication_subnet_group_id").(string)),
	}

	log.Printf("[DEBUG] DMS delete replication subnet group: %#v", request)

	_, err := conn.DeleteReplicationSubnetGroup(request)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsDmsReplicationSubnetGroupSetState(d *schema.ResourceData, group *dms.ReplicationSubnetGroup) error {
	d.SetId(*group.ReplicationSubnetGroupIdentifier)

	subnet_ids := []string{}
	for _, subnet := range group.Subnets {
		subnet_ids = append(subnet_ids, aws.StringValue(subnet.SubnetIdentifier))
	}

	d.Set("replication_subnet_group_description", group.ReplicationSubnetGroupDescription)
	d.Set("replication_subnet_group_id", group.ReplicationSubnetGroupIdentifier)
	d.Set("subnet_ids", subnet_ids)
	d.Set("vpc_id", group.VpcId)

	return nil
}

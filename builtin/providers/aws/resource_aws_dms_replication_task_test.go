package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsDmsReplicationTaskBasic(t *testing.T) {
	resourceName := "aws_dms_replication_task.dms_replication_task"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationTaskConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationTaskExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "replication_task_arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: dmsReplicationTaskConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationTaskExists(resourceName),
				),
			},
		},
	})
}

func checkDmsReplicationTaskExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).dmsconn
		resp, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-task-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		})

		if err != nil {
			return err
		}

		if resp.ReplicationTasks == nil {
			return fmt.Errorf("DMS replication task error: %v", err)
		}
		return nil
	}
}

func dmsReplicationTaskDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dms_replication_task" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).dmsconn
		resp, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-task-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		})

		if err != nil {
			return nil
		}

		if resp != nil && len(resp.ReplicationTasks) > 0 {
			return fmt.Errorf("DMS replication task still exists: %v", err)
		}
	}

	return nil
}

func dmsReplicationTaskConfig(randId string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "dms_vpc" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-test-dms-vpc-%[1]s"
	}
}

resource "aws_subnet" "dms_subnet_1" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_2" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_dms_endpoint" "dms_endpoint_source" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-source-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	server_name = "tf-test-cluster.cluster-xxxxxxx.us-west-2.rds.amazonaws.com"
	port = 3306
	username = "tftest"
	password = "tftest"
}

resource "aws_dms_endpoint" "dms_endpoint_target" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-target-%[1]s"
	endpoint_type = "target"
	engine_name = "aurora"
	server_name = "tf-test-cluster.cluster-xxxxxxx.us-west-2.rds.amazonaws.com"
	port = 3306
	username = "tftest"
	password = "tftest"
}

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_2.id}"]
}

resource "aws_dms_replication_instance" "dms_replication_instance" {
	allocated_storage = 5
	auto_minor_version_upgrade = true
	replication_instance_class = "dms.t2.micro"
	replication_instance_id = "tf-test-dms-replication-instance-%[1]s"
	preferred_maintenance_window = "sun:00:30-sun:02:30"
	publicly_accessible = false
	replication_subnet_group_id = "${aws_dms_replication_subnet_group.dms_replication_subnet_group.replication_subnet_group_id}"
}

resource "aws_dms_replication_task" "dms_replication_task" {
	migration_type = "full-load"
	replication_instance_arn = "${aws_dms_replication_instance.dms_replication_instance.replication_instance_arn}"
	replication_task_id = "tf-test-dms-replication-task-%[1]s"
	replication_task_settings = "{\"TargetMetadata\":{\"TargetSchema\":\"\",\"SupportLobs\":true,\"FullLobMode\":false,\"LobChunkSize\":0,\"LimitedSizeLobMode\":true,\"LobMaxSize\":32,\"LoadMaxFileSize\":0,\"ParallelLoadThreads\":0,\"BatchApplyEnabled\":false},\"FullLoadSettings\":{\"FullLoadEnabled\":true,\"ApplyChangesEnabled\":false,\"TargetTablePrepMode\":\"DROP_AND_CREATE\",\"CreatePkAfterFullLoad\":false,\"StopTaskCachedChangesApplied\":false,\"StopTaskCachedChangesNotApplied\":false,\"ResumeEnabled\":false,\"ResumeMinTableSize\":100000,\"ResumeOnlyClusteredPKTables\":true,\"MaxFullLoadSubTasks\":8,\"TransactionConsistencyTimeout\":600,\"CommitRate\":10000},\"Logging\":{\"EnableLogging\":false,\"LogComponents\":[{\"Id\":\"SOURCE_UNLOAD\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TARGET_LOAD\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"SOURCE_CAPTURE\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TARGET_APPLY\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TASK_MANAGER\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"}],\"CloudWatchLogGroup\":null,\"CloudWatchLogStream\":null},\"ControlTablesSettings\":{\"historyTimeslotInMinutes\":5,\"ControlSchema\":\"\",\"HistoryTimeslotInMinutes\":5,\"HistoryTableEnabled\":false,\"SuspendedTablesTableEnabled\":false,\"StatusTableEnabled\":false},\"StreamBufferSettings\":{\"StreamBufferCount\":3,\"StreamBufferSizeInMB\":8,\"CtrlStreamBufferSizeInMB\":5},\"ChangeProcessingDdlHandlingPolicy\":{\"HandleSourceTableDropped\":true,\"HandleSourceTableTruncated\":true,\"HandleSourceTableAltered\":true},\"ErrorBehavior\":{\"DataErrorPolicy\":\"LOG_ERROR\",\"DataTruncationErrorPolicy\":\"LOG_ERROR\",\"DataErrorEscalationPolicy\":\"SUSPEND_TABLE\",\"DataErrorEscalationCount\":0,\"TableErrorPolicy\":\"SUSPEND_TABLE\",\"TableErrorEscalationPolicy\":\"STOP_TASK\",\"TableErrorEscalationCount\":0,\"RecoverableErrorCount\":-1,\"RecoverableErrorInterval\":5,\"RecoverableErrorThrottling\":true,\"RecoverableErrorThrottlingMax\":1800,\"ApplyErrorDeletePolicy\":\"IGNORE_RECORD\",\"ApplyErrorInsertPolicy\":\"LOG_ERROR\",\"ApplyErrorUpdatePolicy\":\"LOG_ERROR\",\"ApplyErrorEscalationPolicy\":\"LOG_ERROR\",\"ApplyErrorEscalationCount\":0,\"FullLoadIgnoreConflicts\":true},\"ChangeProcessingTuning\":{\"BatchApplyPreserveTransaction\":true,\"BatchApplyTimeoutMin\":1,\"BatchApplyTimeoutMax\":30,\"BatchApplyMemoryLimit\":500,\"BatchSplitSize\":0,\"MinTransactionSize\":1000,\"CommitTimeout\":1,\"MemoryLimitTotal\":1024,\"MemoryKeepTime\":60,\"StatementCacheSize\":50}}"
	source_endpoint_arn = "${aws_dms_endpoint.dms_endpoint_source.endpoint_arn}"
	table_mappings = "{\"rules\":[{\"rule-type\":\"selection\",\"rule-id\":\"1\",\"rule-name\":\"1\",\"object-locator\":{\"schema-name\":\"%%\",\"table-name\":\"%%\"},\"rule-action\":\"include\"}]}"
	tags {
		Name = "tf-test-dms-replication-task-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}
	target_endpoint_arn = "${aws_dms_endpoint.dms_endpoint_target.endpoint_arn}"
}
`, randId)
}

func dmsReplicationTaskConfigUpdate(randId string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "dms_vpc" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-test-dms-vpc-%[1]s"
	}
}

resource "aws_subnet" "dms_subnet_1" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_2" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_dms_endpoint" "dms_endpoint_source" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-source-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	server_name = "tf-test-cluster.cluster-xxxxxxx.us-west-2.rds.amazonaws.com"
	port = 3306
	username = "tftest"
	password = "tftest"
}

resource "aws_dms_endpoint" "dms_endpoint_target" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-target-%[1]s"
	endpoint_type = "target"
	engine_name = "aurora"
	server_name = "tf-test-cluster.cluster-xxxxxxx.us-west-2.rds.amazonaws.com"
	port = 3306
	username = "tftest"
	password = "tftest"
}

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_2.id}"]
}

resource "aws_dms_replication_instance" "dms_replication_instance" {
	allocated_storage = 5
	auto_minor_version_upgrade = true
	replication_instance_class = "dms.t2.micro"
	replication_instance_id = "tf-test-dms-replication-instance-%[1]s"
	preferred_maintenance_window = "sun:00:30-sun:02:30"
	publicly_accessible = false
	replication_subnet_group_id = "${aws_dms_replication_subnet_group.dms_replication_subnet_group.replication_subnet_group_id}"
}

resource "aws_dms_replication_task" "dms_replication_task" {
	migration_type = "full-load"
	replication_instance_arn = "${aws_dms_replication_instance.dms_replication_instance.replication_instance_arn}"
	replication_task_id = "tf-test-dms-replication-task-%[1]s"
	replication_task_settings = "{\"TargetMetadata\":{\"TargetSchema\":\"\",\"SupportLobs\":true,\"FullLobMode\":false,\"LobChunkSize\":0,\"LimitedSizeLobMode\":true,\"LobMaxSize\":32,\"LoadMaxFileSize\":0,\"ParallelLoadThreads\":0,\"BatchApplyEnabled\":false},\"FullLoadSettings\":{\"FullLoadEnabled\":true,\"ApplyChangesEnabled\":false,\"TargetTablePrepMode\":\"DROP_AND_CREATE\",\"CreatePkAfterFullLoad\":false,\"StopTaskCachedChangesApplied\":false,\"StopTaskCachedChangesNotApplied\":false,\"ResumeEnabled\":false,\"ResumeMinTableSize\":100000,\"ResumeOnlyClusteredPKTables\":true,\"MaxFullLoadSubTasks\":7,\"TransactionConsistencyTimeout\":600,\"CommitRate\":10000},\"Logging\":{\"EnableLogging\":false,\"LogComponents\":[{\"Id\":\"SOURCE_UNLOAD\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TARGET_LOAD\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"SOURCE_CAPTURE\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TARGET_APPLY\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"},{\"Id\":\"TASK_MANAGER\",\"Severity\":\"LOGGER_SEVERITY_DEFAULT\"}],\"CloudWatchLogGroup\":null,\"CloudWatchLogStream\":null},\"ControlTablesSettings\":{\"historyTimeslotInMinutes\":5,\"ControlSchema\":\"\",\"HistoryTimeslotInMinutes\":5,\"HistoryTableEnabled\":false,\"SuspendedTablesTableEnabled\":false,\"StatusTableEnabled\":false},\"StreamBufferSettings\":{\"StreamBufferCount\":3,\"StreamBufferSizeInMB\":8,\"CtrlStreamBufferSizeInMB\":5},\"ChangeProcessingDdlHandlingPolicy\":{\"HandleSourceTableDropped\":true,\"HandleSourceTableTruncated\":true,\"HandleSourceTableAltered\":true},\"ErrorBehavior\":{\"DataErrorPolicy\":\"LOG_ERROR\",\"DataTruncationErrorPolicy\":\"LOG_ERROR\",\"DataErrorEscalationPolicy\":\"SUSPEND_TABLE\",\"DataErrorEscalationCount\":0,\"TableErrorPolicy\":\"SUSPEND_TABLE\",\"TableErrorEscalationPolicy\":\"STOP_TASK\",\"TableErrorEscalationCount\":0,\"RecoverableErrorCount\":-1,\"RecoverableErrorInterval\":5,\"RecoverableErrorThrottling\":true,\"RecoverableErrorThrottlingMax\":1800,\"ApplyErrorDeletePolicy\":\"IGNORE_RECORD\",\"ApplyErrorInsertPolicy\":\"LOG_ERROR\",\"ApplyErrorUpdatePolicy\":\"LOG_ERROR\",\"ApplyErrorEscalationPolicy\":\"LOG_ERROR\",\"ApplyErrorEscalationCount\":0,\"FullLoadIgnoreConflicts\":true},\"ChangeProcessingTuning\":{\"BatchApplyPreserveTransaction\":true,\"BatchApplyTimeoutMin\":1,\"BatchApplyTimeoutMax\":30,\"BatchApplyMemoryLimit\":500,\"BatchSplitSize\":0,\"MinTransactionSize\":1000,\"CommitTimeout\":1,\"MemoryLimitTotal\":1024,\"MemoryKeepTime\":60,\"StatementCacheSize\":50}}"
	source_endpoint_arn = "${aws_dms_endpoint.dms_endpoint_source.endpoint_arn}"
	table_mappings = "{\"rules\":[{\"rule-type\":\"selection\",\"rule-id\":\"1\",\"rule-name\":\"1\",\"object-locator\":{\"schema-name\":\"%%\",\"table-name\":\"%%\"},\"rule-action\":\"include\"}]}"
	tags {
		Name = "tf-test-dms-replication-task-%[1]s"
		Update = "updated"
		Add = "added"
	}
	target_endpoint_arn = "${aws_dms_endpoint.dms_endpoint_target.endpoint_arn}"
}
`, randId)
}

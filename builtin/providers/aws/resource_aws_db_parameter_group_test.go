package aws

import (
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBParameterGroup_limit(t *testing.T) {
	var v rds.DBParameterGroup

	groupName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: createAwsDbParameterGroupsExceedDefaultAwsLimit(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.large", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "name", groupName),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "family", "mysql5.6"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "description", "RDS default parameter group: Exceed default AWS parameter group limit of twenty"),

					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1680942586.name", "collation_server"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1680942586.value", "utf8_general_ci"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2450940716.name", "collation_connection"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2450940716.value", "utf8_general_ci"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.242489837.name", "join_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.242489837.value", "16777216"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2026669454.name", "key_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2026669454.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2705275319.name", "max_connections"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2705275319.value", "3200"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3512697936.name", "max_heap_table_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3512697936.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.780730667.name", "performance_schema"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.780730667.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2020346918.name", "performance_schema_users_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2020346918.value", "1048576"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1460834103.name", "query_cache_limit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1460834103.value", "2097152"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.484865451.name", "query_cache_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.484865451.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.255276438.name", "sort_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.255276438.value", "16777216"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2981725119.name", "table_open_cache"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2981725119.value", "4096"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2703661820.name", "tmp_table_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2703661820.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2386583229.name", "binlog_cache_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2386583229.value", "131072"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.4012389720.name", "innodb_flush_log_at_trx_commit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.4012389720.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2688783017.name", "innodb_open_files"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2688783017.value", "4000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.782983977.name", "innodb_read_io_threads"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.782983977.value", "64"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2809980413.name", "innodb_thread_concurrency"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2809980413.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3599115250.name", "innodb_write_io_threads"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3599115250.value", "64"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2557156277.name", "character_set_connection"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2557156277.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2475346812.name", "character_set_database"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2475346812.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1986528518.name", "character_set_filesystem"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1986528518.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1937131004.name", "event_scheduler"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1937131004.value", "on"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3437079877.name", "innodb_buffer_pool_dump_at_shutdown"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3437079877.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1092112861.name", "innodb_file_format"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1092112861.value", "barracuda"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.615571931.name", "innodb_io_capacity"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.615571931.value", "2000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1065962799.name", "innodb_io_capacity_max"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1065962799.value", "3000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1411161182.name", "innodb_lock_wait_timeout"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1411161182.value", "120"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3133315879.name", "innodb_max_dirty_pages_pct"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3133315879.value", "90"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.950177639.name", "log_bin_trust_function_creators"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.950177639.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.591700516.name", "log_warnings"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.591700516.value", "2"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1918306725.name", "log_output"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1918306725.value", "file"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.386204433.name", "max_allowed_packet"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.386204433.value", "1073741824"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1700901269.name", "max_connect_errors"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1700901269.value", "100"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2839701698.name", "query_cache_min_res_unit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2839701698.value", "512"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.427634017.name", "slow_query_log"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.427634017.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.881816039.name", "sync_binlog"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.881816039.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.748684209.name", "tx_isolation"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.748684209.value", "repeatable-read"),
				),
			},
			resource.TestStep{
				Config: updateAwsDbParameterGroupsExceedDefaultAwsLimit(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.large", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "name", groupName),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "family", "mysql5.6"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "description", "Updated RDS default parameter group: Exceed default AWS parameter group limit of twenty"),

					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1680942586.name", "collation_server"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1680942586.value", "utf8_general_ci"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2450940716.name", "collation_connection"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2450940716.value", "utf8_general_ci"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.242489837.name", "join_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.242489837.value", "16777216"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2026669454.name", "key_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2026669454.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2705275319.name", "max_connections"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2705275319.value", "3200"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3512697936.name", "max_heap_table_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3512697936.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.780730667.name", "performance_schema"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.780730667.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2020346918.name", "performance_schema_users_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2020346918.value", "1048576"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1460834103.name", "query_cache_limit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1460834103.value", "2097152"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.484865451.name", "query_cache_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.484865451.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.255276438.name", "sort_buffer_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.255276438.value", "16777216"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2981725119.name", "table_open_cache"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2981725119.value", "4096"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2703661820.name", "tmp_table_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2703661820.value", "67108864"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2386583229.name", "binlog_cache_size"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2386583229.value", "131072"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.4012389720.name", "innodb_flush_log_at_trx_commit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.4012389720.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2688783017.name", "innodb_open_files"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2688783017.value", "4000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.782983977.name", "innodb_read_io_threads"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.782983977.value", "64"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2809980413.name", "innodb_thread_concurrency"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2809980413.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3599115250.name", "innodb_write_io_threads"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3599115250.value", "64"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2557156277.name", "character_set_connection"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2557156277.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2475346812.name", "character_set_database"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2475346812.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1986528518.name", "character_set_filesystem"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1986528518.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1937131004.name", "event_scheduler"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1937131004.value", "on"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3437079877.name", "innodb_buffer_pool_dump_at_shutdown"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3437079877.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1092112861.name", "innodb_file_format"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1092112861.value", "barracuda"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.615571931.name", "innodb_io_capacity"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.615571931.value", "2000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1065962799.name", "innodb_io_capacity_max"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1065962799.value", "3000"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1411161182.name", "innodb_lock_wait_timeout"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1411161182.value", "120"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3133315879.name", "innodb_max_dirty_pages_pct"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.3133315879.value", "90"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.950177639.name", "log_bin_trust_function_creators"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.950177639.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.591700516.name", "log_warnings"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.591700516.value", "2"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1918306725.name", "log_output"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1918306725.value", "file"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.386204433.name", "max_allowed_packet"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.386204433.value", "1073741824"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1700901269.name", "max_connect_errors"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.1700901269.value", "100"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2839701698.name", "query_cache_min_res_unit"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.2839701698.value", "512"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.427634017.name", "slow_query_log"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.427634017.value", "1"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.881816039.name", "sync_binlog"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.881816039.value", "0"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.748684209.name", "tx_isolation"),
					resource.TestCheckResourceAttr("aws_db_parameter_group.large", "parameter.748684209.value", "repeatable-read"),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroup_basic(t *testing.T) {
	var v rds.DBParameterGroup

	groupName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupConfig(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "tags.%", "1"),
				),
			},
			resource.TestStep{
				Config: testAccAWSDBParameterGroupAddParametersConfig(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Test parameter group for terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1706463059.name", "collation_connection"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1706463059.value", "utf8_unicode_ci"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.name", "character_set_results"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.1708034931.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2475805061.name", "collation_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2475805061.value", "utf8_unicode_ci"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "tags.%", "2"),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroup_namePrefix(t *testing.T) {
	var v rds.DBParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBParameterGroupConfig_namePrefix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.test", &v),
					resource.TestMatchResourceAttr(
						"aws_db_parameter_group.test", "name", regexp.MustCompile("^tf-test-")),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroup_generatedName(t *testing.T) {
	var v rds.DBParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDBParameterGroupConfig_generatedName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.test", &v),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroup_withApplyMethod(t *testing.T) {
	var v rds.DBParameterGroup

	groupName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupConfigWithApplyMethod(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.name", "character_set_server"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2421266705.apply_method", "immediate"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.name", "character_set_client"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.value", "utf8"),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "parameter.2478663599.apply_method", "pending-reboot"),
				),
			},
		},
	})
}

func TestAccAWSDBParameterGroup_Only(t *testing.T) {
	var v rds.DBParameterGroup

	groupName := fmt.Sprintf("parameter-group-test-terraform-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDBParameterGroupOnlyConfig(groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSDBParameterGroupExists("aws_db_parameter_group.bar", &v),
					testAccCheckAWSDBParameterGroupAttributes(&v, groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "name", groupName),
					resource.TestCheckResourceAttr(
						"aws_db_parameter_group.bar", "family", "mysql5.6"),
				),
			},
		},
	})
}

func TestResourceAWSDBParameterGroupName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting123",
			ErrCount: 1,
		},
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "1testing123",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbParamGroupName(tc.Value, "aws_db_parameter_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Parameter Group Name to trigger a validation error")
		}
	}
}

func testAccCheckAWSDBParameterGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).rdsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_db_parameter_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeDBParameterGroups(
			&rds.DescribeDBParameterGroupsInput{
				DBParameterGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.DBParameterGroups) != 0 &&
				*resp.DBParameterGroups[0].DBParameterGroupName == rs.Primary.ID {
				return fmt.Errorf("DB Parameter Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "DBParameterGroupNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSDBParameterGroupAttributes(v *rds.DBParameterGroup, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *v.DBParameterGroupName != name {
			return fmt.Errorf("Bad Parameter Group name, expected (%s), got (%s)", name, *v.DBParameterGroupName)
		}

		if *v.DBParameterGroupFamily != "mysql5.6" {
			return fmt.Errorf("bad family: %#v", v.DBParameterGroupFamily)
		}

		return nil
	}
}

func testAccCheckAWSDBParameterGroupExists(n string, v *rds.DBParameterGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DB Parameter Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		opts := rds.DescribeDBParameterGroupsInput{
			DBParameterGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeDBParameterGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.DBParameterGroups) != 1 ||
			*resp.DBParameterGroups[0].DBParameterGroupName != rs.Primary.ID {
			return fmt.Errorf("DB Parameter Group not found")
		}

		*v = *resp.DBParameterGroups[0]

		return nil
	}
}

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func testAccAWSDBParameterGroupConfig(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "bar" {
	name = "%s"
	family = "mysql5.6"
	parameter {
	  name = "character_set_server"
	  value = "utf8"
	}
	parameter {
	  name = "character_set_client"
	  value = "utf8"
	}
	parameter{
	  name = "character_set_results"
	  value = "utf8"
	}
	tags {
		foo = "bar"
	}
}`, n)
}

func testAccAWSDBParameterGroupConfigWithApplyMethod(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "bar" {
	name = "%s"
	family = "mysql5.6"
	parameter {
	  name = "character_set_server"
	  value = "utf8"
	}
	parameter {
	  name = "character_set_client"
	  value = "utf8"
	  apply_method = "pending-reboot"
	}
	tags {
		foo = "bar"
	}
}`, n)
}

func testAccAWSDBParameterGroupAddParametersConfig(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "bar" {
	name = "%s"
	family = "mysql5.6"
	description = "Test parameter group for terraform"
	parameter {
	  name = "character_set_server"
	  value = "utf8"
	}
	parameter {
	  name = "character_set_client"
	  value = "utf8"
	}
	parameter{
	  name = "character_set_results"
	  value = "utf8"
	}
	parameter {
	  name = "collation_server"
	  value = "utf8_unicode_ci"
	}
	parameter {
	  name = "collation_connection"
	  value = "utf8_unicode_ci"
	}
	tags {
		foo = "bar"
		baz = "foo"
	}
}`, n)
}

func testAccAWSDBParameterGroupOnlyConfig(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "bar" {
	name = "%s"
	family = "mysql5.6"
	description = "Test parameter group for terraform"
}`, n)
}

func createAwsDbParameterGroupsExceedDefaultAwsLimit(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "large" {
	name = "%s"
	family = "mysql5.6"
	description = "RDS default parameter group: Exceed default AWS parameter group limit of twenty"

    parameter { name = "binlog_cache_size"                   value = 131072                                            }
    parameter { name = "character_set_client"                value = "utf8"                                            }
    parameter { name = "character_set_connection"            value = "utf8"                                            }
    parameter { name = "character_set_database"              value = "utf8"                                            }
    parameter { name = "character_set_filesystem"            value = "utf8"                                            }
    parameter { name = "character_set_results"               value = "utf8"                                            }
    parameter { name = "character_set_server"                value = "utf8"                                            }
    parameter { name = "collation_connection"                value = "utf8_general_ci"                                 }
    parameter { name = "collation_server"                    value = "utf8_general_ci"                                 }
    parameter { name = "event_scheduler"                     value = "ON"                                              }
    parameter { name = "innodb_buffer_pool_dump_at_shutdown" value = 1                                                 }
    parameter { name = "innodb_file_format"                  value = "Barracuda"                                       }
    parameter { name = "innodb_flush_log_at_trx_commit"      value = 0                                                 }
    parameter { name = "innodb_io_capacity"                  value = 2000                                              }
    parameter { name = "innodb_io_capacity_max"              value = 3000                                              }
    parameter { name = "innodb_lock_wait_timeout"            value = 120                                               }
    parameter { name = "innodb_max_dirty_pages_pct"          value = 90                                                }
    parameter { name = "innodb_open_files"                   value = 4000              apply_method = "pending-reboot" }
    parameter { name = "innodb_read_io_threads"              value = 64                apply_method = "pending-reboot" }
    parameter { name = "innodb_thread_concurrency"           value = 0                                                 }
    parameter { name = "innodb_write_io_threads"             value = 64                apply_method = "pending-reboot" }
    parameter { name = "join_buffer_size"                    value = 16777216                                          }
    parameter { name = "key_buffer_size"                     value = 67108864                                          }
    parameter { name = "log_bin_trust_function_creators"     value = 1                                                 }
    parameter { name = "log_warnings"                        value = 2                                                 }
    parameter { name = "log_output"                          value = "FILE"                                            }
    parameter { name = "max_allowed_packet"                  value = 1073741824                                        }
    parameter { name = "max_connect_errors"                  value = 100                                               }
    parameter { name = "max_connections"                     value = 3200                                              }
    parameter { name = "max_heap_table_size"                 value = 67108864                                          }
    parameter { name = "performance_schema"                  value = 1                 apply_method = "pending-reboot" }
    parameter { name = "performance_schema_users_size"       value = 1048576           apply_method = "pending-reboot" }
    parameter { name = "query_cache_limit"                   value = 2097152                                           }
    parameter { name = "query_cache_min_res_unit"            value = 512                                               }
    parameter { name = "query_cache_size"                    value = 67108864                                          }
    parameter { name = "slow_query_log"                      value = 1                                                 }
    parameter { name = "sort_buffer_size"                    value = 16777216                                          }
    parameter { name = "sync_binlog"                         value = 0                                                 }
    parameter { name = "table_open_cache"                    value = 4096                                              }
    parameter { name = "tmp_table_size"                      value = 67108864                                          }
    parameter { name = "tx_isolation"                        value = "REPEATABLE-READ"                                 }
}`, n)
}

func updateAwsDbParameterGroupsExceedDefaultAwsLimit(n string) string {
	return fmt.Sprintf(`
resource "aws_db_parameter_group" "large" {
	name = "%s"
	family = "mysql5.6"
	description = "Updated RDS default parameter group: Exceed default AWS parameter group limit of twenty"
    parameter { name = "binlog_cache_size"                   value = 131072                                            }
    parameter { name = "character_set_client"                value = "utf8"                                            }
    parameter { name = "character_set_connection"            value = "utf8"                                            }
    parameter { name = "character_set_database"              value = "utf8"                                            }
    parameter { name = "character_set_filesystem"            value = "utf8"                                            }
    parameter { name = "character_set_results"               value = "utf8"                                            }
    parameter { name = "character_set_server"                value = "utf8"                                            }
    parameter { name = "collation_connection"                value = "utf8_general_ci"                                 }
    parameter { name = "collation_server"                    value = "utf8_general_ci"                                 }
    parameter { name = "event_scheduler"                     value = "ON"                                              }
    parameter { name = "innodb_buffer_pool_dump_at_shutdown" value = 1                                                 }
    parameter { name = "innodb_file_format"                  value = "Barracuda"                                       }
    parameter { name = "innodb_flush_log_at_trx_commit"      value = 0                                                 }
    parameter { name = "innodb_io_capacity"                  value = 2000                                              }
    parameter { name = "innodb_io_capacity_max"              value = 3000                                              }
    parameter { name = "innodb_lock_wait_timeout"            value = 120                                               }
    parameter { name = "innodb_max_dirty_pages_pct"          value = 90                                                }
    parameter { name = "innodb_open_files"                   value = 4000              apply_method = "pending-reboot" }
    parameter { name = "innodb_read_io_threads"              value = 64                apply_method = "pending-reboot" }
    parameter { name = "innodb_thread_concurrency"           value = 0                                                 }
    parameter { name = "innodb_write_io_threads"             value = 64                apply_method = "pending-reboot" }
    parameter { name = "join_buffer_size"                    value = 16777216                                          }
    parameter { name = "key_buffer_size"                     value = 67108864                                          }
    parameter { name = "log_bin_trust_function_creators"     value = 1                                                 }
    parameter { name = "log_warnings"                        value = 2                                                 }
    parameter { name = "log_output"                          value = "FILE"                                            }
    parameter { name = "max_allowed_packet"                  value = 1073741824                                        }
    parameter { name = "max_connect_errors"                  value = 100                                               }
    parameter { name = "max_connections"                     value = 3200                                              }
    parameter { name = "max_heap_table_size"                 value = 67108864                                          }
    parameter { name = "performance_schema"                  value = 1                 apply_method = "pending-reboot" }
    parameter { name = "performance_schema_users_size"       value = 1048576           apply_method = "pending-reboot" }
    parameter { name = "query_cache_limit"                   value = 2097152                                           }
    parameter { name = "query_cache_min_res_unit"            value = 512                                               }
    parameter { name = "query_cache_size"                    value = 67108864                                          }
    parameter { name = "slow_query_log"                      value = 1                                                 }
    parameter { name = "sort_buffer_size"                    value = 16777216                                          }
    parameter { name = "sync_binlog"                         value = 0                                                 }
    parameter { name = "table_open_cache"                    value = 4096                                              }
    parameter { name = "tmp_table_size"                      value = 67108864                                          }
    parameter { name = "tx_isolation"                        value = "REPEATABLE-READ"                                 }
}`, n)
}

const testAccDBParameterGroupConfig_namePrefix = `
resource "aws_db_parameter_group" "test" {
	name_prefix = "tf-test-"
	family = "mysql5.6"

	parameter {
		name = "sync_binlog"
		value = 0
	}
}
`

const testAccDBParameterGroupConfig_generatedName = `
resource "aws_db_parameter_group" "test" {
	family = "mysql5.6"

	parameter {
		name = "sync_binlog"
		value = 0
	}
}
`

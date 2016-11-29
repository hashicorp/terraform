package logentries

import (
	"fmt"
	lexp "github.com/hashicorp/terraform/builtin/providers/logentries/expect"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/logentries/le_goclient"
	"testing"
)

type LogResource struct {
	Name            string `tfresource:"name"`
	RetentionPeriod string `tfresource:"retention_period"`
	Source          string `tfresource:"source"`
	Token           string `tfresource:"token"`
	Type            string `tfresource:"type"`
}

func TestAccLogentriesLog_Token(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
		}
	`, logName, logName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":   lexp.Equals(logName),
						"source": lexp.Equals("token"),
						"token":  lexp.RegexMatches("[0-9a-zA-Z]{8}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{4}-[0-9a-zA-Z]{12}"),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLog_SourceApi(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
			source = "api"
		}
	`, logName, logName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":   lexp.Equals(logName),
						"source": lexp.Equals("api"),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLog_SourceAgent(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	fileName := "/opt/foo"
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
			source = "agent"
			filename = "%s"
		}
	`, logName, logName, fileName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":     lexp.Equals(logName),
						"source":   lexp.Equals("agent"),
						"filename": lexp.Equals(fileName),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLog_RetentionPeriod1M(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
			retention_period = "1M"
		}
	`, logName, logName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":             lexp.Equals(logName),
						"retention_period": lexp.Equals("1M"),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLog_RetentionPeriodAccountDefault(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
		}
	`, logName, logName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":             lexp.Equals(logName),
						"retention_period": lexp.Equals("ACCOUNT_DEFAULT"),
					},
				),
			},
		},
	})
}

func TestAccLogentriesLog_RetentionPeriodAccountUnlimited(t *testing.T) {
	var logResource LogResource

	logName := fmt.Sprintf("terraform-test-%s", acctest.RandString(8))
	testAccLogentriesLogConfig := fmt.Sprintf(`
		resource "logentries_logset" "test_logset" {
			name = "%s"
		}
		resource "logentries_log" "test_log" {
			logset_id = "${logentries_logset.test_logset.id}"
			name = "%s"
			retention_period = "UNLIMITED"
		}
	`, logName, logName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			testAccCheckLogentriesLogDestroy(s)
			testAccCheckLogentriesLogSetDestroy(s)
			return nil
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogentriesLogConfig,
				Check: lexp.TestCheckResourceExpectation(
					"logentries_log.test_log",
					&logResource,
					testAccCheckLogentriesLogExists,
					map[string]lexp.TestExpectValue{
						"name":             lexp.Equals(logName),
						"retention_period": lexp.Equals("UNLIMITED"),
					},
				),
			},
		},
	})
}

func testAccCheckLogentriesLogDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*logentries.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "logentries_logset" {
			continue
		}

		resp, err := client.Log.Read(logentries.LogReadRequest{Key: rs.Primary.ID})

		if err == nil {
			return fmt.Errorf("Log still exists: %#v", resp)
		}
	}

	return nil
}

func testAccCheckLogentriesLogExists(n string, fact interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No LogSet Key is set")
		}

		client := testAccProvider.Meta().(*logentries.Client)

		resp, err := client.Log.Read(logentries.LogReadRequest{Key: rs.Primary.ID})

		if err != nil {
			return err
		}

		res := fact.(*LogResource)
		res.Name = resp.Name
		res.RetentionPeriod, _ = enumForRetentionPeriod(resp.Retention)
		res.Source = resp.Source
		res.Token = resp.Token
		res.Type = resp.Type

		return nil
	}
}

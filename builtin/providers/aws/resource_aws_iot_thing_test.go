package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIoTThing_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTThingDestroy_basic,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTThing_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIotThingExists_basic("aws_iot_thing.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTThingDestroy_basic(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_thing" {
			continue
		}

		out, err := conn.ListThings(&iot.ListThingsInput{})

		if err != nil {
			return err
		}

		for _, t := range out.Things {
			if *t.ThingName == rs.Primary.ID {
				return fmt.Errorf("IoT thing still exists:\n%s", t)
			}
		}

	}

	return nil
}

func testAccCheckAWSIotThingExists_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

func TestAccAWSIoTThing_attributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTThingDestroy_attributes,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTThing_attributes,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIotThingExists_attributes("aws_iot_thing.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTThingDestroy_attributes(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_thing" {
			continue
		}

		out, err := conn.ListThings(&iot.ListThingsInput{})

		if err != nil {
			return err
		}

		for _, t := range out.Things {
			if *t.ThingName == rs.Primary.ID {
				return fmt.Errorf("IoT thing still exists:\n%s", t)
			}
		}

	}

	return nil
}

func testAccCheckAWSIotThingExists_attributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

func TestAccAWSIoTThing_principal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTThingDestroy_principal,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTThing_principal,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIotThingExists_principal("aws_iot_thing.device3"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTThingDestroy_principal(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).iotconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_iot_thing" {
			continue
		}

		out, err := conn.ListThings(&iot.ListThingsInput{})

		if err != nil {
			return err
		}

		for _, t := range out.Things {
			if *t.ThingName == rs.Primary.ID {
				return fmt.Errorf("IoT thing still exists:\n%s", t)
			}
		}

	}

	return nil
}

func testAccCheckAWSIotThingExists_principal(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSIoTThing_basic = `
resource "aws_iot_thing" "foo" {
	name = "foo-thing"
}
`
var testAccAWSIoTThing_attributes = `
resource "aws_iot_thing" "foo" {
	name = "foo-thing"

  attributes {
    key1 = "val1"
    key2 = "val2"
 }
}
`

var testAccAWSIoTThing_principal = `
resource "aws_iot_thing" "device3" {
  name = "MyDevice3"
  principals = ["${aws_iot_certificate.cert.arn}"]

  attributes {
    Manufacturer = "Amazon"
    Type = "IoT Device A"
    SerialNumber = "10293847562912"
  }
}

resource "aws_iot_certificate" "cert" {
  csr = "${file("test-fixtures/csr.pem")}"
  active = true
}
`

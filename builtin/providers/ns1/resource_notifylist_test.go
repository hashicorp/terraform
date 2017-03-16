package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

func TestAccNotifyList_basic(t *testing.T) {
	var nl monitor.NotifyList
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNotifyListDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNotifyListBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotifyListExists("ns1_notifylist.test", &nl),
					testAccCheckNotifyListName(&nl, "terraform test"),
				),
			},
		},
	})
}

func TestAccNotifyList_updated(t *testing.T) {
	var nl monitor.NotifyList
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNotifyListDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNotifyListBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotifyListExists("ns1_notifylist.test", &nl),
					testAccCheckNotifyListName(&nl, "terraform test"),
				),
			},
			resource.TestStep{
				Config: testAccNotifyListUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotifyListExists("ns1_notifylist.test", &nl),
					testAccCheckNotifyListName(&nl, "terraform test"),
				),
			},
		},
	})
}

func testAccCheckNotifyListState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["ns1_notifylist.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "ns1_notifylist.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
}

func testAccCheckNotifyListExists(n string, nl *monitor.NotifyList) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Resource not found: %v", n)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("ID is not set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundNl, _, err := client.Notifications.Get(id)

		if err != nil {
			return err
		}

		if foundNl.ID != id {
			return fmt.Errorf("Notify List not found want: %#v, got %#v", id, foundNl)
		}

		*nl = *foundNl

		return nil
	}
}

func testAccCheckNotifyListDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_notifylist" {
			continue
		}

		nl, _, err := client.Notifications.Get(rs.Primary.Attributes["id"])

		if err == nil {
			return fmt.Errorf("Notify List still exists %#v: %#v", err, nl)
		}
	}

	return nil
}

func testAccCheckNotifyListName(nl *monitor.NotifyList, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nl.Name != expected {
			return fmt.Errorf("Name: got: %#v want: %#v", nl.Name, expected)
		}
		return nil
	}
}

const testAccNotifyListBasic = `
resource "ns1_notifylist" "test" {
  name = "terraform test"
  notifications = {
    type = "webhook"
    config = {
      url = "http://localhost:9090"
    }
  }
}
`

const testAccNotifyListUpdated = `
resource "ns1_notifylist" "test" {
  name = "terraform test"
  notifications = {
    type = "webhook"
    config = {
      url = "http://localhost:9091"
    }
  }
}
`

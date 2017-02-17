package pass

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestDataSourcePassword(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: testDataSourcePassword_config,
				Check:  testDataSourcePassword_check,
			},
		},
	})
}

var testDataSourcePassword_config = `

resource "pass_password" "test" {
    path = "secret/foo"
    data = "{\"zip\": \"zap\"}"
}

data "pass_password" "test" {
    path = "${pass_password.test.path}"
}

`

func testDataSourcePassword_check(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["data.pass_password.test"]
	if resourceState == nil {
		return fmt.Errorf("resource not found in state %v", s.Modules[0].Resources)
	}

	iState := resourceState.Primary
	if iState == nil {
		return fmt.Errorf("resource has no primary instance")
	}

	wantJson := "{\"zip\": \"zap\"}\n"
	if got, want := iState.Attributes["data_raw"], wantJson; got != want {
		return fmt.Errorf("data contains %s; want %s", got, want)
	}

	if got, want := iState.Attributes["data.zip"], "zap"; got != want {
		return fmt.Errorf("data[\"zip\"] contains %s; want %s", got, want)
	}

	return nil
}

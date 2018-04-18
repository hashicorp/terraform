package cidr

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestCidrSubnet(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: subnets,
				Check:  r.ComposeTestCheckFunc(efficientSubnetCheck),
			},
		},
	},
	)
}
func TestCidrSubnetWaste(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: wasteOfSpace,
				Check:  r.ComposeTestCheckFunc(wastedSubnetCheck),
			},
		},
	},
	)
}

func wastedSubnetCheck(s *terraform.State) error {
	answers := [][]string{
		[]string{"private_subnet1", "192.168.0.0/28"},
		[]string{"private_subnet2", "192.168.0.16/28"},
		[]string{"private_subnet3", "192.168.0.32/28"},
		[]string{"private_max", "192.168.0.32/28"},
		[]string{"public_subnet1", "192.168.0.128/25"},
		[]string{"public_subnet2", "192.168.1.0/25"},
		[]string{"public_max", "192.168.1.0/25"},
		[]string{"elb_subnet1", "192.168.4.0/22"},
		[]string{"elb_max", "192.168.4.0/22"},
	}
	return checkAnswers(answers, s.RootModule().Outputs)
}

func efficientSubnetCheck(s *terraform.State) error {
	answers := [][]string{
		[]string{"private_subnet1", "192.168.0.0/24"},
		[]string{"private_subnet2", "192.168.1.0/24"},
		[]string{"private_subnet3", "192.168.2.0/24"},
		[]string{"private_max", "192.168.2.0/24"},
		[]string{"public_subnet1", "192.168.3.0/25"},
		[]string{"public_subnet2", "192.168.3.128/25"},
		[]string{"public_subnet3", "192.168.4.0/25"},
		[]string{"public_max", "192.168.4.0/25"},
		[]string{"elb_subnet1", "192.168.4.128/28"},
		[]string{"elb_subnet2", "192.168.4.144/28"},
		[]string{"elb_subnet3", "192.168.4.160/28"},
		[]string{"elb_max", "192.168.4.160/28"},
	}
	return checkAnswers(answers, s.RootModule().Outputs)
}

func checkAnswers(expected [][]string, actual map[string]*terraform.OutputState) error {
	for _, ans := range expected {
		got := actual[ans[0]]
		if ans[1] != got.Value {
			fmt.Printf("Outputs %v\n", actual)
			return fmt.Errorf("Output expected %s, got %s\n", ans[1], got.Value)
		}
	}
	return nil
}

const subnets = `
data "cidr_subnet" "private" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 24 
	subnet_count = 3
}

data "cidr_subnet" "public" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 25
	subnet_count = 3
	start_after = "${data.cidr_subnet.private.max_subnet}"
}

data "cidr_subnet" "elb" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 28
	subnet_count = 3
	start_after = "${data.cidr_subnet.public.max_subnet}"
}

output "public_subnet1"  { value = "${data.cidr_subnet.public.subnet_cidrs[0]}" }
output "public_subnet2"  { value = "${data.cidr_subnet.public.subnet_cidrs[1]}" }
output "public_subnet3"  { value = "${data.cidr_subnet.public.subnet_cidrs[2]}" }
output "public_max"      { value = "${data.cidr_subnet.public.max_subnet}" }
output "private_subnet1" { value = "${data.cidr_subnet.private.subnet_cidrs[0]}" }
output "private_subnet2" { value = "${data.cidr_subnet.private.subnet_cidrs[1]}" }
output "private_subnet3" { value = "${data.cidr_subnet.private.subnet_cidrs[2]}" }
output "private_max"     { value = "${data.cidr_subnet.private.max_subnet}" }
output "elb_subnet1"     { value = "${data.cidr_subnet.elb.subnet_cidrs[0]}" }
output "elb_subnet2"     { value = "${data.cidr_subnet.elb.subnet_cidrs[1]}" }
output "elb_subnet3"     { value = "${data.cidr_subnet.elb.subnet_cidrs[2]}" }
output "elb_max"         { value = "${data.cidr_subnet.elb.max_subnet}" }
`

const wasteOfSpace = `
data "cidr_subnet" "private" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 28 
	subnet_count = 3
}

data "cidr_subnet" "public" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 25
	subnet_count = 2
	start_after = "${data.cidr_subnet.private.max_subnet}"
}

data "cidr_subnet" "elb" {
	cidr_block = "192.168.0.0/21"
	subnet_mask = 22
	start_after = "${data.cidr_subnet.public.max_subnet}"
}

output "public_subnet1"  { value = "${data.cidr_subnet.public.subnet_cidrs[0]}" }
output "public_subnet2"  { value = "${data.cidr_subnet.public.subnet_cidrs[1]}" }
output "public_max"      { value = "${data.cidr_subnet.public.max_subnet}" }
output "private_subnet1" { value = "${data.cidr_subnet.private.subnet_cidrs[0]}" }
output "private_subnet2" { value = "${data.cidr_subnet.private.subnet_cidrs[1]}" }
output "private_subnet3" { value = "${data.cidr_subnet.private.subnet_cidrs[2]}" }
output "private_max"     { value = "${data.cidr_subnet.private.max_subnet}" }
output "elb_subnet1"     { value = "${data.cidr_subnet.elb.subnet_cidrs[0]}" }
output "elb_max"         { value = "${data.cidr_subnet.elb.max_subnet}" }
`

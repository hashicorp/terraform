package stun

import (
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pixelbender/go-stun/stun"
)

var testProviders = map[string]terraform.ResourceProvider{
	"stun": Provider(),
}

func TestAccStun(t *testing.T) {
	var hosts = []string{
		"stun.ekiga.net",
		"stun.ekiga.net:3478",
		"global.stun.twilio.com",
	}

	for _, host := range hosts {
		resource.Test(t, resource.TestCase{
			Providers: testProviders,
			Steps: []resource.TestStep{
				{
					Config: testStunConfig(host),
					Check: resource.ComposeTestCheckFunc(
						notEmpty("ip_address"),
					),
				},
			},
		})
	}
}

func TestStun_IPv4(t *testing.T) {
	testStun(t, "4")
}

func TestStun_IPv6(t *testing.T) {
	testStun(t, "6")
}

func testStun(t *testing.T, ipv string) {
	// start a UDP listener
	l, err := net.ListenPacket("udp"+ipv, "")
	if err != nil {
		t.Fatal("listen error", err)
	}
	defer l.Close()

	// hand off the UDP listener to a STUN server in a goroutine
	go stun.NewServer(nil).ServePacket(l)

	// setup our test
	host := l.LocalAddr().String()
	var loopback net.IP
	switch ipv {
	case "4":
		loopback = net.ParseIP("127.0.0.1")
	case "6":
		loopback = net.IPv6loopback
	}
	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: testStunConfig(host),
				Check: resource.ComposeTestCheckFunc(
					assert("ip_address", loopback.String()),
					assert("ip_family", "ipv"+ipv),
				),
			},
		},
	})
}

func assert(attribute, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r := s.RootModule().Resources["data.stun.local"]
		got := r.Primary.Attributes[attribute]
		if got != want {
			return fmt.Errorf("%s expected to be %q but got %q", attribute, want, got)
		}
		return nil
	}
}

func testStunConfig(host string) string {
	return fmt.Sprintf(`
	data "stun" "local" {
		server = %q
	}`, host)
}

func notEmpty(attribute string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r := s.RootModule().Resources["data.stun.local"]
		got := r.Primary.Attributes[attribute]
		if got == "" {
			return fmt.Errorf("%s expected not to be empty, but it was", attribute)
		}
		return nil
	}
}

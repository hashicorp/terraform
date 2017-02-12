package stun

import (
	"fmt"
	"net"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pixelbender/go-stun/stun"
)

var testProviders = map[string]terraform.ResourceProvider{
	"stun": Provider(),
}

func TestStun(t *testing.T) {
	var hosts = []string{
		"stun.ekiga.net",
		"stun.ekiga.net:3478",
		"global.stun.twilio.com",
	}

	for _, host := range hosts {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				{
					Config: testStunConfig(host),
					Check: func(s *terraform.State) error {
						got := s.RootModule().Outputs["ip_address"]
						if got.Value == "" {
							return fmt.Errorf("host:\n%s\nip_address:\n%s", host, got)
						}
						got = s.RootModule().Outputs["ip_family"]
						if got.Value == "" {
							return fmt.Errorf("host:\n%s\nip_family:\n%s", host, got)
						}
						return nil
					},
				},
			},
		})
	}
}

func TestStunIPV(t *testing.T) {
	for _, ipv := range []string{"4", "6"} {

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
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				{
					Config: testStunConfig(host),
					Check: func(s *terraform.State) error {
						ip_address := s.RootModule().Outputs["ip_address"]
						if want := loopback.String(); ip_address.Value != want {
							return fmt.Errorf("host:\n%s\nip_address:\n%s\nwant:\n%s", host, ip_address.Value, want)
						}
						ip_family := s.RootModule().Outputs["ip_family"]
						want := "ipv" + ipv
						if ip_family.Value != want {
							return fmt.Errorf("host:\n%s\nip_family:\n%s\nwant:\n%s", host, ip_family, want)
						}
						return nil
					},
				},
			},
		})
	}
}

func testStunConfig(host string) string {
	return fmt.Sprintf(`
	data "stun" "local" {
		server = %q
	}
	output "ip_address" {
		value = "${data.stun.local.ip_address}"
	}
	output "ip_family" {
		value = "${data.stun.local.ip_family}"
	}`, host)
}

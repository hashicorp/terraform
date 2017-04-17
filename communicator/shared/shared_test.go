package shared

import (
	"testing"
)

func TestIpFormatting_Ipv4(t *testing.T) {
	formatted := IpFormat("127.0.0.1")
	if formatted != "127.0.0.1" {
		t.Fatal("expected", "127.0.0.1", "got", formatted)
	}
}

func TestIpFormatting_Hostname(t *testing.T) {
	formatted := IpFormat("example.com")
	if formatted != "example.com" {
		t.Fatal("expected", "example.com", "got", formatted)
	}
}

func TestIpFormatting_Ipv6(t *testing.T) {
	formatted := IpFormat("::1")
	if formatted != "[::1]" {
		t.Fatal("expected", "[::1]", "got", formatted)
	}
}

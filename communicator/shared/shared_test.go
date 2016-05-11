package shared

import (
	"testing"
)

func TestHostnameFormatting_Ipv4(t *testing.T) {
	h := HostFormatterImpl{}
	formatted := h.Format("127.0.0.1")
	if formatted != "127.0.0.1" {
		t.Fatal("expected", "127.0.0.1", "got", formatted)
	}
}

func TestHostnameFormatting_Hostname(t *testing.T) {
	h := HostFormatterImpl{}
	formatted := h.Format("example.com")
	if formatted != "example.com" {
		t.Fatal("expected", "example.com", "got", formatted)
	}
}

func TestHostnameFormatting_Ipv6(t *testing.T) {
	h := HostFormatterImpl{}
	formatted := h.Format("::1")
	if formatted != "[::1]" {
		t.Fatal("expected", "[::1]", "got", formatted)
	}
}

package winrmcp

import "testing"

func Test_parsing_an_addr_to_a_winrm_endpoint(t *testing.T) {
	endpoint, err := parseEndpoint("1.2.3.4:1234", false, false, nil)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "1.2.3.4" {
		t.Error("Host should be 1.2.3.4")
	}
	if endpoint.Port != 1234 {
		t.Error("Port should be 1234")
	}
	if endpoint.Insecure {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS {
		t.Error("Endpoint should be HTTP not HTTPS")
	}
}

func Test_parsing_an_addr_without_a_port_to_a_winrm_endpoint(t *testing.T) {
	certBytes := []byte{1, 2, 3, 4, 5, 6}
	endpoint, err := parseEndpoint("1.2.3.4", true, true, certBytes)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "1.2.3.4" {
		t.Error("Host should be 1.2.3.4")
	}
	if endpoint.Port != 5985 {
		t.Error("Port should be 5985")
	}
	if endpoint.Insecure != true {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS != true {
		t.Error("Endpoint should be HTTPS")
	}

	if len(*endpoint.CACert) != len(certBytes) {
		t.Error("Length of CACert is wrong")
	}
	for i := 0; i < len(certBytes); i++ {
		if (*endpoint.CACert)[i] != certBytes[i] {
			t.Error("CACert is not set correctly")
		}
	}
}

func Test_parsing_an_empty_addr_to_a_winrm_endpoint(t *testing.T) {
	endpoint, err := parseEndpoint("", false, false, nil)

	if endpoint != nil {
		t.Error("Endpoint should be nil")
	}
	if err == nil {
		t.Error("Expected an error")
	}
}

func Test_parsing_an_addr_with_a_bad_port(t *testing.T) {
	endpoint, err := parseEndpoint("1.2.3.4:ABCD", false, false, nil)

	if endpoint != nil {
		t.Error("Endpoint should be nil")
	}
	if err == nil {
		t.Error("Expected an error")
	}
}

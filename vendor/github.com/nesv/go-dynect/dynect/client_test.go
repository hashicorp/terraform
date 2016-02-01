package dynect

import (
	"os"
	"testing"
	"strings"
)

var (
	DynCustomerName string
	DynUsername     string
	DynPassword     string
	testZone string
)

func init() {
	DynCustomerName = os.Getenv("DYNECT_CUSTOMER_NAME")
	DynUsername = os.Getenv("DYNECT_USER_NAME")
	DynPassword = os.Getenv("DYNECT_PASSWORD")
	testZone = os.Getenv("DYNECT_TEST_ZONE")
}

func TestSetup(t *testing.T) {
	if len(DynCustomerName) == 0 {
		t.Fatal("DYNECT_CUSTOMER_NAME not set")
	}

	if len(DynUsername) == 0 {
		t.Fatal("DYNECT_USER_NAME not set")
	}

	if len(DynPassword) == 0 {
		t.Fatal("DYNECT_PASSWORD not set")
	}

	if len(testZone) == 0 {
		t.Fatal("DYNECT_TEST_ZONE not specified")
	}
}

func TestLoginLogout(t *testing.T) {
	client := NewClient(DynCustomerName)
	client.Verbose(true)
	err := client.Login(DynUsername, DynPassword)
	if err != nil {
		t.Error(err)
	}

	err = client.Logout()
	if err != nil {
		t.Error(err)
	}
}

func TestZonesRequest(t *testing.T) {
	client := NewClient(DynCustomerName)
	client.Verbose(true)
	err := client.Login(DynUsername, DynPassword)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err = client.Logout()
		if err != nil {
			t.Error(err)
		}
	}()

	var resp ZonesResponse
	err = client.Do("GET", "Zone", nil, &resp)
	if err != nil {
		t.Error(err)
	}

	nresults := len(resp.Data)
	for i, zone := range resp.Data {
		parts := strings.Split(zone, "/")
		t.Logf("(%d/%d) %q", i+1, nresults, parts[len(parts)-2])
	}
}

func TestFetchingAllZoneRecords(t *testing.T) {
	client := NewClient(DynCustomerName)
	client.Verbose(true)
	err := client.Login(DynUsername, DynPassword)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err = client.Logout()
		if err != nil {
			t.Error(err)
		}
	}()

	var resp AllRecordsResponse
	err = client.Do("GET", "AllRecord/" + testZone, nil, &resp)
	if err != nil {
		t.Error(err)
	}
	for _, zr := range resp.Data {
		parts := strings.Split(zr, "/")
		uri := strings.Join(parts[2:], "/")
		t.Log(uri)

		var record RecordResponse
		err := client.Do("GET", uri, nil, &record)
		if err != nil {
			t.Fatal(err)
		}

		t.Log("OK")
	}
}

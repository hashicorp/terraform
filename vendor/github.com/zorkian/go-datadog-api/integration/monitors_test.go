package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/zorkian/go-datadog-api"
	"testing"
)

func init() {
	client = initTest()
}

func TestCreateAndDeleteMonitor(t *testing.T) {
	expected := getTestMonitor()
	// create the monitor and compare it
	actual := createTestMonitor(t)
	defer cleanUpMonitor(t, actual.Id)

	// Set ID of our original struct to zero we we can easily compare the results
	expected.Id = actual.Id
	assert.Equal(t, expected, actual)

	actual, err := client.GetMonitor(actual.Id)
	if err != nil {
		t.Fatalf("Retrieving a monitor failed when it shouldn't: (%s)", err)
	}
	assert.Equal(t, expected, actual)
}

func TestUpdateMonitor(t *testing.T) {

	monitor := createTestMonitor(t)
	defer cleanUpMonitor(t, monitor.Id)

	monitor.Name = "___New-Test-Monitor___"
	if err := client.UpdateMonitor(monitor); err != nil {
		t.Fatalf("Updating a monitor failed when it shouldn't: %s", err)
	}

	actual, err := client.GetMonitor(monitor.Id)
	if err != nil {
		t.Fatalf("Retrieving a monitor failed when it shouldn't: %s", err)
	}

	assert.Equal(t, monitor, actual)

}

func TestGetMonitor(t *testing.T) {
	monitors, err := client.GetMonitors()
	if err != nil {
		t.Fatalf("Retrieving monitors failed when it shouldn't: %s", err)
	}
	num := len(monitors)

	monitor := createTestMonitor(t)
	defer cleanUpMonitor(t, monitor.Id)

	monitors, err = client.GetMonitors()
	if err != nil {
		t.Fatalf("Retrieving monitors failed when it shouldn't: %s", err)
	}

	if num+1 != len(monitors) {
		t.Fatalf("Number of monitors didn't match expected: %d != %d", len(monitors), num+1)
	}
}

func TestMuteUnmuteMonitor(t *testing.T) {
	monitor := createTestMonitor(t)
	defer cleanUpMonitor(t, monitor.Id)

	// Mute
	err := client.MuteMonitor(monitor.Id)
	if err != nil {
		t.Fatalf("Failed to mute monitor")

	}

	monitor, err = client.GetMonitor(monitor.Id)
	if err != nil {
		t.Fatalf("Retrieving monitors failed when it shouldn't: %s", err)
	}

	// Mute without options will result in monitor.Options.Silenced
	// to have a key of "*" with value 0
	assert.Equal(t, 0, monitor.Options.Silenced["*"])

	// Unmute
	err = client.UnmuteMonitor(monitor.Id)
	if err != nil {
		t.Fatalf("Failed to unmute monitor")
	}

	// Update remote state
	monitor, err = client.GetMonitor(monitor.Id)
	if err != nil {
		t.Fatalf("Retrieving monitors failed when it shouldn't: %s", err)
	}

	// Assert this map is empty
	assert.Equal(t, 0, len(monitor.Options.Silenced))
}

/*
	Testing of global mute and unmuting has not been added for following reasons:
	* Disabling and enabling of global monitoring does an @all mention which is noisy
	* It exposes risk to users that run integration tests in their main account
	* There is no endpoint to verify success
*/

func getTestMonitor() *datadog.Monitor {

	o := datadog.Options{
		NotifyNoData:    true,
		NoDataTimeframe: 60,
		Silenced:        map[string]int{},
	}

	return &datadog.Monitor{
		Message: "Test message",
		Query:   "avg(last_15m):avg:system.disk.in_use{*} by {host,device} > 0.8",
		Name:    "Test monitor",
		Options: o,
		Type:    "metric alert",
	}
}

func createTestMonitor(t *testing.T) *datadog.Monitor {
	monitor := getTestMonitor()
	monitor, err := client.CreateMonitor(monitor)
	if err != nil {
		t.Fatalf("Creating a monitor failed when it shouldn't: %s", err)
	}

	return monitor
}

func cleanUpMonitor(t *testing.T, id int) {
	if err := client.DeleteMonitor(id); err != nil {
		t.Fatalf("Deleting a monitor failed when it shouldn't. Manual cleanup needed. (%s)", err)
	}

	deletedMonitor, err := client.GetMonitor(id)
	if deletedMonitor != nil {
		t.Fatal("Monitor hasn't been deleted when it should have been. Manual cleanup needed.")
	}

	if err == nil {
		t.Fatal("Fetching deleted monitor didn't lead to an error.")
	}
}

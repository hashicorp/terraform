package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/zorkian/go-datadog-api"
	"testing"
)

func init() {
	client = initTest()
}

func TestCreateAndDeleteDowntime(t *testing.T) {
	expected := getTestDowntime()
	// create the downtime and compare it
	actual := createTestDowntime(t)
	defer cleanUpDowntime(t, actual.Id)

	// Set ID of our original struct to zero we we can easily compare the results
	expected.Id = actual.Id
	assert.Equal(t, expected, actual)

	actual, err := client.GetDowntime(actual.Id)
	if err != nil {
		t.Fatalf("Retrieving a downtime failed when it shouldn't: (%s)", err)
	}
	assert.Equal(t, expected, actual)
}

func TestUpdateDowntime(t *testing.T) {

	downtime := createTestDowntime(t)

	downtime.Scope = []string{"env:downtime_test", "env:downtime_test2"}
	defer cleanUpDowntime(t, downtime.Id)

	if err := client.UpdateDowntime(downtime); err != nil {
		t.Fatalf("Updating a downtime failed when it shouldn't: %s", err)
	}

	actual, err := client.GetDowntime(downtime.Id)
	if err != nil {
		t.Fatalf("Retrieving a downtime failed when it shouldn't: %s", err)
	}

	assert.Equal(t, downtime, actual)

}

func TestGetDowntime(t *testing.T) {
	downtimes, err := client.GetDowntimes()
	if err != nil {
		t.Fatalf("Retrieving downtimes failed when it shouldn't: %s", err)
	}
	num := len(downtimes)

	downtime := createTestDowntime(t)
	defer cleanUpDowntime(t, downtime.Id)

	downtimes, err = client.GetDowntimes()
	if err != nil {
		t.Fatalf("Retrieving downtimes failed when it shouldn't: %s", err)
	}

	if num+1 != len(downtimes) {
		t.Fatalf("Number of downtimes didn't match expected: %d != %d", len(downtimes), num+1)
	}
}

func getTestDowntime() *datadog.Downtime {

	r := &datadog.Recurrence{
		Type:     "weeks",
		Period:   1,
		WeekDays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
	}

	return &datadog.Downtime{
		Message:    "Test downtime message",
		Scope:      []string{"env:downtime_test"},
		Start:      1577836800,
		End:        1577840400,
		Recurrence: r,
	}
}

func createTestDowntime(t *testing.T) *datadog.Downtime {
	downtime := getTestDowntime()
	downtime, err := client.CreateDowntime(downtime)
	if err != nil {
		t.Fatalf("Creating a downtime failed when it shouldn't: %s", err)
	}

	return downtime
}

func cleanUpDowntime(t *testing.T, id int) {
	if err := client.DeleteDowntime(id); err != nil {
		t.Fatalf("Deleting a downtime failed when it shouldn't. Manual cleanup needed. (%s)", err)
	}

	deletedDowntime, err := client.GetDowntime(id)
	if deletedDowntime != nil && deletedDowntime.Canceled == 0 {
		t.Fatal("Downtime hasn't been deleted when it should have been. Manual cleanup needed.")
	}

	if err == nil && deletedDowntime.Canceled == 0 {
		t.Fatal("Fetching deleted downtime didn't lead to an error and downtime Canceled not set.")
	}
}

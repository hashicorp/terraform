package integration

import (
	"github.com/zorkian/go-datadog-api"
	"testing"
)

func init() {
	client = initTest()
}

func TestCreateAndDeleteDashboard(t *testing.T) {
	expected := getTestDashboard()
	// create the dashboard and compare it
	actual, err := client.CreateDashboard(expected)
	if err != nil {
		t.Fatalf("Creating a dashboard failed when it shouldn't. (%s)", err)
	}

	defer cleanUpDashboard(t, actual.Id)

	assertDashboardEquals(t, actual, expected)

	// now try to fetch it freshly and compare it again
	actual, err = client.GetDashboard(actual.Id)
	if err != nil {
		t.Fatalf("Retrieving a dashboard failed when it shouldn't. (%s)", err)
	}
	assertDashboardEquals(t, actual, expected)

}

func TestUpdateDashboard(t *testing.T) {
	expected := getTestDashboard()
	board, err := client.CreateDashboard(expected)
	if err != nil {
		t.Fatalf("Creating a dashboard failed when it shouldn't. (%s)", err)
	}

	defer cleanUpDashboard(t, board.Id)
	board.Title = "___New-Test-Board___"

	if err := client.UpdateDashboard(board); err != nil {
		t.Fatalf("Updating a dashboard failed when it shouldn't: %s", err)
	}

	actual, err := client.GetDashboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a dashboard failed when it shouldn't: %s", err)
	}

	assertDashboardEquals(t, actual, board)
}

func TestGetDashboards(t *testing.T) {
	boards, err := client.GetDashboards()
	if err != nil {
		t.Fatalf("Retrieving dashboards failed when it shouldn't: %s", err)
	}

	num := len(boards)
	board := createTestDashboard(t)
	defer cleanUpDashboard(t, board.Id)

	boards, err = client.GetDashboards()
	if err != nil {
		t.Fatalf("Retrieving dashboards failed when it shouldn't: %s", err)
	}

	if num+1 != len(boards) {
		t.Fatalf("Number of dashboards didn't match expected: %d != %d", len(boards), num+1)
	}
}

func getTestDashboard() *datadog.Dashboard {
	return &datadog.Dashboard{
		Title:             "___Test-Board___",
		Description:       "Testboard description",
		TemplateVariables: []datadog.TemplateVariable{},
		Graphs:            createGraph(),
	}
}

func createTestDashboard(t *testing.T) *datadog.Dashboard {
	board := getTestDashboard()
	board, err := client.CreateDashboard(board)
	if err != nil {
		t.Fatalf("Creating a dashboard failed when it shouldn't: %s", err)
	}

	return board
}

func cleanUpDashboard(t *testing.T, id int) {
	if err := client.DeleteDashboard(id); err != nil {
		t.Fatalf("Deleting a dashboard failed when it shouldn't. Manual cleanup needed. (%s)", err)
	}

	deletedBoard, err := client.GetDashboard(id)
	if deletedBoard != nil {
		t.Fatal("Dashboard hasn't been deleted when it should have been. Manual cleanup needed.")
	}

	if err == nil {
		t.Fatal("Fetching deleted dashboard didn't lead to an error. Manual cleanup needed.")
	}
}

type TestGraphDefintionRequests struct {
	Query   string `json:"q"`
	Stacked bool   `json:"stacked"`
}

func createGraph() []datadog.Graph {
	graphDefinition := datadog.Graph{}.Definition
	graphDefinition.Viz = "timeseries"
	r := datadog.Graph{}.Definition.Requests
	graphDefinition.Requests = append(r, TestGraphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})
	graph := datadog.Graph{Title: "Mandatory graph", Definition: graphDefinition}
	graphs := []datadog.Graph{}
	graphs = append(graphs, graph)
	return graphs
}

func assertDashboardEquals(t *testing.T, actual, expected *datadog.Dashboard) {
	if actual.Title != expected.Title {
		t.Errorf("Dashboard title does not match: %s != %s", actual.Title, expected.Title)
	}
	if actual.Description != expected.Description {
		t.Errorf("Dashboard description does not match: %s != %s", actual.Description, expected.Description)
	}
	if len(actual.Graphs) != len(expected.Graphs) {
		t.Errorf("Number of Dashboard graphs does not match: %d != %d", len(actual.Graphs), len(expected.Graphs))
	}
	if len(actual.TemplateVariables) != len(expected.TemplateVariables) {
		t.Errorf("Number of Dashboard template variables does not match: %d != %d", len(actual.TemplateVariables), len(expected.TemplateVariables))
	}
}

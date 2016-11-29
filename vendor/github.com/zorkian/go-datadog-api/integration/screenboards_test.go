package integration

import (
	"github.com/zorkian/go-datadog-api"
	"testing"
)

func init() {
	client = initTest()
}

func TestCreateAndDeleteScreenboard(t *testing.T) {
	expected := getTestScreenboard()
	// create the screenboard and compare it
	actual, err := client.CreateScreenboard(expected)
	if err != nil {
		t.Fatalf("Creating a screenboard failed when it shouldn't. (%s)", err)
	}

	defer cleanUpScreenboard(t, actual.Id)

	assertScreenboardEquals(t, actual, expected)

	// now try to fetch it freshly and compare it again
	actual, err = client.GetScreenboard(actual.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed when it shouldn't. (%s)", err)
	}

	assertScreenboardEquals(t, actual, expected)

}

func TestShareAndRevokeScreenboard(t *testing.T) {
	expected := getTestScreenboard()
	// create the screenboard
	actual, err := client.CreateScreenboard(expected)
	if err != nil {
		t.Fatalf("Creating a screenboard failed when it shouldn't: %s", err)
	}

	defer cleanUpScreenboard(t, actual.Id)

	// share screenboard and verify it was shared
	var response datadog.ScreenShareResponse
	err = client.ShareScreenboard(actual.Id, &response)
	if err != nil {
		t.Fatalf("Failed to share screenboard: %s", err)
	}

	// revoke screenboard
	err = client.RevokeScreenboard(actual.Id)
	if err != nil {
		t.Fatalf("Failed to revoke sharing of screenboard: %s", err)
	}
}

func TestUpdateScreenboard(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	board.Title = "___New-Test-Board___"
	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed when it shouldn't: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed when it shouldn't: %s", err)
	}

	assertScreenboardEquals(t, actual, board)

}

func TestGetScreenboards(t *testing.T) {
	boards, err := client.GetScreenboards()
	if err != nil {
		t.Fatalf("Retrieving screenboards failed when it shouldn't: %s", err)
	}
	num := len(boards)

	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	boards, err = client.GetScreenboards()
	if err != nil {
		t.Fatalf("Retrieving screenboards failed when it shouldn't: %s", err)
	}

	if num+1 != len(boards) {
		t.Fatalf("Number of screenboards didn't match expected: %d != %d", len(boards), num+1)
	}
}

func getTestScreenboard() *datadog.Screenboard {
	return &datadog.Screenboard{
		Title:   "___Test-Board___",
		Height:  "600",
		Width:   "800",
		Widgets: []datadog.Widget{},
	}
}

func createTestScreenboard(t *testing.T) *datadog.Screenboard {
	board := getTestScreenboard()
	board, err := client.CreateScreenboard(board)
	if err != nil {
		t.Fatalf("Creating a screenboard failed when it shouldn't: %s", err)
	}

	return board
}

func cleanUpScreenboard(t *testing.T, id int) {
	if err := client.DeleteScreenboard(id); err != nil {
		t.Fatalf("Deleting a screenboard failed when it shouldn't. Manual cleanup needed. (%s)", err)
	}

	deletedBoard, err := client.GetScreenboard(id)
	if deletedBoard != nil {
		t.Fatal("Screenboard hasn't been deleted when it should have been. Manual cleanup needed.")
	}

	if err == nil {
		t.Fatal("Fetching deleted screenboard didn't lead to an error. Manual cleanup needed.")
	}
}

func assertScreenboardEquals(t *testing.T, actual, expected *datadog.Screenboard) {
	if actual.Title != expected.Title {
		t.Errorf("Screenboard title does not match: %s != %s", actual.Title, expected.Title)
	}
	if actual.Width != expected.Width {
		t.Errorf("Screenboard width does not match: %s != %s", actual.Width, expected.Width)
	}
	if actual.Height != expected.Height {
		t.Errorf("Screenboard width does not match: %s != %s", actual.Height, expected.Height)
	}
	if len(actual.Widgets) != len(expected.Widgets) {
		t.Errorf("Number of Screenboard widgets does not match: %d != %d", len(actual.Widgets), len(expected.Widgets))
	}
}

package tfe

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func testLogReader(t *testing.T, h http.HandlerFunc) (*httptest.Server, *LogReader) {
	ts := httptest.NewServer(h)

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		HTTPClient: ts.Client(),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	logURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	lr := &LogReader{
		client: client,
		ctx:    context.Background(),
		logURL: logURL,
	}

	return ts, lr
}

func TestLogReader_withMarkersSingle(t *testing.T) {
	t.Parallel()

	logReads := 0
	ts, lr := testLogReader(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReads++
		switch {
		case logReads == 2:
			w.Write([]byte("\x02Terraform run started - logs - Terraform run finished\x03"))
		}
	}))
	defer ts.Close()

	doneReads := 0
	lr.done = func() (bool, error) {
		doneReads++
		if logReads >= 2 {
			return true, nil
		}
		return false, nil
	}

	logs, err := ioutil.ReadAll(lr)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Terraform run started - logs - Terraform run finished"
	if string(logs) != expected {
		t.Fatalf("expected %s, got: %s", expected, string(logs))
	}
	if doneReads != 1 {
		t.Fatalf("expected 1 done reads, got %d reads", doneReads)
	}
	if logReads != 3 {
		t.Fatalf("expected 3 log reads, got %d reads", logReads)
	}
}

func TestLogReader_withMarkersDouble(t *testing.T) {
	t.Parallel()

	logReads := 0
	ts, lr := testLogReader(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReads++
		switch {
		case logReads == 2:
			w.Write([]byte("\x02Terraform run started"))
		case logReads == 3:
			w.Write([]byte(" - logs - Terraform run finished\x03"))
		}
	}))
	defer ts.Close()

	doneReads := 0
	lr.done = func() (bool, error) {
		doneReads++
		if logReads >= 3 {
			return true, nil
		}
		return false, nil
	}

	logs, err := ioutil.ReadAll(lr)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Terraform run started - logs - Terraform run finished"
	if string(logs) != expected {
		t.Fatalf("expected %s, got: %s", expected, string(logs))
	}
	if doneReads != 1 {
		t.Fatalf("expected 1 done reads, got %d reads", doneReads)
	}
	if logReads != 4 {
		t.Fatalf("expected 4 log reads, got %d reads", logReads)
	}
}

func TestLogReader_withMarkersMulti(t *testing.T) {
	t.Parallel()

	logReads := 0
	ts, lr := testLogReader(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReads++
		switch {
		case logReads == 2:
			w.Write([]byte("\x02"))
		case logReads == 3:
			w.Write([]byte("Terraform run started"))
		case logReads == 16:
			w.Write([]byte(" - logs - "))
		case logReads == 30:
			w.Write([]byte("Terraform run finished"))
		case logReads == 31:
			w.Write([]byte("\x03"))
		}
	}))
	defer ts.Close()

	doneReads := 0
	lr.done = func() (bool, error) {
		doneReads++
		if logReads >= 31 {
			return true, nil
		}
		return false, nil
	}

	logs, err := ioutil.ReadAll(lr)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Terraform run started - logs - Terraform run finished"
	if string(logs) != expected {
		t.Fatalf("expected %s, got: %s", expected, string(logs))
	}
	if doneReads != 3 {
		t.Fatalf("expected 3 done reads, got %d reads", doneReads)
	}
	if logReads != 31 {
		t.Fatalf("expected 31 log reads, got %d reads", logReads)
	}
}

func TestLogReader_withoutMarkers(t *testing.T) {
	t.Parallel()

	logReads := 0
	ts, lr := testLogReader(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReads++
		switch {
		case logReads == 2:
			w.Write([]byte("Terraform run started"))
		case logReads == 16:
			w.Write([]byte(" - logs - "))
		case logReads == 31:
			w.Write([]byte("Terraform run finished"))
		}
	}))
	defer ts.Close()

	doneReads := 0
	lr.done = func() (bool, error) {
		doneReads++
		if logReads >= 31 {
			return true, nil
		}
		return false, nil
	}

	logs, err := ioutil.ReadAll(lr)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Terraform run started - logs - Terraform run finished"
	if string(logs) != expected {
		t.Fatalf("expected %s, got: %s", expected, string(logs))
	}
	if doneReads != 25 {
		t.Fatalf("expected 14 done reads, got %d reads", doneReads)
	}
	if logReads != 32 {
		t.Fatalf("expected 32 log reads, got %d reads", logReads)
	}
}

func TestLogReader_withoutEndOfTextMarker(t *testing.T) {
	t.Parallel()

	logReads := 0
	ts, lr := testLogReader(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logReads++
		switch {
		case logReads == 2:
			w.Write([]byte("\x02"))
		case logReads == 3:
			w.Write([]byte("Terraform run started"))
		case logReads == 16:
			w.Write([]byte(" - logs - "))
		case logReads == 31:
			w.Write([]byte("Terraform run finished"))
		}
	}))
	defer ts.Close()

	doneReads := 0
	lr.done = func() (bool, error) {
		doneReads++
		if logReads >= 31 {
			return true, nil
		}
		return false, nil
	}

	logs, err := ioutil.ReadAll(lr)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Terraform run started - logs - Terraform run finished"
	if string(logs) != expected {
		t.Fatalf("expected %s, got: %s", expected, string(logs))
	}
	if doneReads != 3 {
		t.Fatalf("expected 3 done reads, got %d reads", doneReads)
	}
	if logReads != 42 {
		t.Fatalf("expected 42 log reads, got %d reads", logReads)
	}
}

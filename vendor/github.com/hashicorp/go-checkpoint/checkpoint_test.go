package checkpoint

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestCheck(t *testing.T) {
	expected := &CheckResponse{
		Product:             "test",
		CurrentVersion:      "1.0",
		CurrentReleaseDate:  0,
		CurrentDownloadURL:  "http://www.hashicorp.com",
		CurrentChangelogURL: "http://www.hashicorp.com",
		ProjectWebsite:      "http://www.hashicorp.com",
		Outdated:            false,
		Alerts:              []*CheckAlert{},
	}

	actual, err := Check(&CheckParams{
		Product: "test",
		Version: "1.0",
	})

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestCheck_disabled(t *testing.T) {
	os.Setenv("CHECKPOINT_DISABLE", "1")
	defer os.Setenv("CHECKPOINT_DISABLE", "")

	expected := &CheckResponse{}

	actual, err := Check(&CheckParams{
		Product: "test",
		Version: "1.0",
	})

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected %+v to equal %+v", actual, expected)
	}
}

func TestCheck_cache(t *testing.T) {
	dir, err := ioutil.TempDir("", "checkpoint")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &CheckResponse{
		Product:             "test",
		CurrentVersion:      "1.0",
		CurrentReleaseDate:  0,
		CurrentDownloadURL:  "http://www.hashicorp.com",
		CurrentChangelogURL: "http://www.hashicorp.com",
		ProjectWebsite:      "http://www.hashicorp.com",
		Outdated:            false,
		Alerts:              []*CheckAlert{},
	}

	var actual *CheckResponse
	for i := 0; i < 5; i++ {
		var err error
		actual, err = Check(&CheckParams{
			Product:   "test",
			Version:   "1.0",
			CacheFile: filepath.Join(dir, "cache"),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestCheck_cacheNested(t *testing.T) {
	dir, err := ioutil.TempDir("", "checkpoint")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &CheckResponse{
		Product:             "test",
		CurrentVersion:      "1.0",
		CurrentReleaseDate:  0,
		CurrentDownloadURL:  "http://www.hashicorp.com",
		CurrentChangelogURL: "http://www.hashicorp.com",
		ProjectWebsite:      "http://www.hashicorp.com",
		Outdated:            false,
		Alerts:              []*CheckAlert{},
	}

	var actual *CheckResponse
	for i := 0; i < 5; i++ {
		var err error
		actual, err = Check(&CheckParams{
			Product:   "test",
			Version:   "1.0",
			CacheFile: filepath.Join(dir, "nested", "cache"),
		})
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestCheckInterval(t *testing.T) {
	expected := &CheckResponse{
		Product:             "test",
		CurrentVersion:      "1.0",
		CurrentReleaseDate:  0,
		CurrentDownloadURL:  "http://www.hashicorp.com",
		CurrentChangelogURL: "http://www.hashicorp.com",
		ProjectWebsite:      "http://www.hashicorp.com",
		Outdated:            false,
		Alerts:              []*CheckAlert{},
	}

	params := &CheckParams{
		Product: "test",
		Version: "1.0",
	}

	calledCh := make(chan struct{})
	checkFn := func(actual *CheckResponse, err error) {
		defer close(calledCh)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("bad: %#v", actual)
		}
	}

	doneCh := CheckInterval(params, 500*time.Millisecond, checkFn)
	defer close(doneCh)

	select {
	case <-calledCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestCheckInterval_disabled(t *testing.T) {
	os.Setenv("CHECKPOINT_DISABLE", "1")
	defer os.Setenv("CHECKPOINT_DISABLE", "")

	params := &CheckParams{
		Product: "test",
		Version: "1.0",
	}

	calledCh := make(chan struct{})
	checkFn := func(actual *CheckResponse, err error) {
		defer close(calledCh)
	}

	doneCh := CheckInterval(params, 500*time.Millisecond, checkFn)
	defer close(doneCh)

	select {
	case <-calledCh:
		t.Fatal("expected callback to not invoke")
	case <-time.After(time.Second):
	}
}

func TestRandomStagger(t *testing.T) {
	intv := 24 * time.Hour
	min := 18 * time.Hour
	max := 30 * time.Hour
	for i := 0; i < 1000; i++ {
		out := randomStagger(intv)
		if out < min || out > max {
			t.Fatalf("bad: %v", out)
		}
	}
}

package module

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
)

func init() {
	if os.Getenv("TF_LOG") == "" {
		log.SetOutput(ioutil.Discard)
	}
}

const fixtureDir = "./test-fixtures"

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	return dir
}

func testConfig(t *testing.T, n string) *config.Config {
	t.Helper()
	c, err := config.LoadDir(filepath.Join(fixtureDir, n))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testStorage(t *testing.T, d *disco.Disco) *Storage {
	t.Helper()
	return NewStorage(tempDir(t), d, nil)
}

// test discovery maps registry.terraform.io, localhost, localhost.localdomain,
// and example.com to the test server.
func testDisco(s *httptest.Server) *disco.Disco {
	services := map[string]interface{}{
		// Note that both with and without trailing slashes are supported behaviours
		// TODO: add specific tests to enumerate both possibilities.
		"modules.v1": fmt.Sprintf("%s/v1/modules", s.URL),
	}
	d := disco.NewDisco()

	d.ForceHostServices(svchost.Hostname("registry.terraform.io"), services)
	d.ForceHostServices(svchost.Hostname("localhost"), services)
	d.ForceHostServices(svchost.Hostname("localhost.localdomain"), services)
	d.ForceHostServices(svchost.Hostname("example.com"), services)
	return d
}

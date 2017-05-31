package discovery

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// lists a constant set of providers, and always returns a protocol version
// equal to the Patch number.
func testReleaseServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/terraform-providers/terraform-provider-test/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(versionList))
	})

	return httptest.NewServer(handler)
}

func TestVersionListing(t *testing.T) {
	server := testReleaseServer()
	defer server.Close()

	providersURL.releases = server.URL + "/"

	versions, err := listProviderVersions("test")
	if err != nil {
		t.Fatal(err)
	}

	Versions(versions).Sort()

	expected := []string{
		"1.2.4",
		"1.2.3",
		"1.2.1",
	}

	if len(versions) != len(expected) {
		t.Fatalf("Received wrong number of versions. expected: %q, got: %q", expected, versions)
	}

	for i, v := range versions {
		if v.String() != expected[i] {
			t.Fatalf("incorrect version: %q, expected %q", v, expected[i])
		}
	}
}

func TestNewestVersion(t *testing.T) {
	var available []Version
	for _, v := range []string{"1.2.3", "1.2.1", "1.2.4"} {
		version, err := VersionStr(v).Parse()
		if err != nil {
			t.Fatal(err)
		}
		available = append(available, version)
	}

	reqd, err := ConstraintStr(">1.2.1").Parse()
	if err != nil {
		t.Fatal(err)
	}

	found, err := newestVersion(available, reqd)
	if err != nil {
		t.Fatal(err)
	}

	if found.String() != "1.2.4" {
		t.Fatalf("expected newest version 1.2.4, got: %s", found)
	}

	reqd, err = ConstraintStr("> 1.2.4").Parse()
	if err != nil {
		t.Fatal(err)
	}

	found, err = newestVersion(available, reqd)
	if err == nil {
		t.Fatalf("expceted error, got version %s", found)
	}
}

const versionList = `<!DOCTYPE html>
<html>
<body>
  <ul>
  <li>
    <a href="../">../</a>
  </li>
  <li>
    <a href="/terraform-provider-test/1.2.3/">terraform-provider-test_1.2.3</a>
  </li>
  <li>
    <a href="/terraform-provider-test/1.2.1/">terraform-provider-test_1.2.1</a>
  </li>
  <li>
    <a href="/terraform-provider-test/1.2.4/">terraform-provider-test_1.2.4</a>
  </li>
  </ul>
  <footer>
    Proudly fronted by <a href="https://fastly.com/?utm_source=hashicorp" target="_TOP">Fastly</a>
  </footer>
</body>
</html>
`

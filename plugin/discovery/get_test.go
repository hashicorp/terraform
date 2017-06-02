package discovery

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

const testProviderFile = "test provider binary"

// return the directory listing for the "test" provider
func testListingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(versionList))
}

// returns a 200 for a valid provider url, using the patch number for the
// plugin protocol version.
func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/terraform-providers/terraform-provider-test/" {
		testListingHandler(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	filename := parts[4]

	reg := regexp.MustCompile(`(terraform-provider-test_(\d).(\d).(\d)_([^_]+)_([^._]+)).zip`)

	fileParts := reg.FindStringSubmatch(filename)
	if len(fileParts) != 7 {
		http.Error(w, "invalid provider: "+filename, http.StatusNotFound)
		return
	}

	w.Header().Set(protocolVersionHeader, fileParts[4])

	// write a dummy file
	z := zip.NewWriter(w)
	f, err := z.Create(fileParts[1] + "_X" + fileParts[4])
	if err != nil {
		panic(err)
	}
	io.WriteString(f, testProviderFile)
	z.Close()
}

func testReleaseServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/terraform-providers/terraform-provider-test/", testHandler)

	return httptest.NewServer(handler)
}

func TestMain(m *testing.M) {
	server := testReleaseServer()
	releaseHost = server.URL

	os.Exit(m.Run())
}

func TestVersionListing(t *testing.T) {
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

func TestCheckProtocolVersions(t *testing.T) {
	if checkPlugin(providerURL("test", VersionStr("1.2.3").MustParse().String()), 4) {
		t.Fatal("protocol version 4 is not compatible")
	}

	if !checkPlugin(providerURL("test", VersionStr("1.2.3").MustParse().String()), 3) {
		t.Fatal("protocol version 3 should be compatible")
	}
}

func TestGetProvider(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	// attempt to use an incompatible protocol version
	err = GetProvider(tmpDir, "test", AllVersions, 5)
	if err == nil {
		t.Fatal("protocol version is incompatible")
	}

	err = GetProvider(tmpDir, "test", AllVersions, 3)
	if err != nil {
		t.Fatal(err)
	}

	// we should have version 1.2.3
	fileName := fmt.Sprintf("terraform-provider-test_1.2.3_%s_%s_X3", runtime.GOOS, runtime.GOARCH)
	dest := filepath.Join(tmpDir, fileName)
	f, err := ioutil.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}

	// provider should have been unzipped
	if string(f) != testProviderFile {
		t.Fatalf("test provider contains: %q", f)
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

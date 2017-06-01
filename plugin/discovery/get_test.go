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

func TestFilterProtocolVersions(t *testing.T) {
	versions, err := listProviderVersions("test")
	if err != nil {
		t.Fatal(err)
	}

	// use plugin protocl version 3, which should only return version 1.2.3
	compat := filterProtocolVersions("test", versions, 3)

	if len(compat) != 1 || compat[0].String() != "1.2.3" {
		t.Fatal("found wrong versions: %q", compat)
	}

	compat = filterProtocolVersions("test", versions, 6)
	if len(compat) != 0 {
		t.Fatal("should be no compatible versions, got: %q", compat)
	}
}

func TestGetProvider(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	fileName := fmt.Sprintf("terraform-provider-test_1.2.3_%s_%s_X3", runtime.GOOS, runtime.GOARCH)

	err = GetProvider(tmpDir, "test", AllVersions, 3)
	if err != nil {
		t.Fatal(err)
	}

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

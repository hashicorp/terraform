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
	"reflect"
	"regexp"
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
	if r.URL.Path == "/terraform-provider-test/" {
		testListingHandler(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	filename := parts[3]

	reg := regexp.MustCompile(`(terraform-provider-test)_(\d).(\d).(\d)_([^_]+)_([^._]+).zip`)

	fileParts := reg.FindStringSubmatch(filename)
	if len(fileParts) != 7 {
		http.Error(w, "invalid provider: "+filename, http.StatusNotFound)
		return
	}

	w.Header().Set(protocolVersionHeader, fileParts[4])

	// write a dummy file
	z := zip.NewWriter(w)
	fn := fmt.Sprintf("%s_v%s.%s.%s_x%s", fileParts[1], fileParts[2], fileParts[3], fileParts[4], fileParts[4])
	f, err := z.Create(fn)
	if err != nil {
		panic(err)
	}
	io.WriteString(f, testProviderFile)
	z.Close()
}

func testReleaseServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/terraform-provider-test/", testHandler)

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

func TestProviderInstaller(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	// attempt to use an incompatible protocol version
	i := &ProviderInstaller{
		Dir: tmpDir,

		PluginProtocolVersion: 5,
	}
	_, err = i.Get("test", AllVersions)
	if err == nil {
		t.Fatal("want error for incompatible version")
	}

	i = &ProviderInstaller{
		Dir: tmpDir,

		PluginProtocolVersion: 3,
	}
	gotMeta, err := i.Get("test", AllVersions)
	if err != nil {
		t.Fatal(err)
	}

	// we should have version 1.2.3
	dest := filepath.Join(tmpDir, "terraform-provider-test_v1.2.3_x3")

	wantMeta := PluginMeta{
		Name:    "test",
		Version: VersionStr("1.2.3"),
		Path:    dest,
	}
	if !reflect.DeepEqual(gotMeta, wantMeta) {
		t.Errorf("wrong result meta\ngot:  %#v\nwant: %#v", gotMeta, wantMeta)
	}

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

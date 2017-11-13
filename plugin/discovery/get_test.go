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

	"github.com/mitchellh/cli"
)

const testProviderFile = "test provider binary"

// return the directory listing for the "test" provider
func testListingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(versionList))
}

func testChecksumHandler(w http.ResponseWriter, r *http.Request) {
	// this exact plugin has a signnature and checksum file
	if r.URL.Path == "/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS" {
		http.ServeFile(w, r, "testdata/terraform-provider-template_0.1.0_SHA256SUMS")
		return
	}
	if r.URL.Path == "/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS.sig" {
		http.ServeFile(w, r, "testdata/terraform-provider-template_0.1.0_SHA256SUMS.sig")
		return
	}

	// this this checksum file is corrupt and doesn't match the sig
	if r.URL.Path == "/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS" {
		http.ServeFile(w, r, "testdata/terraform-provider-badsig_0.1.0_SHA256SUMS")
		return
	}
	if r.URL.Path == "/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS.sig" {
		http.ServeFile(w, r, "testdata/terraform-provider-badsig_0.1.0_SHA256SUMS.sig")
		return
	}

	http.Error(w, "signtaure files not found", http.StatusNotFound)
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
	handler.HandleFunc("/terraform-provider-template/", testChecksumHandler)
	handler.HandleFunc("/terraform-provider-badsig/", testChecksumHandler)

	return httptest.NewServer(handler)
}

func TestMain(m *testing.M) {
	server := testReleaseServer()
	releaseHost = server.URL

	os.Exit(m.Run())
}

func TestVersionListing(t *testing.T) {
	i := &ProviderInstaller{}
	versions, err := i.listProviderVersions("test")
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
	i := &ProviderInstaller{}
	if checkPlugin(i.providerURL("test", VersionStr("1.2.3").MustParse().String()), 4) {
		t.Fatal("protocol version 4 is not compatible")
	}

	if !checkPlugin(i.providerURL("test", VersionStr("1.2.3").MustParse().String()), 3) {
		t.Fatal("protocol version 3 should be compatible")
	}
}

func TestProviderInstallerGet(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	// attempt to use an incompatible protocol version
	i := &ProviderInstaller{
		Dir: tmpDir,
		PluginProtocolVersion: 5,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
	}
	_, err = i.Get("test", AllVersions)
	if err != ErrorNoVersionCompatible {
		t.Fatal("want error for incompatible version")
	}

	i = &ProviderInstaller{
		Dir: tmpDir,
		PluginProtocolVersion: 3,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
	}

	{
		_, err := i.Get("test", ConstraintStr(">9.0.0").MustParse())
		if err != ErrorNoSuitableVersion {
			t.Fatal("want error for mismatching constraints")
		}
	}

	{
		_, err := i.Get("nonexist", AllVersions)
		if err != ErrorNoSuchProvider {
			t.Fatal("want error for no such provider")
		}
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

func TestProviderInstallerPurgeUnused(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	unwantedPath := filepath.Join(tmpDir, "terraform-provider-test_v0.0.1_x2")
	wantedPath := filepath.Join(tmpDir, "terraform-provider-test_v1.2.3_x3")

	f, err := os.Create(unwantedPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	f, err = os.Create(wantedPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	i := &ProviderInstaller{
		Dir: tmpDir,
		PluginProtocolVersion: 3,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
	}
	purged, err := i.PurgeUnused(map[string]PluginMeta{
		"test": PluginMeta{
			Name:    "test",
			Version: VersionStr("1.2.3"),
			Path:    wantedPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := purged.Count(), 1; got != want {
		t.Errorf("wrong purged count %d; want %d", got, want)
	}
	if got, want := purged.Newest().Path, unwantedPath; got != want {
		t.Errorf("wrong purged path %s; want %s", got, want)
	}

	files, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	gotFilenames := make([]string, len(files))
	for i, info := range files {
		gotFilenames[i] = info.Name()
	}
	wantFilenames := []string{"terraform-provider-test_v1.2.3_x3"}

	if !reflect.DeepEqual(gotFilenames, wantFilenames) {
		t.Errorf("wrong filenames after purge\ngot:  %#v\nwant: %#v", gotFilenames, wantFilenames)
	}
}

// Test fetching a provider's checksum file while verifying its signature.
func TestProviderChecksum(t *testing.T) {
	i := &ProviderInstaller{}

	// we only need the checksum, as getter is doing the actual file comparison.
	sha256sum, err := i.getProviderChecksum("template", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}

	// get the expected checksum for our os/arch
	sumData, err := ioutil.ReadFile("testdata/terraform-provider-template_0.1.0_SHA256SUMS")
	if err != nil {
		t.Fatal(err)
	}

	expected := checksumForFile(sumData, i.providerFileName("template", "0.1.0"))

	if sha256sum != expected {
		t.Fatalf("expected: %s\ngot %s\n", sha256sum, expected)
	}
}

// Test fetching a provider's checksum file witha bad signature
func TestProviderChecksumBadSignature(t *testing.T) {
	i := &ProviderInstaller{}

	// we only need the checksum, as getter is doing the actual file comparison.
	sha256sum, err := i.getProviderChecksum("badsig", "0.1.0")
	if err == nil {
		t.Fatal("expcted error")
	}

	if !strings.Contains(err.Error(), "signature") {
		t.Fatal("expected signature error, got:", err)
	}

	if sha256sum != "" {
		t.Fatal("expected no checksum, got:", sha256sum)
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

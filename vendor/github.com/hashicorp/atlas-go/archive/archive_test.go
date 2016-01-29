package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

const fixturesDir = "./test-fixtures"

var testHasGit bool
var testHasHg bool

func init() {
	if _, err := exec.LookPath("git"); err == nil {
		testHasGit = true
	}

	if _, err := exec.LookPath("hg"); err == nil {
		testHasHg = true
	}
}

func TestArchiveOptsIsSet(t *testing.T) {
	cases := []struct {
		Opts *ArchiveOpts
		Set  bool
	}{
		{
			&ArchiveOpts{},
			false,
		},
		{
			&ArchiveOpts{VCS: true},
			true,
		},
		{
			&ArchiveOpts{Exclude: make([]string, 0, 0)},
			false,
		},
		{
			&ArchiveOpts{Exclude: []string{"foo"}},
			true,
		},
		{
			&ArchiveOpts{Include: make([]string, 0, 0)},
			false,
		},
		{
			&ArchiveOpts{Include: []string{"foo"}},
			true,
		},
	}

	for i, tc := range cases {
		if tc.Opts.IsSet() != tc.Set {
			t.Fatalf("%d: expected %#v", i, tc.Set)
		}
	}
}

func TestArchive_file(t *testing.T) {
	path := filepath.Join(testFixture("archive-file"), "foo.txt")
	r, err := CreateArchive(path, new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_fileCompressed(t *testing.T) {
	path := filepath.Join(testFixture("archive-file-compressed"), "file.tar.gz")
	r, err := CreateArchive(path, new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"./foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_fileNoExist(t *testing.T) {
	tf := tempFile(t)
	if err := os.Remove(tf); err != nil {
		t.Fatalf("err: %s", err)
	}

	r, err := CreateArchive(tf, nil)
	if err == nil {
		t.Fatal("err should not be nil")
	}
	if r != nil {
		t.Fatal("should be nil")
	}
}

func TestArchive_fileWithOpts(t *testing.T) {
	r, err := CreateArchive(tempFile(t), &ArchiveOpts{VCS: true})
	if err == nil {
		t.Fatal("err should not be nil")
	}
	if r != nil {
		t.Fatal("should be nil")
	}
}

func TestArchive_dirExtra(t *testing.T) {
	opts := &ArchiveOpts{
		Extra: map[string]string{
			"hello.txt": filepath.Join(
				testFixture("archive-subdir"), "subdir", "hello.txt"),
		},
	}

	r, err := CreateArchive(testFixture("archive-flat"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"baz.txt",
		"foo.txt",
		"hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirExtraDir(t *testing.T) {
	opts := &ArchiveOpts{
		Extra: map[string]string{
			"foo": filepath.Join(testFixture("archive-subdir"), "subdir"),
		},
	}

	r, err := CreateArchive(testFixture("archive-flat"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"baz.txt",
		"foo.txt",
		"foo/",
		"foo/hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("modes don't work on Windows")
	}

	opts := &ArchiveOpts{}

	r, err := CreateArchive(testFixture("archive-dir-mode"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"file.txt-exec",
	}

	entries := testArchive(t, r, true)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}
func TestArchive_dirSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("git symlinks don't work on Windows")
	}

	path := filepath.Join(testFixture("archive-symlink"), "link", "link")
	r, err := CreateArchive(path, new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirWithSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("git symlinks don't work on Windows")
	}

	path := filepath.Join(testFixture("archive-symlink"), "link")
	r, err := CreateArchive(path, new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"link/",
		"link/foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirNoVCS(t *testing.T) {
	r, err := CreateArchive(testFixture("archive-flat"), new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"baz.txt",
		"foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirSubdirsNoVCS(t *testing.T) {
	r, err := CreateArchive(testFixture("archive-subdir"), new(ArchiveOpts))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"bar.txt",
		"foo.txt",
		"subdir/",
		"subdir/hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirExclude(t *testing.T) {
	opts := &ArchiveOpts{
		Exclude: []string{"subdir", "subdir/*"},
	}

	r, err := CreateArchive(testFixture("archive-subdir"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"bar.txt",
		"foo.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirInclude(t *testing.T) {
	opts := &ArchiveOpts{
		Include: []string{"bar.txt"},
	}

	r, err := CreateArchive(testFixture("archive-subdir"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"bar.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_dirIncludeStar(t *testing.T) {
	opts := &ArchiveOpts{
		Include: []string{"build/**/*"},
	}

	r, err := CreateArchive(testFixture("archive-subdir-splat"), opts)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"build/",
		"build/darwin-amd64/",
		"build/darwin-amd64/build.txt",
		"build/linux-amd64/",
		"build/linux-amd64/build.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_git(t *testing.T) {
	if !testHasGit {
		t.Log("git not found, skipping")
		t.Skip()
	}

	// Git doesn't allow nested ".git" directories so we do some hackiness
	// here to get around that...
	testDir := testFixture("archive-git")
	oldName := filepath.ToSlash(filepath.Join(testDir, "DOTgit"))
	newName := filepath.ToSlash(filepath.Join(testDir, ".git"))
	os.Remove(newName)
	if err := os.Rename(oldName, newName); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Rename(newName, oldName)

	// testDir with VCS set to true
	r, err := CreateArchive(testDir, &ArchiveOpts{VCS: true})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"bar.txt",
		"foo.txt",
		"subdir/",
		"subdir/hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}

	// Test that metadata was added
	if r.Metadata == nil {
		t.Fatal("expected archive metadata to be set")
	}

	expectedMetadata := map[string]string{
		"branch":          "master",
		"commit":          "7525d17cbbb56f3253a20903ffddc07c6c935c76",
		"remote.origin":   "https://github.com/hashicorp/origin.git",
		"remote.upstream": "https://github.com/hashicorp/upstream.git",
	}

	if !reflect.DeepEqual(r.Metadata, expectedMetadata) {
		t.Fatalf("expected %+v to be %+v", r.Metadata, expectedMetadata)
	}
}

func TestArchive_gitSubdir(t *testing.T) {
	if !testHasGit {
		t.Log("git not found, skipping")
		t.Skip()
	}

	// Git doesn't allow nested ".git" directories so we do some hackiness
	// here to get around that...
	testDir := testFixture("archive-git")
	oldName := filepath.ToSlash(filepath.Join(testDir, "DOTgit"))
	newName := filepath.ToSlash(filepath.Join(testDir, ".git"))
	os.Remove(newName)
	if err := os.Rename(oldName, newName); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Rename(newName, oldName)

	// testDir with VCS set to true
	r, err := CreateArchive(filepath.Join(testDir, "subdir"), &ArchiveOpts{VCS: true})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("bad: %#v", entries)
	}
}

func TestArchive_hg(t *testing.T) {
	if !testHasHg {
		t.Log("hg not found, skipping")
		t.Skip()
	}

	// testDir with VCS set to true
	testDir := testFixture("archive-hg")
	r, err := CreateArchive(testDir, &ArchiveOpts{VCS: true})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"bar.txt",
		"foo.txt",
		"subdir/",
		"subdir/hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("\n-- Expected --\n%#v\n-- Found --\n%#v", expected, entries)
	}
}

func TestArchive_hgSubdir(t *testing.T) {
	if !testHasHg {
		t.Log("hg not found, skipping")
		t.Skip()
	}

	// testDir with VCS set to true
	testDir := filepath.Join(testFixture("archive-hg"), "subdir")
	r, err := CreateArchive(testDir, &ArchiveOpts{VCS: true})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{
		"hello.txt",
	}

	entries := testArchive(t, r, false)
	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("\n-- Expected --\n%#v\n-- Found --\n%#v", expected, entries)
	}
}

func TestReadCloseRemover(t *testing.T) {
	f, err := ioutil.TempFile("", "atlas-go")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	r := &readCloseRemover{F: f}
	if err := r.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := os.Stat(f.Name()); err == nil {
		t.Fatal("file should not exist anymore")
	}
}

func testArchive(t *testing.T, r *Archive, detailed bool) []string {
	// Finish the archiving process in-memory
	var buf bytes.Buffer
	n, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if n != r.Size {
		t.Fatalf("bad size: %d (expected: %d)", n, r.Size)
	}

	gzipR, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tarR := tar.NewReader(gzipR)

	// Read all the entries
	result := make([]string, 0, 5)
	for {
		hdr, err := tarR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		text := hdr.Name
		if detailed {
			// Check if the file is executable. We use these stub names
			// to compensate for umask differences in test environments
			// and limitations in using "git clone".
			if hdr.FileInfo().Mode()&0111 != 0 {
				text = hdr.Name + "-exec"
			} else {
				text = hdr.Name + "-reg"
			}
		}

		result = append(result, text)
	}

	sort.Strings(result)
	return result
}

func tempFile(t *testing.T) string {
	tf, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer tf.Close()

	return tf.Name()
}

func testFixture(n string) string {
	return filepath.Join(fixturesDir, n)
}

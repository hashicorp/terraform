package configload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/afero"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// LoadConfigWithSnapshot is a variant of LoadConfig that also simultaneously
// creates an in-memory snapshot of the configuration files used, which can
// be later used to create a loader that may read only from this snapshot.
func (l *Loader) LoadConfigWithSnapshot(rootDir string) (*configs.Config, *Snapshot, hcl.Diagnostics) {
	rootMod, diags := l.parser.LoadConfigDir(rootDir)
	if rootMod == nil {
		return nil, nil, diags
	}

	snap := &Snapshot{
		Modules: map[string]*SnapshotModule{},
	}
	walker := l.makeModuleWalkerSnapshot(snap)
	cfg, cDiags := configs.BuildConfig(rootMod, walker)
	diags = append(diags, cDiags...)

	addDiags := l.addModuleToSnapshot(snap, "", rootDir, "", nil)
	diags = append(diags, addDiags...)

	return cfg, snap, diags
}

// NewLoaderFromSnapshot creates a Loader that reads files only from the
// given snapshot.
//
// A snapshot-based loader cannot install modules, so calling InstallModules
// on the return value will cause a panic.
//
// A snapshot-based loader also has access only to configuration files. Its
// underlying parser does not have access to other files in the native
// filesystem, such as values files. For those, either use a normal loader
// (created by NewLoader) or use the configs.Parser API directly.
func NewLoaderFromSnapshot(snap *Snapshot) *Loader {
	fs := snapshotFS{snap}
	parser := configs.NewParser(fs)

	ret := &Loader{
		parser: parser,
		modules: moduleMgr{
			FS:         afero.Afero{Fs: fs},
			CanInstall: false,
			manifest:   snap.moduleManifest(),
		},
	}

	return ret
}

// Snapshot is an in-memory representation of the source files from a
// configuration, which can be used as an alternative configurations source
// for a loader with NewLoaderFromSnapshot.
//
// The primary purpose of a Snapshot is to build the configuration portion
// of a plan file (see ../../plans/planfile) so that it can later be reloaded
// and used to recover the exact configuration that the plan was built from.
type Snapshot struct {
	// Modules is a map from opaque module keys (suitable for use as directory
	// names on all supported operating systems) to the snapshot information
	// about each module.
	Modules map[string]*SnapshotModule
}

// NewEmptySnapshot constructs and returns a snapshot containing only an empty
// root module. This is not useful for anything except placeholders in tests.
func NewEmptySnapshot() *Snapshot {
	return &Snapshot{
		Modules: map[string]*SnapshotModule{
			manifestKey(addrs.RootModule): &SnapshotModule{
				Files: map[string][]byte{},
			},
		},
	}
}

// SnapshotModule represents a single module within a Snapshot.
type SnapshotModule struct {
	// Dir is the path, relative to the root directory given when the
	// snapshot was created, where the module appears in the snapshot's
	// virtual filesystem.
	Dir string

	// Files is a map from each configuration file filename for the
	// module to a raw byte representation of the source file contents.
	Files map[string][]byte

	// SourceAddr is the source address given for this module in configuration.
	SourceAddr string `json:"Source"`

	// Version is the version of the module that is installed, or nil if
	// the module is installed from a source that does not support versions.
	Version *version.Version `json:"-"`
}

// moduleManifest constructs a module manifest based on the contents of
// the receiving snapshot.
func (s *Snapshot) moduleManifest() moduleManifest {
	ret := make(moduleManifest)

	for k, modSnap := range s.Modules {
		ret[k] = moduleRecord{
			Key:        k,
			Dir:        modSnap.Dir,
			SourceAddr: modSnap.SourceAddr,
			Version:    modSnap.Version,
		}
	}

	return ret
}

// makeModuleWalkerSnapshot creates a configs.ModuleWalker that will exhibit
// the same lookup behaviors as l.moduleWalkerLoad but will additionally write
// source files from the referenced modules into the given snapshot.
func (l *Loader) makeModuleWalkerSnapshot(snap *Snapshot) configs.ModuleWalker {
	return configs.ModuleWalkerFunc(
		func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
			mod, v, diags := l.moduleWalkerLoad(req)
			if diags.HasErrors() {
				return mod, v, diags
			}

			key := manifestKey(req.Path)
			record, exists := l.modules.manifest[key]

			if !exists {
				// Should never happen, since otherwise moduleWalkerLoader would've
				// returned an error and we would've returned already.
				panic(fmt.Sprintf("module %s is not present in manifest", key))
			}

			addDiags := l.addModuleToSnapshot(snap, key, record.Dir, record.SourceAddr, record.Version)
			diags = append(diags, addDiags...)

			return mod, v, diags
		},
	)
}

func (l *Loader) addModuleToSnapshot(snap *Snapshot, key string, dir string, sourceAddr string, v *version.Version) hcl.Diagnostics {
	var diags hcl.Diagnostics

	primaryFiles, overrideFiles, moreDiags := l.parser.ConfigDirFiles(dir)
	if moreDiags.HasErrors() {
		// Any diagnostics we get here should be already present
		// in diags, so it's weird if we get here but we'll allow it
		// and return a general error message in that case.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read directory for module",
			Detail:   fmt.Sprintf("The source directory %s could not be read", dir),
		})
		return diags
	}

	snapMod := &SnapshotModule{
		Dir:        dir,
		Files:      map[string][]byte{},
		SourceAddr: sourceAddr,
		Version:    v,
	}

	files := make([]string, 0, len(primaryFiles)+len(overrideFiles))
	files = append(files, primaryFiles...)
	files = append(files, overrideFiles...)
	sources := l.Sources() // should be populated with all the files we need by now
	for _, filePath := range files {
		filename := filepath.Base(filePath)
		src, exists := sources[filePath]
		if !exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing source file for snapshot",
				Detail:   fmt.Sprintf("The source code for file %s could not be found to produce a configuration snapshot.", filePath),
			})
			continue
		}
		snapMod.Files[filepath.Clean(filename)] = src
	}

	snap.Modules[key] = snapMod

	return diags
}

// snapshotFS is an implementation of afero.Fs that reads from a snapshot.
//
// This is not intended as a general-purpose filesystem implementation. Instead,
// it just supports the minimal functionality required to support the
// configuration loader and parser as an implementation detail of creating
// a loader from a snapshot.
type snapshotFS struct {
	snap *Snapshot
}

var _ afero.Fs = snapshotFS{}

func (fs snapshotFS) Create(name string) (afero.File, error) {
	return nil, fmt.Errorf("cannot create file inside configuration snapshot")
}

func (fs snapshotFS) Mkdir(name string, perm os.FileMode) error {
	return fmt.Errorf("cannot create directory inside configuration snapshot")
}

func (fs snapshotFS) MkdirAll(name string, perm os.FileMode) error {
	return fmt.Errorf("cannot create directories inside configuration snapshot")
}

func (fs snapshotFS) Open(name string) (afero.File, error) {

	// Our "filesystem" is sparsely populated only with the directories
	// mentioned by modules in our snapshot, so the high-level process
	// for opening a file is:
	// - Find the module snapshot corresponding to the containing directory
	// - Find the file within that snapshot
	// - Wrap the resulting byte slice in a snapshotFile to return
	//
	// The other possibility handled here is if the given name is for the
	// module directory itself, in which case we'll return a snapshotDir
	// instead.
	//
	// This function doesn't try to be incredibly robust in supporting
	// different permutations of paths, etc because in practice we only
	// need to support the path forms that our own loader and parser will
	// generate.

	dir := filepath.Dir(name)
	fn := filepath.Base(name)
	directDir := filepath.Clean(name)

	// First we'll check to see if this is an exact path for a module directory.
	// We need to do this first (rather than as part of the next loop below)
	// because a module in a child directory of another module can otherwise
	// appear to be a file in that parent directory.
	for _, candidate := range fs.snap.Modules {
		modDir := filepath.Clean(candidate.Dir)
		if modDir == directDir {
			// We've matched the module directory itself
			filenames := make([]string, 0, len(candidate.Files))
			for n := range candidate.Files {
				filenames = append(filenames, n)
			}
			sort.Strings(filenames)
			return snapshotDir{
				filenames: filenames,
			}, nil
		}
	}

	// If we get here then the given path isn't a module directory exactly, so
	// we'll treat it as a file path and try to find a module directory it
	// could be located in.
	var modSnap *SnapshotModule
	for _, candidate := range fs.snap.Modules {
		modDir := filepath.Clean(candidate.Dir)
		if modDir == dir {
			modSnap = candidate
			break
		}
	}
	if modSnap == nil {
		return nil, os.ErrNotExist
	}

	src, exists := modSnap.Files[fn]
	if !exists {
		return nil, os.ErrNotExist
	}

	return &snapshotFile{
		src: src,
	}, nil
}

func (fs snapshotFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return fs.Open(name)
}

func (fs snapshotFS) Remove(name string) error {
	return fmt.Errorf("cannot remove file inside configuration snapshot")
}

func (fs snapshotFS) RemoveAll(path string) error {
	return fmt.Errorf("cannot remove files inside configuration snapshot")
}

func (fs snapshotFS) Rename(old, new string) error {
	return fmt.Errorf("cannot rename file inside configuration snapshot")
}

func (fs snapshotFS) Stat(name string) (os.FileInfo, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	_, isDir := f.(snapshotDir)
	return snapshotFileInfo{
		name:  filepath.Base(name),
		isDir: isDir,
	}, nil
}

func (fs snapshotFS) Name() string {
	return "ConfigSnapshotFS"
}

func (fs snapshotFS) Chmod(name string, mode os.FileMode) error {
	return fmt.Errorf("cannot set file mode inside configuration snapshot")
}

func (fs snapshotFS) Chtimes(name string, atime, mtime time.Time) error {
	return fmt.Errorf("cannot set file times inside configuration snapshot")
}

type snapshotFile struct {
	snapshotFileStub
	src []byte
	at  int64
}

var _ afero.File = (*snapshotFile)(nil)

func (f *snapshotFile) Read(p []byte) (n int, err error) {
	if len(p) > 0 && f.at == int64(len(f.src)) {
		return 0, io.EOF
	}
	if f.at > int64(len(f.src)) {
		return 0, io.ErrUnexpectedEOF
	}
	if int64(len(f.src))-f.at >= int64(len(p)) {
		n = len(p)
	} else {
		n = int(int64(len(f.src)) - f.at)
	}
	copy(p, f.src[f.at:f.at+int64(n)])
	f.at += int64(n)
	return
}

func (f *snapshotFile) ReadAt(p []byte, off int64) (n int, err error) {
	f.at = off
	return f.Read(p)
}

func (f *snapshotFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.at = offset
	case 1:
		f.at += offset
	case 2:
		f.at = int64(len(f.src)) + offset
	}
	return f.at, nil
}

type snapshotDir struct {
	snapshotFileStub
	filenames []string
	at        int
}

var _ afero.File = snapshotDir{}

func (f snapshotDir) Readdir(count int) ([]os.FileInfo, error) {
	names, err := f.Readdirnames(count)
	if err != nil {
		return nil, err
	}
	ret := make([]os.FileInfo, len(names))
	for i, name := range names {
		ret[i] = snapshotFileInfo{
			name:  name,
			isDir: false,
		}
	}
	return ret, nil
}

func (f snapshotDir) Readdirnames(count int) ([]string, error) {
	var outLen int
	names := f.filenames[f.at:]
	if count > 0 {
		if len(names) < count {
			outLen = len(names)
		} else {
			outLen = count
		}
		if len(names) == 0 {
			return nil, io.EOF
		}
	} else {
		outLen = len(names)
	}
	f.at += outLen

	return names[:outLen], nil
}

// snapshotFileInfo is a minimal implementation of os.FileInfo to support our
// virtual filesystem from snapshots.
type snapshotFileInfo struct {
	name  string
	isDir bool
}

var _ os.FileInfo = snapshotFileInfo{}

func (fi snapshotFileInfo) Name() string {
	return fi.name
}

func (fi snapshotFileInfo) Size() int64 {
	// In practice, our parser and loader never call Size
	return -1
}

func (fi snapshotFileInfo) Mode() os.FileMode {
	return os.ModePerm
}

func (fi snapshotFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi snapshotFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi snapshotFileInfo) Sys() interface{} {
	return nil
}

type snapshotFileStub struct{}

func (f snapshotFileStub) Close() error {
	return nil
}

func (f snapshotFileStub) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("cannot read")
}

func (f snapshotFileStub) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, fmt.Errorf("cannot read")
}

func (f snapshotFileStub) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("cannot seek")
}

func (f snapshotFileStub) Write(p []byte) (n int, err error) {
	return f.WriteAt(p, 0)
}

func (f snapshotFileStub) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, fmt.Errorf("cannot write to file in snapshot")
}

func (f snapshotFileStub) WriteString(s string) (n int, err error) {
	return 0, fmt.Errorf("cannot write to file in snapshot")
}

func (f snapshotFileStub) Name() string {
	// in practice, the loader and parser never use this
	return "<unimplemented>"
}

func (f snapshotFileStub) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("cannot use Readdir on a file")
}

func (f snapshotFileStub) Readdirnames(count int) ([]string, error) {
	return nil, fmt.Errorf("cannot use Readdir on a file")
}

func (f snapshotFileStub) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("cannot stat")
}

func (f snapshotFileStub) Sync() error {
	return nil
}

func (f snapshotFileStub) Truncate(size int64) error {
	return fmt.Errorf("cannot write to file in snapshot")
}

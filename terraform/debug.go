package terraform

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DebugInfo is the global handler for writing the debug archive. All methods
// are safe to call concurrently. Setting DebugInfo to nil will disable writing
// the debug archive. All methods are safe to call on the nil value.
var dbug *debugInfo

// SetDebugInfo initializes the debug handler with a backing file in the
// provided directory. This must be called before any other terraform package
// operations or not at all. Once his is called, CloseDebugInfo should be
// called before program exit.
func SetDebugInfo(path string) error {
	if os.Getenv("TF_DEBUG") == "" {
		return nil
	}

	di, err := newDebugInfoFile(path)
	if err != nil {
		return err
	}

	dbug = di
	return nil
}

// CloseDebugInfo is the exported interface to Close the debug info handler.
// The debug handler needs to be closed before program exit, so we export this
// function to be deferred in the appropriate entrypoint for our executable.
func CloseDebugInfo() error {
	return dbug.Close()
}

// newDebugInfoFile initializes the global debug handler with a backing file in
// the provided directory.
func newDebugInfoFile(dir string) (*debugInfo, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	// FIXME: not guaranteed unique, but good enough for now
	name := fmt.Sprintf("debug-%s", time.Now().Format("2006-01-02-15-04-05.999999999"))
	archivePath := filepath.Join(dir, name+".tar.gz")

	f, err := os.OpenFile(archivePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return nil, err
	}
	return newDebugInfo(name, f)
}

// newDebugInfo initializes the global debug handler.
func newDebugInfo(name string, w io.Writer) (*debugInfo, error) {
	gz := gzip.NewWriter(w)

	d := &debugInfo{
		name: name,
		w:    w,
		gz:   gz,
		tar:  tar.NewWriter(gz),
	}

	// create the subdirs we need
	topHdr := &tar.Header{
		Name:     name,
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	graphsHdr := &tar.Header{
		Name:     name + "/graphs",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	err := d.tar.WriteHeader(topHdr)
	// if the first errors, the second will too
	err = d.tar.WriteHeader(graphsHdr)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// debugInfo provides various methods for writing debug information to a
// central archive. The debugInfo struct should be initialized once before any
// output is written, and Close should be called before program exit. All
// exported methods on debugInfo will be safe for concurrent use. The exported
// methods are also all safe to call on a nil pointer, so that there is no need
// for conditional blocks before writing debug information.
//
// Each write operation done by the debugInfo will flush the gzip.Writer and
// tar.Writer, and call Sync() or Flush() on the output writer as needed. This
// ensures that as much data as possible is written to storage in the event of
// a crash. The append format of the tar file, and the stream format of the
// gzip writer allow easy recovery f the data in the event that the debugInfo
// is not closed before program exit.
type debugInfo struct {
	sync.Mutex

	// archive root directory name
	name string

	// current operation phase
	phase string

	// step is monotonic counter for for recording the order of operations
	step int

	// flag to protect Close()
	closed bool

	// the debug log output is in a tar.gz format, written to the io.Writer w
	w   io.Writer
	gz  *gzip.Writer
	tar *tar.Writer
}

// Set the name of the current operational phase in the debug handler. Each file
// in the archive will contain the name of the phase in which it was created,
// i.e. "input", "apply", "plan", "refresh", "validate"
func (d *debugInfo) SetPhase(phase string) {
	if d == nil {
		return
	}
	d.Lock()
	defer d.Unlock()

	d.phase = phase
}

// Close the debugInfo, finalizing the data in storage. This closes the
// tar.Writer, the gzip.Wrtier, and if the output writer is an io.Closer, it is
// also closed.
func (d *debugInfo) Close() error {
	if d == nil {
		return nil
	}

	d.Lock()
	defer d.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	d.tar.Close()
	d.gz.Close()

	if c, ok := d.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// debug buffer is an io.WriteCloser that will write itself to the debug
// archive when closed.
type debugBuffer struct {
	debugInfo *debugInfo
	name      string
	buf       bytes.Buffer
}

func (b *debugBuffer) Write(d []byte) (int, error) {
	return b.buf.Write(d)
}

func (b *debugBuffer) Close() error {
	return b.debugInfo.WriteFile(b.name, b.buf.Bytes())
}

// ioutils only has a noop ReadCloser
type nopWriteCloser struct{}

func (nopWriteCloser) Write([]byte) (int, error) { return 0, nil }
func (nopWriteCloser) Close() error              { return nil }

// NewFileWriter returns an io.WriteClose that will be buffered and written to
// the debug archive when closed.
func (d *debugInfo) NewFileWriter(name string) io.WriteCloser {
	if d == nil {
		return nopWriteCloser{}
	}

	return &debugBuffer{
		debugInfo: d,
		name:      name,
	}
}

type syncer interface {
	Sync() error
}

type flusher interface {
	Flush() error
}

// Flush the tar.Writer and the gzip.Writer. Flush() or Sync() will be called
// on the output writer if they are available.
func (d *debugInfo) flush() {
	d.tar.Flush()
	d.gz.Flush()

	if f, ok := d.w.(flusher); ok {
		f.Flush()
	}

	if s, ok := d.w.(syncer); ok {
		s.Sync()
	}
}

// WriteFile writes data as a single file to the debug arhive.
func (d *debugInfo) WriteFile(name string, data []byte) error {
	if d == nil {
		return nil
	}

	d.Lock()
	defer d.Unlock()
	return d.writeFile(name, data)
}

func (d *debugInfo) writeFile(name string, data []byte) error {
	defer d.flush()
	path := fmt.Sprintf("%s/%d-%s-%s", d.name, d.step, d.phase, name)
	d.step++

	hdr := &tar.Header{
		Name: path,
		Mode: 0644,
		Size: int64(len(data)),
	}
	err := d.tar.WriteHeader(hdr)
	if err != nil {
		return err
	}

	_, err = d.tar.Write(data)
	return err
}

// DebugHook implements all methods of the terraform.Hook interface, and writes
// the arguments to a file in the archive. When a suitable format for the
// argument isn't available, the argument is encoded using json.Marshal. If the
// debug handler is nil, all DebugHook methods are noop, so no time is spent in
// marshaling the data structures.
type DebugHook struct{}

func (*DebugHook) PreApply(ii *InstanceInfo, is *InstanceState, id *InstanceDiff) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer

	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	if is != nil {
		buf.WriteString(is.String() + "\n")
	}

	idCopy, err := id.Copy()
	if err != nil {
		return HookActionContinue, err
	}
	js, err := json.MarshalIndent(idCopy, "", "  ")
	if err != nil {
		return HookActionContinue, err
	}
	buf.Write(js)

	dbug.WriteFile("hook-PreApply", buf.Bytes())

	return HookActionContinue, nil
}

func (*DebugHook) PostApply(ii *InstanceInfo, is *InstanceState, err error) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer

	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	if is != nil {
		buf.WriteString(is.String() + "\n")
	}

	if err != nil {
		buf.WriteString(err.Error())
	}

	dbug.WriteFile("hook-PostApply", buf.Bytes())

	return HookActionContinue, nil
}

func (*DebugHook) PreDiff(ii *InstanceInfo, is *InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	if is != nil {
		buf.WriteString(is.String())
		buf.WriteString("\n")
	}
	dbug.WriteFile("hook-PreDiff", buf.Bytes())

	return HookActionContinue, nil
}

func (*DebugHook) PostDiff(ii *InstanceInfo, id *InstanceDiff) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	idCopy, err := id.Copy()
	if err != nil {
		return HookActionContinue, err
	}
	js, err := json.MarshalIndent(idCopy, "", "  ")
	if err != nil {
		return HookActionContinue, err
	}
	buf.Write(js)

	dbug.WriteFile("hook-PostDiff", buf.Bytes())

	return HookActionContinue, nil
}

func (*DebugHook) PreProvisionResource(ii *InstanceInfo, is *InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	if is != nil {
		buf.WriteString(is.String())
		buf.WriteString("\n")
	}
	dbug.WriteFile("hook-PreProvisionResource", buf.Bytes())

	return HookActionContinue, nil
}

func (*DebugHook) PostProvisionResource(ii *InstanceInfo, is *InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId())
		buf.WriteString("\n")
	}

	if is != nil {
		buf.WriteString(is.String())
		buf.WriteString("\n")
	}
	dbug.WriteFile("hook-PostProvisionResource", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) PreProvision(ii *InstanceInfo, s string) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId())
		buf.WriteString("\n")
	}
	buf.WriteString(s + "\n")

	dbug.WriteFile("hook-PreProvision", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) PostProvision(ii *InstanceInfo, s string) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}
	buf.WriteString(s + "\n")

	dbug.WriteFile("hook-PostProvision", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) ProvisionOutput(ii *InstanceInfo, s1 string, s2 string) {
	if dbug == nil {
		return
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId())
		buf.WriteString("\n")
	}
	buf.WriteString(s1 + "\n")
	buf.WriteString(s2 + "\n")

	dbug.WriteFile("hook-ProvisionOutput", buf.Bytes())
}

func (*DebugHook) PreRefresh(ii *InstanceInfo, is *InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	if is != nil {
		buf.WriteString(is.String())
		buf.WriteString("\n")
	}
	dbug.WriteFile("hook-PreRefresh", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) PostRefresh(ii *InstanceInfo, is *InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId())
		buf.WriteString("\n")
	}

	if is != nil {
		buf.WriteString(is.String())
		buf.WriteString("\n")
	}
	dbug.WriteFile("hook-PostRefresh", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) PreImportState(ii *InstanceInfo, s string) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer
	if ii != nil {
		buf.WriteString(ii.HumanId())
		buf.WriteString("\n")
	}
	buf.WriteString(s + "\n")

	dbug.WriteFile("hook-PreImportState", buf.Bytes())
	return HookActionContinue, nil
}

func (*DebugHook) PostImportState(ii *InstanceInfo, iss []*InstanceState) (HookAction, error) {
	if dbug == nil {
		return HookActionContinue, nil
	}

	var buf bytes.Buffer

	if ii != nil {
		buf.WriteString(ii.HumanId() + "\n")
	}

	for _, is := range iss {
		if is != nil {
			buf.WriteString(is.String() + "\n")
		}
	}
	dbug.WriteFile("hook-PostImportState", buf.Bytes())
	return HookActionContinue, nil
}

// skip logging this for now, since it could be huge
func (*DebugHook) PostStateUpdate(*State) (HookAction, error) {
	return HookActionContinue, nil
}

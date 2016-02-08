
# winrmtest

An in-progress testing package to compliment the [masterzen/winrm](https://github.com/masterzen/winrm) Go-based winrm library.

My primary use-case for this is for [dylanmei/packer-communicator-winrm](https://github.com/dylanmei/packer-communicator-winrm), a [Packer](http://packer.io) communicator plugin for interacting with machines using Windows Remote Management.

## Example Use

A fictitious "Windows tools" package.

```

package wintools

import (
	"io"
	"testing"
	"github.com/dylanmei/winrmtest"
)

func Test_empty_temp_directory(t *testing.T) {
	r := winrmtest.NewRemote()
	defer r.Close()

	r.CommandFunc(wimrmtest.MatchText("dir C:\Temp"), func(out, err io.Writer) int {
		out.Write([]byte(` Volume in drive C is Windows 2012 R2
 Volume Serial Number is XXXX-XXXX

 Directory of C:\

File Not Found`))
		return 0
	})

	lister := NewDirectoryLister(r.Host, r.Port)
	list, _ := lister.TempDirectory()

	if count := len(list.Dirs()); count != 0 {
		t.Errorf("Expected 0 directories but found %d.\n", count)
	}

	if count := len(list.Files()); count != 0 {
		t.Errorf("Expected 0 files but found %d.\n", count)
	}
}
```


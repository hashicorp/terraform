package statefile

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-test/deep"

	tfversion "github.com/hashicorp/terraform/version"
)

func TestRoundtrip(t *testing.T) {
	const dir = "testdata/roundtrip"
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	currentVersion := tfversion.Version

	for _, info := range entries {
		const inSuffix = ".in.tfstate"
		const outSuffix = ".out.tfstate"

		if info.IsDir() {
			continue
		}
		inName := info.Name()
		if !strings.HasSuffix(inName, inSuffix) {
			continue
		}
		name := inName[:len(inName)-len(inSuffix)]
		outName := name + outSuffix

		t.Run(name, func(t *testing.T) {
			ir, err := os.Open(filepath.Join(dir, inName))
			if err != nil {
				t.Fatal(err)
			}
			oSrcWant, err := ioutil.ReadFile(filepath.Join(dir, outName))
			if err != nil {
				t.Fatal(err)
			}

			f, err := Read(ir)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			var buf bytes.Buffer
			err = Write(f, &buf)
			if err != nil {
				t.Fatal(err)
			}
			oSrcGot := buf.Bytes()

			var oGot, oWant map[string]interface{}
			err = json.Unmarshal(oSrcGot, &oGot)
			if err != nil {
				t.Fatalf("result isn't JSON: %s", err)
			}
			err = json.Unmarshal(oSrcWant, &oWant)
			if err != nil {
				t.Fatalf("wanted result isn't JSON: %s", err)
			}

			// A newly written state should always reflect the current terraform version.
			oWant["terraform_version"] = currentVersion

			problems := deep.Equal(oGot, oWant)
			sort.Strings(problems)
			for _, problem := range problems {
				t.Error(problem)
			}
		})
	}
}

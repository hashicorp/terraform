package module

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type TerraformfileEntry struct {
	MatchVersion string `json:"matchVersion,omitempty"`
	Source       string `json:"source"`
	Version      string `json:"version,omitempty"`
}

type Terraformfile map[string]TerraformfileEntry

func NewTerraformfile() (*Terraformfile, error) {
	var tfile Terraformfile
	var tfilepath string
	tfilepathenv := os.Getenv("TERRAFORMFILE_PATH")
	if tfilepathenv != "" {
		tfilepath = tfilepathenv
	}
	tfilepath = "./Terraformfile"

	tfiledata, err := ioutil.ReadFile(tfilepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Terrformfile is not a must.
			return nil, nil
		}
		return nil, err
	}

	err = json.Unmarshal(tfiledata, &tfile)
	if err != nil {
		return nil, err
	}
	return &tfile, nil
}

func (tfile *Terraformfile) GetTerraformEntryOk(source string) (TerraformfileEntry, bool) {
	if tfile == nil {
		return TerraformfileEntry{}, false
	}
	entry, ok := (*tfile)[source]
	return entry, ok
}

//
// func (tfile *Terraformfile) InTerraformfile() bool {
// 	if len(tfile) < 1 {
// 		return false
// 	}
// 	return
// }

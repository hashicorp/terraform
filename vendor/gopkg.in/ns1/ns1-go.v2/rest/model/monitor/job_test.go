package monitor

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJobs(t *testing.T) {
	data := []byte(`[
  {
    "id": "52a27d4397d5f07003fdbe7b",
    "config": {
      "host": "1.2.3.4"
    },
    "status": {
      "lga": {
        "since": 1389407609,
        "status": "up"
      },
      "global": {
        "since": 1389407609,
        "status": "up"
      },
      "sjc": {
        "since": 1389404014,
        "status": "up"
      }
    },
    "rules": [
      {
        "key": "rtt",
        "value": 100,
        "comparison": "<"
      }
    ],
    "job_type": "ping",
    "regions": [
      "lga",
      "sjc"
    ],
    "active": true,
    "frequency": 60,
    "policy": "quorum",
    "region_scope": "fixed"
  }
]`)
	mjl := []*Job{}
	if err := json.Unmarshal(data, &mjl); err != nil {
		t.Error(err)
	}
	if len(mjl) != 1 {
		fmt.Println(mjl)
		t.Error("Do not have any jobs")
	}
	j := mjl[0]
	if j.ID != "52a27d4397d5f07003fdbe7b" {
		t.Error("Wrong ID")
	}
	conf := j.Config
	if conf["host"] != "1.2.3.4" {
		t.Error("Wrong host")
	}
	status := j.Status["global"]
	if status.Since != 1389407609 {
		t.Error("since has unexpected value")
	}
	if status.Status != "up" {
		t.Error("Status is not up")
	}
	r := j.Rules[0]
	assert.Equal(t, r.Key, "rtt", "RTT rule key is wrong")
	assert.Equal(t, r.Value.(float64), float64(100), "RTT rule value is wrong")
	if r.Comparison != "<" {
		t.Error("RTT rule comparison is wrong")
	}
	if j.Type != "ping" {
		t.Error("Jobtype is wrong")
	}
	if j.Regions[0] != "lga" {
		t.Error("First region is not lga")
	}
	if !j.Active {
		t.Error("Job is not active")
	}
	if j.Frequency != 60 {
		t.Error("Job frequency != 60")
	}
	if j.Policy != "quorum" {
		t.Error("Job policy is not quorum")
	}
	if j.RegionScope != "fixed" {
		t.Error("Job region scope is not fixed")
	}
}

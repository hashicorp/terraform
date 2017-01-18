package rundeck

import (
	"encoding/xml"
	"testing"
)

type marshalTest struct {
	Name        string
	Input       interface{}
	ExpectedXML string
}

type unmarshalTestFunc func(result interface{}) error

type unmarshalTest struct {
	Name     string
	Input    string
	Output   interface{}
	TestFunc unmarshalTestFunc
}

func testMarshalXML(t *testing.T, tests []marshalTest) {
	for _, test := range tests {
		xmlBytes, err := xml.Marshal(test.Input)
		if err != nil {
			t.Errorf("Error in Marshall for test %s: %s", test.Name, err.Error())
			continue
		}
		xmlStr := string(xmlBytes)
		if xmlStr != test.ExpectedXML {
			t.Errorf("Test %s got %s, but wanted %s", test.Name, xmlStr, test.ExpectedXML)
			continue
		}
	}
}

func testUnmarshalXML(t *testing.T, tests []unmarshalTest) {
	for _, test := range tests {
		xml.Unmarshal([]byte(test.Input), test.Output)
		err := test.TestFunc(test.Output)
		if err != nil {
			t.Errorf("Test %s %s", test.Name, err.Error())
		}
	}
}

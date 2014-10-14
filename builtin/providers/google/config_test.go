package google

import (
	"reflect"
	"testing"
)

func TestConfigLoadJSON_account(t *testing.T) {
	var actual accountFile
	if err := loadJSON(&actual, "./test-fixtures/fake_account.json"); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := accountFile{
		PrivateKeyId: "foo",
		PrivateKey:   "bar",
		ClientEmail:  "foo@bar.com",
		ClientId:     "id@foo.com",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

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

func TestConfigLoadJSON_client(t *testing.T) {
	var actual clientSecretsFile
	if err := loadJSON(&actual, "./test-fixtures/fake_client.json"); err != nil {
		t.Fatalf("err: %s", err)
	}

	var expected clientSecretsFile
	expected.Web.AuthURI = "https://accounts.google.com/o/oauth2/auth"
	expected.Web.ClientEmail = "foo@developer.gserviceaccount.com"
	expected.Web.ClientId = "foo.apps.googleusercontent.com"
	expected.Web.TokenURI = "https://accounts.google.com/o/oauth2/token"

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

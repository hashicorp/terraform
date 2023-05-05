// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// verify that we can locate public key data
func TestFindKeyData(t *testing.T) {
	// set up a test directory
	td := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	id := "provisioner_id"

	pub := generateSSHKey(t, id)
	pubData := pub.Marshal()

	// backup the pub file, and replace it with a broken file to ensure we
	// extract the public key from the private key.
	if err := os.Rename(id+".pub", "saved.pub"); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(id+".pub", []byte("not a public key"), 0600); err != nil {
		t.Fatal(err)
	}

	foundData := findIDPublicKey(id)
	if !bytes.Equal(foundData, pubData) {
		t.Fatalf("public key %q does not match", foundData)
	}

	// move the pub file back, and break the private key file to simulate an
	// encrypted private key
	if err := os.Rename("saved.pub", id+".pub"); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(id, []byte("encrypted private key"), 0600); err != nil {
		t.Fatal(err)
	}

	foundData = findIDPublicKey(id)
	if !bytes.Equal(foundData, pubData) {
		t.Fatalf("public key %q does not match", foundData)
	}

	// check the file by path too
	foundData = findIDPublicKey(filepath.Join(".", id))
	if !bytes.Equal(foundData, pubData) {
		t.Fatalf("public key %q does not match", foundData)
	}
}

func generateSSHKey(t *testing.T, idFile string) ssh.PublicKey {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	privFile, err := os.OpenFile(idFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer privFile.Close()
	privPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	if err := pem.Encode(privFile, privPEM); err != nil {
		t.Fatal(err)
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(idFile+".pub", ssh.MarshalAuthorizedKey(pub), 0600)
	if err != nil {
		t.Fatal(err)
	}

	return pub
}

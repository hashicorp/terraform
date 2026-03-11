// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providercache

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

type providerPkg struct {
	zipContent    []byte
	shaSumContent []byte
	sigContent    []byte
	key           *bytes.Buffer
	keyID         string
	filePrefix    string // <provider_name>_<version>
}

// createTestProvider creates a test provider package with required verification files
func createTestProvider(t *testing.T, providerName, version string) *providerPkg {
	entity, pub := generateGPGKey(t)
	dir := t.TempDir()
	fileName := fmt.Sprintf("%s_%s", providerName, version)

	// Create a zip file for the provider
	zipFile := filepath.Join(dir, fileName)
	zipContent := createProviderZip(t, zipFile, fileName)

	// Generate SHA256SUMS file
	sumFile, sumContent, err := createSHASums(dir, fileName)
	if err != nil {
		t.Fatalf("failed to create SHA sums: %s", err)
	}

	// Sign the SHA256SUMS file
	sigContent, err := createDetachedSignature(sumFile, entity)
	if err != nil {
		t.Fatalf("failed to create signature: %s", err)
	}

	return &providerPkg{
		zipContent:    zipContent,
		shaSumContent: sumContent,
		sigContent:    sigContent,
		key:           pub,
		keyID:         entity.PrimaryKey.KeyIdString(),
		filePrefix:    fileName,
	}
}

func generateGPGKey(t *testing.T) (*openpgp.Entity, *bytes.Buffer) {
	entity, err := openpgp.NewEntity("Terraform Test", "test", "terraform@example.com", nil)
	if err != nil {
		t.Fatalf("failed to create entity: %s", err)
	}

	// Export the public key in armored format
	pubBuf := bytes.NewBuffer(nil)
	w, err := armor.Encode(pubBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("failed to create armor writer: %s", err)
	}

	if err := entity.Serialize(w); err != nil {
		t.Fatalf("failed to serialize key: %s", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to finalize armor: %s", err)
	}

	return entity, pubBuf
}

func createProviderZip(t *testing.T, zipPath, filename string) []byte {
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// Add file to the archive
	_, err := zipWriter.Create(fmt.Sprintf("terraform-provider-%s", filename))
	if err != nil {
		t.Fatalf("failed to create zip entry: %s", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %s", err)
	}

	// Write the zip to a file
	zipContent := zipBuffer.Bytes()
	if err := os.WriteFile(zipPath+".zip", zipContent, 0666); err != nil {
		t.Fatalf("failed to write zip file: %s", err)
	}

	return zipContent
}

// createDetachedSignature creates a PGP detached signature for the given file
func createDetachedSignature(filePath string, entity *openpgp.Entity) ([]byte, error) {
	r, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer safeClose(r)

	sigPath := filePath + ".sig"
	w, err := os.Create(sigPath)
	if err != nil {
		return nil, err
	}
	defer safeClose(w)

	if err := openpgp.DetachSign(w, entity, r, nil); err != nil {
		return nil, err
	}

	// Read the signature back
	return os.ReadFile(sigPath)
}

// createSHASums creates SHA256 sums for all files in the directory
func createSHASums(dir, name string) (string, []byte, error) {
	var sums []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		sum, err := calculateFileHash(filePath)
		if err != nil {
			return "", nil, err
		}

		sums = append(sums, fmt.Sprintf("%x  %s", sum, file.Name()))
	}

	sumContent := []byte(strings.Join(sums, "\n") + "\n")
	sumPath := filepath.Join(dir, fmt.Sprintf("%s_SHA256SUMS", name))

	if err := os.WriteFile(sumPath, sumContent, 0666); err != nil {
		return "", nil, err
	}

	return sumPath, sumContent, nil
}

// calculateFileHash returns the SHA256 hash of a file
func calculateFileHash(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer safeClose(f)

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// safeClose safely closes a file, logging any errors
func safeClose(f *os.File) {
	if err := f.Close(); err != nil {
		fmt.Printf("failed to close file: %s\n", err)
	}
}

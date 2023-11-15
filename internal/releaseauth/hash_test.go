// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package releaseauth

import (
	"strings"
	"testing"
)

func Test_ParseChecksums(t *testing.T) {
	sample := `bb611bb4c082fec9943d3c315fcd7cacd7dabce43fbf79b8e6b451bb4e54096d  terraform-cloudplugin_0.1.0-prototype_darwin_amd64.zip
295bf15c2af01d18ce7832d6d357667119e4b14eb8fd2454d506b23ed7825652  terraform-cloudplugin_0.1.0-prototype_darwin_arm64.zip
b0744b9c8c0eb7ea61824728c302d0fd4fda4bb841fb6b3e701ef9eb10adbc39  terraform-cloudplugin_0.1.0-prototype_freebsd_386.zip
8fc967d1402c5106fb0ca1b084b7edd2b11fd8d7c2225f5cd05584a56e0b2a16  terraform-cloudplugin_0.1.0-prototype_freebsd_amd64.zip
2f35b2fc748b6f279b067a4eefd65264f811a2ae86a969461851dae546aa402d  terraform-cloudplugin_0.1.0-prototype_linux_386.zip
c877c8cebf76209c2c7d427d31e328212cd4716fdd8b6677939fd2a01e06a2d0  terraform-cloudplugin_0.1.0-prototype_linux_amd64.zip
97ff8fe4e2e853c9ea54605305732e5b16437045230a2df21f410e36edcfe7bd  terraform-cloudplugin_0.1.0-prototype_linux_arm.zip
d415a1c39b9ec79bd00efe72d0bf14e557833b6c1ce9898f223a7dd22abd0241  terraform-cloudplugin_0.1.0-prototype_linux_arm64.zip
0f33a13eca612d1b3cda959d655a1535d69bcc1195dee37407c667c12c4900b5  terraform-cloudplugin_0.1.0-prototype_solaris_amd64.zip
a6d572e5064e1b1cf8b0b4e64bc058dc630313c95e975b44e0540f231655d31c  terraform-cloudplugin_0.1.0-prototype_windows_386.zip
2aaceed12ebdf25d21f9953a09c328bd8892f5a5bd5382bd502f054478f56998  terraform-cloudplugin_0.1.0-prototype_windows_amd64.zip
`

	sums, err := ParseChecksums([]byte(sample))
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}

	expectedSum, err := SHA256FromHex("2f35b2fc748b6f279b067a4eefd65264f811a2ae86a969461851dae546aa402d")
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}

	if found := sums["terraform-cloudplugin_0.1.0-prototype_linux_386.zip"]; found != expectedSum {
		t.Errorf("Expected %q, got %q", expectedSum, found)
	}
}

func Test_ParseChecksums_Empty(t *testing.T) {
	sample := `
`

	sums, err := ParseChecksums([]byte(sample))
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}

	expectedSum, err := SHA256FromHex("2f35b2fc748b6f279b067a4eefd65264f811a2ae86a969461851dae546aa402d")
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}

	err = sums.Validate("terraform-cloudplugin_0.1.0-prototype_linux_arm.zip", expectedSum)
	if err == nil || !strings.Contains(err.Error(), "no checksum found for filename") {
		t.Errorf("Expected error %q, got nil", "no checksum found for filename")
	}
}

func Test_ParseChecksums_BadFormat(t *testing.T) {
	sample := `xxxxxxxxxxxxxxxxxxxxxx  terraform-cloudplugin_0.1.0-prototype_darwin_amd64.zip
	zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz  terraform-cloudplugin_0.1.0-prototype_darwin_arm64.zip
`

	_, err := ParseChecksums([]byte(sample))
	if err == nil || !strings.Contains(err.Error(), "failed to parse checksums") {
		t.Fatalf("Expected error %q, got: %s", "failed to parse checksums", err)
	}
}

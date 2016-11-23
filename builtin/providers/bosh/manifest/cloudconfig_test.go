package manifest_test

import (
	. "github.com/hashicorp/terraform/builtin/providers/bosh/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudConfigManifest Parsing Tests", func() {
	Describe("Test parsing CloudConfig manifest", func() {
		Describe("Test parsing AWS CloudCondig manifest", func() {
			b := []byte(cloudconfig)
			ccm := NewCloudConfigManifest(b)
			It("it should parse AZs", func() {
				Expect(ccm.AZs[0].Name).Should(Equal("z1"))
				Expect(ccm.AZs[0].CloudProperties["availability_zone"]).Should(Equal("us-east-1a"))
				Expect(ccm.AZs[1].Name).Should(Equal("z2"))
				Expect(ccm.AZs[1].CloudProperties["availability_zone"]).Should(Equal("us-east-1b"))
			})
			It("it should parse VM types section", func() {
				Expect(ccm.VMTypes[0].Name).Should(Equal("default"))
				Expect(ccm.VMTypes[0].CloudProperties["instance_type"]).Should(Equal("t2.micro"))
				Expect(ccm.VMTypes[0].CloudProperties["ephemeral_disk"].(CloudProperties)["size"]).Should(Equal(3000))
				Expect(ccm.VMTypes[0].CloudProperties["ephemeral_disk"].(CloudProperties)["type"]).Should(Equal("gp2"))
				Expect(ccm.VMTypes[1].Name).Should(Equal("large"))
				Expect(ccm.VMTypes[1].CloudProperties["instance_type"]).Should(Equal("m3.large"))
				Expect(ccm.VMTypes[1].CloudProperties["ephemeral_disk"].(CloudProperties)["size"]).Should(Equal(30000))
				Expect(ccm.VMTypes[1].CloudProperties["ephemeral_disk"].(CloudProperties)["type"]).Should(Equal("gp2"))
			})
			It("it should parse Disk types section", func() {
				Expect(ccm.DiskTypes[0].Name).Should(Equal("default"))
				Expect(ccm.DiskTypes[0].DiskSize).Should(Equal(3000))
				Expect(ccm.DiskTypes[0].CloudProperties["type"]).Should(Equal("gp2"))
				Expect(ccm.DiskTypes[1].Name).Should(Equal("large"))
				Expect(ccm.DiskTypes[1].DiskSize).Should(Equal(50000))
				Expect(ccm.DiskTypes[1].CloudProperties["type"]).Should(Equal("gp2"))
			})
			It("it should parse networks section", func() {
				Expect(ccm.Networks[0].Name).Should(Equal("default"))
				Expect(ccm.Networks[0].Type).Should(Equal("manual"))
				Expect(ccm.Networks[0].Subnets[0].Range).Should(Equal("10.10.0.0/24"))
				Expect(ccm.Networks[0].Subnets[0].Gateway).Should(Equal("10.10.0.1"))
				Expect(ccm.Networks[0].Subnets[0].AZ).Should(Equal("z1"))
				Expect(ccm.Networks[0].Subnets[0].DNS[0]).Should(Equal("10.10.0.2"))
				Expect(ccm.Networks[0].Subnets[0].CloudProperties["subnet"]).Should(Equal("subnet-f2744a86"))
				Expect(ccm.Networks[0].Subnets[1].Range).Should(Equal("10.10.64.0/24"))
				Expect(ccm.Networks[0].Subnets[1].Gateway).Should(Equal("10.10.64.1"))
				Expect(ccm.Networks[0].Subnets[1].AZ).Should(Equal("z2"))
				Expect(ccm.Networks[0].Subnets[1].DNS[0]).Should(Equal("10.10.0.2"))
				Expect(ccm.Networks[0].Subnets[1].Static[0]).Should(Equal("10.10.64.121"))
				Expect(ccm.Networks[0].Subnets[1].Static[1]).Should(Equal("10.10.64.122"))
				Expect(ccm.Networks[0].Subnets[1].CloudProperties["subnet"]).Should(Equal("subnet-eb8bd3ad"))
				Expect(ccm.Networks[1].Name).Should(Equal("vip"))
				Expect(ccm.Networks[1].Type).Should(Equal("vip"))
			})
			It("it should parse compilation section", func() {
				Expect(ccm.Compilation.Workers).Should(Equal(5))
				Expect(ccm.Compilation.AZ).Should(Equal("z1"))
				Expect(ccm.Compilation.VMType).Should(Equal("large"))
				Expect(ccm.Compilation.Network).Should(Equal("default"))
				Expect(ccm.Compilation.ReuseCompilationVMs).Should(Equal(true))
			})
		})
	})
})

const cloudconfig = `
azs:
- name: z1
  cloud_properties: {availability_zone: us-east-1a}
- name: z2
  cloud_properties: {availability_zone: us-east-1b}

vm_types:
- name: default
  cloud_properties:
    instance_type: t2.micro
    ephemeral_disk: {size: 3000, type: gp2}
- name: large
  cloud_properties:
    instance_type: m3.large
    ephemeral_disk: {size: 30000, type: gp2}

disk_types:
- name: default
  disk_size: 3000
  cloud_properties: {type: gp2}
- name: large
  disk_size: 50_000
  cloud_properties: {type: gp2}

networks:
- name: default
  type: manual
  subnets:
  - range: 10.10.0.0/24
    gateway: 10.10.0.1
    az: z1
    static: [10.10.0.62]
    dns: [10.10.0.2]
    cloud_properties: {subnet: subnet-f2744a86}
  - range: 10.10.64.0/24
    gateway: 10.10.64.1
    az: z2
    static: [10.10.64.121, 10.10.64.122]
    dns: [10.10.0.2]
    cloud_properties: {subnet: subnet-eb8bd3ad}
- name: vip
  type: vip

compilation:
  workers: 5
  reuse_compilation_vms: true
  az: z1
  vm_type: large
  network: default
`

# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

$script = <<SCRIPT
# Install Go and prerequisites
apt-get -qq update
apt-get -qq install build-essential curl git-core libpcre3-dev mercurial pkg-config zip
hg clone -u release https://code.google.com/p/go /opt/go
cd /opt/go/src && ./all.bash

# Setup the GOPATH
mkdir -p /opt/gopath
cat <<EOF >/etc/profile.d/gopath.sh
export GOPATH="/opt/gopath"
export PATH="/opt/go/bin:\$GOPATH/bin:\$PATH"
EOF

# Make sure the GOPATH is usable by vagrant
chown -R vagrant:vagrant /opt/go
chown -R vagrant:vagrant /opt/gopath
SCRIPT

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "chef/ubuntu-12.04"

  config.vm.provision "shell", inline: $script

  ["vmware_fusion", "vmware_workstation"].each do |p|
    config.vm.provider "p" do |v|
      v.vmx["memsize"] = "2048"
      v.vmx["numvcpus"] = "2"
      v.vmx["cpuid.coresPerSocket"] = "1"
    end
  end
end

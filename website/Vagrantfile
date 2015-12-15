# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

$script = <<SCRIPT
sudo apt-get -y update

# RVM/Ruby
sudo apt-get -y install curl
# manually install GPG key in a proxy-friendly way
curl -sSL https://rvm.io/mpapis.asc | gpg --import -
curl -sSL https://get.rvm.io | bash -s stable
. ~/.bashrc
. ~/.bash_profile
rvm install 2.0.0
rvm --default use 2.0.0

# Middleman deps
cd /vagrant
gem install bundle
sudo apt-get install -y git-core
bundle

# Run the middleman server
bundle exec middleman server &
SCRIPT

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "bento/ubuntu-12.04"
  config.vm.network "private_network", ip: "33.33.30.10"
  config.vm.provision "shell", inline: $script, privileged: false
  config.vm.synced_folder ".", "/vagrant", type: "rsync"
  config.vm.network "forwarded_port", guest: 4567, host: 4567
end

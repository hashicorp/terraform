#!/bin/bash

pushd $(dirname $0)

base_url=http://dl.bintray.com/dougm/ttylinux
ttylinux="ttylinux-pc_i486-16.1"
files="${ttylinux}.iso ${ttylinux}-live.ova ${ttylinux}.ova"

for name in $files ; do
  wget -N $base_url/$name
done

wget -N https://github.com/icebreaker/floppybird/raw/master/build/floppybird.img

# extract ova so we can also use the .vmdk and .ovf files directly
tar -xvf ${ttylinux}.ova

popd

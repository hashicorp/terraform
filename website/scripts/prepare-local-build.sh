#!/bin/bash

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd $scriptdir/..

# transform regular anchor links to link_to helpers
find source -name *.erb -exec \
sed -r -i -e 's/<a href="\/([^"]+)">([^<]+)<\/a>/<%= link_to "\2", "\/\1" %>/' {} \;

# transform anchor links with class attributes to link_to helpers
find source -name *.erb -exec \
sed -r -i -e \
's/<a class="([^"]+)" href="\/([^"]+)">([^<]+)<\/a>/<%= link_to "\3", "\/\2", :class => "\1" %>/' {} \;

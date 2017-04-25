#petname

##NAME []()

**petname** − a utility to generate "pet names", consisting of a random combination of adverbs, an adjective, and an animal name

##SYNOPSIS []()

**petname** \[-w|--words INT\] \[-l|--letters INT\] \[-s|--separator STR\] \[-d|--dir STR\] \[-c|--complexity INT\] \[-u|--ubuntu\]

##OPTIONS []()
- -w|--words number of words in the name, default is 2
- -l|--letters maximum number of letters in each word, default is unlimited
- -s|--separator string used to separate name words, default is ’-’
- -d|--dir directory containing adverbs.txt, adjectives.txt, names.txt, default is */usr/share/petname/*
- -c|--complexity \[0, 1, 2\]; 0 = easy words, 1 = standard words, 2 = complex words, default=1
- -u|--ubuntu generate ubuntu-style names, alliteration of first character of each word

##DESCRIPTION []()

This utility will generate "pet names", consisting of a random combination of an adverb, adjective, and an animal name. These are useful for unique hostnames or container names, for instance.

As such, PetName tries to follow the tenets of Zooko’s triangle. Names are:

- human meaningful
- decentralized
- secure

##EXAMPLES []()

```
$ petname
wiggly-yellowtail

$ petname --words 1
robin

$ petname --words 3
primly-lasting-toucan

$ petname --words 4
angrily-impatiently-sage-longhorn

$ petname --separator ":"
cool:gobbler

$ petname --separator "" --words 3
comparablyheartylionfish

$ petname --ubuntu
amazed-asp

$ petname --complexity 0
massive-colt
```

##CODE []()

Besides this shell utility, there are also native libraries: python-petname, python3-petname, and golang-petname. Here are some programmatic examples in code:

**Golang Example**
```golang
package main

import (
	"flag"
	"fmt"
	"github.com/dustinkirkland/golang-petname"
)

var (
	words = flag.Int("words", 2, "The number of words in the pet name")
	separator = flag.String("separator", "-", "The separator between words in the pet name")
)

func main() {
	flag.Parse()
	fmt.Println(petname.Generate(\*words, \*separator))
}
```

**Python Example**
See: https://pypi.golang.org/pypi/petname

$ pip install petname
$ sudo apt-get install golang-petname

```python
#!/usr/bin/python
import argparse
import petname

parser = argparse.ArgumentParser(description="Generate human readable random names")
parser.add_argument("-w", "--words", help="Number of words in name, default=2", default=2)
parser.add_argument("-s", "--separator", help="Separator between words, default='-'", default="-")
parser.options = parser.parse_args()

print petname.Generate(int(parser.options.words), parser.options.separator)
```

##AUTHOR []()

This manpage and the utility were written by Dustin Kirkland &lt;dustin.kirkland@gmail.com&gt; for Ubuntu systems (but may be used by others). Permission is granted to copy, distribute and/or modify this document and the utility under the terms of the Apache2 License.

The complete text of the Apache2 License can be found in */usr/share/common-licenses/Apache-2.0* on Debian/Ubuntu systems.

------------------------------------------------------------------------

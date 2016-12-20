Inflector
=========

Inflector pluralizes and singularizes English nouns.

[![Build Status](https://travis-ci.org/gedex/inflector.png?branch=master)](https://travis-ci.org/gedex/inflector)
[![Coverage Status](https://coveralls.io/repos/gedex/inflector/badge.png?branch=master)](https://coveralls.io/r/gedex/inflector?branch=master)
[![GoDoc](https://godoc.org/github.com/gedex/inflector?status.svg)](https://godoc.org/github.com/gedex/inflector)

## Basic Usage

There are only two exported functions: `Pluralize` and `Singularize`.

~~~go
fmt.Println(inflector.Singularize("People")) // will print "Person"
fmt.Println(inflector.Pluralize("octopus")) // will print "octopuses"
~~~

## Credits

* [CakePHP's Inflector](https://github.com/cakephp/cakephp/blob/master/lib/Cake/Utility/Inflector.php)

## License

This library is distributed under the BSD-style license found in the LICENSE.md file.

# go-dotenv

Go parsing library for the dotenv format.

There is no formal definition of the dotenv format but it has been introduced
by https://github.com/bkeepers/dotenv which is thus canonical. This library is a port of that.

This library was developed specifically for [direnv](https://direnv.net).

## Features

* `k=v` format
* bash `export k=v` format
* yaml `k: v` format
* comments

## Missing

* support for variable expansion, probably needs API breakage

## Alternatives

Some other good alternatives with various variations.

* https://github.com/joho/godotenv
* https://github.com/lazureykis/dotenv
* https://github.com/subosito/gotenv


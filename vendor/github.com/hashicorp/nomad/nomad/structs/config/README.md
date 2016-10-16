# Overview

`nomad/structs/config` is a package for configuration `struct`s that are
shared among packages that needs the same `struct` definitions, but can't
import each other without creating a cyle.  This `config` package must be
terminal in the import graph (or very close to terminal in the dependency
graph).

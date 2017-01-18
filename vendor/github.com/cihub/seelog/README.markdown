Seelog
=======

Seelog is a powerful and easy-to-learn logging framework that provides functionality for flexible dispatching, filtering, and formatting log messages.
It is natively written in the [Go](http://golang.org/) programming language. 

[![Build Status](https://drone.io/github.com/cihub/seelog/status.png)](https://drone.io/github.com/cihub/seelog/latest)

Features
------------------

* Xml configuring to be able to change logger parameters without recompilation
* Changing configurations on the fly without app restart
* Possibility to set different log configurations for different project files and functions
* Adjustable message formatting
* Simultaneous log output to multiple streams
* Choosing logger priority strategy to minimize performance hit
* Different output writers
  * Console writer
  * File writer 
  * Buffered writer (Chunk writer)
  * Rolling log writer (Logging with rotation)
  * SMTP writer
  * Others... (See [Wiki](https://github.com/cihub/seelog/wiki))
* Log message wrappers (JSON, XML, etc.)
* Global variables and functions for easy usage in standalone apps
* Functions for flexible usage in libraries

Quick-start
-----------

```go
package main

import log "github.com/cihub/seelog"

func main() {
    defer log.Flush()
    log.Info("Hello from Seelog!")
}
```

Installation
------------

If you don't have the Go development environment installed, visit the 
[Getting Started](http://golang.org/doc/install.html) document and follow the instructions. Once you're ready, execute the following command:

```
go get -u github.com/cihub/seelog
```

*IMPORTANT*: If you are not using the latest release version of Go, check out this [wiki page](https://github.com/cihub/seelog/wiki/Notes-on-'go-get')

Documentation
---------------

Seelog has github wiki pages, which contain detailed how-tos references: https://github.com/cihub/seelog/wiki

Examples
---------------

Seelog examples can be found here: [seelog-examples](https://github.com/cihub/seelog-examples)

Issues
---------------

Feel free to push issues that could make Seelog better: https://github.com/cihub/seelog/issues

Changelog
---------------
* **v2.6** : Config using code and custom formatters
    * Configuration using code in addition to xml (All internal receiver/dispatcher/logger types are now exported).
    * Custom formatters. Check [wiki](https://github.com/cihub/seelog/wiki/Custom-formatters)
    * Bugfixes and internal improvements.
* **v2.5** : Interaction with other systems. Part 2: custom receivers
    * Finished custom receivers feature. Check [wiki](https://github.com/cihub/seelog/wiki/custom-receivers)
    * Added 'LoggerFromCustomReceiver'
    * Added 'LoggerFromWriterWithMinLevelAndFormat'
    * Added 'LoggerFromCustomReceiver'
    * Added 'LoggerFromParamConfigAs...' 
* **v2.4** : Interaction with other systems. Part 1: wrapping seelog
    * Added configurable caller stack skip logic
    * Added 'SetAdditionalStackDepth' to 'LoggerInterface'
* **v2.3** : Rethinking 'rolling' receiver
    * Reimplemented 'rolling' receiver
    * Added 'Max rolls' feature for 'rolling' receiver with type='date'
    * Fixed 'rolling' receiver issue: renaming on Windows
* **v2.2** : go1.0 compatibility point [go1.0 tag]
    * Fixed internal bugs
    * Added 'ANSI n [;k]' format identifier:  %EscN
    * Made current release go1 compatible 
* **v2.1** : Some new features
    * Rolling receiver archiving option.
    * Added format identifier: %Line
    * Smtp: added paths to PEM files directories
    * Added format identifier: %FuncShort
    * Warn, Error and Critical methods now return an error
* **v2.0** : Second major release. BREAKING CHANGES.
    * Support of binaries with stripped symbols
    * Added log strategy: adaptive
    * Critical message now forces Flush()
    * Added predefined formats: xml-debug, xml-debug-short, xml, xml-short, json-debug, json-debug-short, json, json-short, debug, debug-short, fast
    * Added receiver: conn (network connection writer)
    * BREAKING CHANGE: added Tracef, Debugf, Infof, etc. to satisfy the print/printf principle
    * Bug fixes
* **v1.0** : Initial release. Features:
    * Xml config
    * Changing configurations on the fly without app restart
    * Contraints and exceptions
    * Formatting
    * Log strategies: sync, async loop, async timer
    * Receivers: buffered, console, file, rolling, smtp




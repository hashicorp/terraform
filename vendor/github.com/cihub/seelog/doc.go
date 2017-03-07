// Copyright (c) 2014 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

/*
Package seelog implements logging functionality with flexible dispatching, filtering, and formatting.

Creation

To create a logger, use one of the following constructors:
  func LoggerFromConfigAsBytes
  func LoggerFromConfigAsFile
  func LoggerFromConfigAsString
  func LoggerFromWriterWithMinLevel
  func LoggerFromWriterWithMinLevelAndFormat
  func LoggerFromCustomReceiver (check https://github.com/cihub/seelog/wiki/Custom-receivers)
Example:
  import log "github.com/cihub/seelog"

  func main() {
      logger, err := log.LoggerFromConfigAsFile("seelog.xml")
      if err != nil {
          panic(err)
      }
      defer logger.Flush()
      ... use logger ...
  }
The "defer" line is important because if you are using asynchronous logger behavior, without this line you may end up losing some
messages when you close your application because they are processed in another non-blocking goroutine. To avoid that you
explicitly defer flushing all messages before closing.

Usage

Logger created using one of the LoggerFrom* funcs can be used directly by calling one of the main log funcs.
Example:
  import log "github.com/cihub/seelog"

  func main() {
      logger, err := log.LoggerFromConfigAsFile("seelog.xml")
      if err != nil {
          panic(err)
      }
      defer logger.Flush()
      logger.Trace("test")
      logger.Debugf("var = %s", "abc")
  }

Having loggers as variables is convenient if you are writing your own package with internal logging or if you have
several loggers with different options.
But for most standalone apps it is more convenient to use package level funcs and vars. There is a package level
var 'Current' made for it. You can replace it with another logger using 'ReplaceLogger' and then use package level funcs:
  import log "github.com/cihub/seelog"

  func main() {
      logger, err := log.LoggerFromConfigAsFile("seelog.xml")
      if err != nil {
          panic(err)
      }
      log.ReplaceLogger(logger)
      defer log.Flush()
      log.Trace("test")
      log.Debugf("var = %s", "abc")
  }
Last lines
      log.Trace("test")
      log.Debugf("var = %s", "abc")
do the same as
      log.Current.Trace("test")
      log.Current.Debugf("var = %s", "abc")
In this example the 'Current' logger was replaced using a 'ReplaceLogger' call and became equal to 'logger' variable created from config.
This way you are able to use package level funcs instead of passing the logger variable.

Configuration

Main seelog point is to configure logger via config files and not the code.
The configuration is read by LoggerFrom* funcs. These funcs read xml configuration from different sources and try
to create a logger using it.

All the configuration features are covered in detail in the official wiki: https://github.com/cihub/seelog/wiki.
There are many sections covering different aspects of seelog, but the most important for understanding configs are:
    https://github.com/cihub/seelog/wiki/Constraints-and-exceptions
    https://github.com/cihub/seelog/wiki/Dispatchers-and-receivers
    https://github.com/cihub/seelog/wiki/Formatting
    https://github.com/cihub/seelog/wiki/Logger-types
After you understand these concepts, check the 'Reference' section on the main wiki page to get the up-to-date
list of dispatchers, receivers, formats, and logger types.

Here is an example config with all these features:
    <seelog type="adaptive" mininterval="2000000" maxinterval="100000000" critmsgcount="500" minlevel="debug">
        <exceptions>
            <exception filepattern="test*" minlevel="error"/>
        </exceptions>
        <outputs formatid="all">
            <file path="all.log"/>
            <filter levels="info">
              <console formatid="fmtinfo"/>
            </filter>
            <filter levels="error,critical" formatid="fmterror">
              <console/>
              <file path="errors.log"/>
            </filter>
        </outputs>
        <formats>
            <format id="fmtinfo" format="[%Level] [%Time] %Msg%n"/>
            <format id="fmterror" format="[%LEVEL] [%Time] [%FuncShort @ %File.%Line] %Msg%n"/>
            <format id="all" format="[%Level] [%Time] [@ %File.%Line] %Msg%n"/>
            <format id="criticalemail" format="Critical error on our server!\n    %Time %Date %RelFile %Func %Msg \nSent by Seelog"/>
        </formats>
    </seelog>
This config represents a logger with adaptive timeout between log messages (check logger types reference) which
logs to console, all.log, and errors.log depending on the log level. Its output formats also depend on log level. This logger will only
use log level 'debug' and higher (minlevel is set) for all files with names that don't start with 'test'. For files starting with 'test'
this logger prohibits all levels below 'error'.

Configuration using code

Although configuration using code is not recommended, it is sometimes needed and it is possible to do with seelog. Basically, what
you need to do to get started is to create constraints, exceptions and a dispatcher tree (same as with config). Most of the New*
functions in this package are used to provide such capabilities.

Here is an example of configuration in code, that demonstrates an async loop logger that logs to a simple split dispatcher with
a console receiver using a specified format and is filtered using a top-level min-max constraints and one expection for
the 'main.go' file. So, this is basically a demonstration of configuration of most of the features:

  package main

  import log "github.com/cihub/seelog"

  func main() {
      defer log.Flush()
      log.Info("Hello from Seelog!")

      consoleWriter, _ := log.NewConsoleWriter()
      formatter, _ := log.NewFormatter("%Level %Msg %File%n")
      root, _ := log.NewSplitDispatcher(formatter, []interface{}{consoleWriter})
      constraints, _ := log.NewMinMaxConstraints(log.TraceLvl, log.CriticalLvl)
      specificConstraints, _ := log.NewListConstraints([]log.LogLevel{log.InfoLvl, log.ErrorLvl})
      ex, _ := log.NewLogLevelException("*", "*main.go", specificConstraints)
      exceptions := []*log.LogLevelException{ex}

      logger := log.NewAsyncLoopLogger(log.NewLoggerConfig(constraints, exceptions, root))
      log.ReplaceLogger(logger)

      log.Trace("This should not be seen")
      log.Debug("This should not be seen")
      log.Info("Test")
      log.Error("Test2")
  }

Examples

To learn seelog features faster you should check the examples package: https://github.com/cihub/seelog-examples
It contains many example configs and usecases.
*/
package seelog

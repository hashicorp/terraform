---
layout: "docs"
page_title: "Drivers: Java"
sidebar_current: "docs-drivers-java"
description: |-
  The Java task driver is used to run Jars using the JVM.
---

# Java Driver

Name: `java`

The `Java` driver is used to execute Java applications packaged into a Java Jar
file. The driver requires the Jar file to be accessible from the Nomad
client via the [`artifact` downloader](/docs/jobspec/index.html#artifact_doc). 

## Task Configuration

The `java` driver supports the following configuration in the job spec:

* `jar_path` - The path to the downloaded Jar. In most cases this will just be
  the name of the Jar. However, if the supplied artifact is an archive that
  contains the Jar in a subfolder, the path will need to be the relative path
  (`subdir/from_archive/my.jar`).

*   `args` - (Optional) A list of arguments to the optional `command`.
    References to environment variables or any [interpretable Nomad
    variables](/docs/jobspec/interpreted.html) will be interpreted
    before launching the task. For example:

    ```
        args = ["${nomad.datacenter}", "${MY_ENV}", "${meta.foo}"]
    ```

* `jvm_options` - (Optional) A list of JVM options to be passed while invoking
  java. These options are passed not validated in any way in Nomad.

## Examples

A simple config block to run a Java Jar:

```
task "web" {
  driver = "java"

  config {
    jar_path = "local/hello.jar"
    jvm_options = ["-Xmx2048m", "-Xms256m"]
  }

  # Specifying an artifact is required with the "java"
  # driver. This is the # mechanism to ship the Jar to be run.
  artifact {
    source = "https://dl.dropboxusercontent.com/u/1234/hello.jar"

    options {
      checksum = "md5:123445555555555"
    }
  }
```

## Client Requirements

The `java` driver requires Java to be installed and in your systems `$PATH`. The
task must also specify at least one artifact to download as this is the only way
to retrieve the Jar being run.

## Client Attributes

The `java` driver will set the following client attributes:

* `driver.java` - Set to `1` if Java is found on the host node. Nomad determines
this by executing `java -version` on the host and parsing the output
* `driver.java.version` - Version of Java, ex: `1.6.0_65`
* `driver.java.runtime` - Runtime version, ex: `Java(TM) SE Runtime Environment (build 1.6.0_65-b14-466.1-11M4716)`
* `driver.java.vm` - Virtual Machine information, ex: `Java HotSpot(TM) 64-Bit Server VM (build 20.65-b04-466.1, mixed mode)`

## Resource Isolation

The resource isolation provided varies by the operating system of
the client and the configuration.

On Linux, Nomad will attempt to use cgroups, namespaces, and chroot
to isolate the resources of a process. If the Nomad agent is not
running as root many of these mechanisms cannot be used.

As a baseline, the Java jars will be run inside a Java Virtual Machine,
providing a minimum amount of isolation.


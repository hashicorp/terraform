---
layout: "http"
page_title: "HTTP API: /v1/client/fs"
sidebar_current: "docs-http-client-fs"
description: |-
  The '/v1/client/fs` endpoints are used to read the contents of an allocation
  directory.
---

# /v1/client/fs

The client `fs` endpoints are used to read the contents of files and
directories inside an allocation directory. The API endpoints are hosted by the
Nomad client and requests have to be made to the Client where the particular
allocation was placed.

## GET

<dl>
  <dt>Description</dt>
  <dd>
     Read contents of a file in an allocation directory.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/fs/cat/<Allocation-ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">path</span>
        <span class="param-flags">required</span>
         The path relative to the root of the allocation directory. It 
         defaults to `/`
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```
...
07:49 docker/3e8f0f4a67c2[924]: 1:M 22 Jun 21:07:49.110 # Server started, Redis version 3.2.1
07:49 docker/3e8f0f4a67c2[924]: 1:M 22 Jun 21:07:49.110 * The server is now ready to accept connections on port 6379
...
    ```

  </dd>

</dl>

<dl>
  <dt>Description</dt>
  <dd>
     Stream contents of a file in an allocation directory.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/fs/stream/<Allocation-ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">path</span>
        <span class="param-flags">required</span>
         The path relative to the root of the allocation directory. It 
         defaults to `/`
      </li>
      <li>
        <span class="param">offset</span>
         The offset to start streaming from. Defaults to 0.
      </li>
      <li>
        <span class="param">origin</span>
        Origin can be either "start" or "end" and applies the offset relative to
        either the start or end of the file respectively. Defaults to "start".
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```
...
    {
        "File":"alloc/logs/redis.stdout.0",
        "Offset":3604480
        "Data": "NTMxOTMyCjUzMTkzMwo1MzE5MzQKNTMx..."
    }
    {
        "File":"alloc/logs/redis.stdout.0",
        "FileEvent": "file deleted"
    }
    ```

  </dd>


  <dt>Field Reference</dt>
  <dd>
    The return value is a stream of frames. These frames contain the following
    fields:

    <ul>
      <li>
        <span class="param">Data</span>
        A base64 encoding of the bytes being streamed.
      </li>
      <li>
        <span class="param">FileEvent</span>
        An event that could cause a change in the streams position. The possible
        values are "file deleted" and "file truncated".
      </li>
      <li>
        <span class="param">Offset</span>
        Offset is the offset into the stream.
      </li>
      <li>
        <span class="param">File</span>
        The name of the file being streamed.
      </li>
    </ul>
  </dd>
</dl>

<a id="logs"></a>

<dl>
  <dt>Description</dt>
  <dd>
     Stream a tasks stdout/stderr logs.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/fs/logs/<Allocation-ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">task</span>
        <span class="param-flags">required</span>
        The name of the task inside the allocation to stream logs from.
      </li>
      <li>
        <span class="param">follow</span>
        <span class="param-flags">required</span>
         A boolean of whether to follow logs.
      </li>
      <li>
        <span class="param">type</span>
         Either, "stdout" or "stderr", defaults to "stdout" if omitted.
      </li>
      <li>
        <span class="param">offset</span>
         The offset to start streaming from. Defaults to 0.
      </li>
      <li>
        <span class="param">origin</span>
        Origin can be either "start" or "end" and applies the offset relative to
        either the start or end of the logs respectively. Defaults to "start".
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```
...
    {
        "File":"alloc/logs/redis.stdout.0",
        "Offset":3604480
        "Data": "NTMxOTMyCjUzMTkzMwo1MzE5MzQKNTMx..."
    }
    {
        "File":"alloc/logs/redis.stdout.0",
        "FileEvent": "file deleted"
    }
    ```

  </dd>


  <dt>Field Reference</dt>
  <dd>
    The return value is a stream of frames. These frames contain the following
    fields:

    <ul>
      <li>
        <span class="param">Data</span>
        A base64 encoding of the bytes being streamed.
      </li>
      <li>
        <span class="param">FileEvent</span>
        An event that could cause a change in the streams position. The possible
        values are "file deleted" and "file truncated".
      </li>
      <li>
        <span class="param">Offset</span>
        Offset is the offset into the stream.
      </li>
      <li>
        <span class="param">File</span>
        The name of the file being streamed.
      </li>
    </ul>
  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
     List files in an allocation directory.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/fs/ls/<Allocation-ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">path</span>
        <span class="param-flags">required</span>
        The path relative to the root of the allocation directory. It 
        defaults to `/`, the root of the allocation directory.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    [
      {
        "Name": "alloc",
        "IsDir": true,
        "Size": 4096,
        "FileMode": "drwxrwxr-x",
        "ModTime": "2016-03-15T15:40:00.414236712-07:00"
      },
      {
        "Name": "redis",
        "IsDir": true,
        "Size": 4096,
        "FileMode": "drwxrwxr-x",
        "ModTime": "2016-03-15T15:40:56.810238153-07:00"
      }
    ]
    ```

  </dd>
</dl>

<dl>
  <dt>Description</dt>
  <dd>
     Stat a file in an allocation directory.
  </dd>

  <dt>Method</dt>
  <dd>GET</dd>

  <dt>URL</dt>
  <dd>`/v1/client/fs/stat/<Allocation-ID>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">path</span>
        <span class="param-flags">required</span>
        The path of the file relative to the root of the allocation directory.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>

    ```javascript
    {
      "Name": "redis-syslog-collector.out",
      "IsDir": false,
      "Size": 96,
      "FileMode": "-rw-rw-r--",
      "ModTime": "2016-03-15T15:40:56.822238153-07:00"
    }
    ```

  </dd>
</dl>

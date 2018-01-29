**Do not run an instance of `froyg` that is open to the Internet unless you're absolutely sure that it is safe to do so. If you have any doubt, do not do it.**

## Overview

`froyg` automatically exposes **every object** in **every bucket** in **every region** that you have access to over HTTP without authentication.

For example, given an instance of `froyg` running on `localhost:1815` and an object at the key `foo/bar/hello-world` in the bucket `mybucket` in the region `us-east-1`, you can access this file at

```
http://localhost:1815/us-east-1/mybucket/foo/bar/hello-world
```

## Installation

```
$ go get -u src.beast175.com/citruspi/froyg
```
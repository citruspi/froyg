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

## Example

```
$ aws s3api create-bucket --bucket froyg-cc1 --create-bucket-configuration LocationConstraint=ca-central-1
{
    "Location": "http://froyg-cc1.s3.amazonaws.com/"
}
$ aws s3api create-bucket --bucket froyg-se1 --create-bucket-configuration LocationConstraint=sa-east-1
{
    "Location": "http://froyg-se1.s3.amazonaws.com/"
}
$ echo 'Hello from Canada!' > hello-world
$ aws s3 --region ca-central-1 cp hello-world s3://froyg-cc1/some/path/hello-world
upload: ./hello-world to s3://froyg-cc1/some/path/hello-world
$ echo 'Hello from Brazil!' > hello-world
$ aws s3 --region sa-east-1 cp hello-world s3://froyg-se1/another/path/hello-world
upload: ./hello-world to s3://froyg-se1/another/path/hello-world
$ curl http://localhost:1815/ca-central-1/froyg-cc1/some/path/hello-world
Hello from Canada!
$ curl http://localhost:1815/sa-east-1/froyg-se1/another/path/hello-world
Hello from Brazil!
```
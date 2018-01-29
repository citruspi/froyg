<img src="https://src.beast175.com/citruspi/froyg/raw/master/header.png" width="100%"/>

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

## Sample Use Cases

- Quickly access an object in S3 (e.g. a text file, a PDF, an image, etc) on your local workstation
- Backend for Nginx, Varnish, etc.
- Enable any program which understands HTTP to access objects in S3 without understanding S3

## Todo

- Allow binding to an address other than `:1815`
- Allow the user to set an explict whitelist of regions which can be accessed
- Allow the user to set an explict whitelist of buckets which can be accessed
- Set additional/improved HTTP headers based on S3 object attributes, e.g.
  - `Cache-Control`
  - `Content-Disposition`
  - `Content-Encoding`
  - `Content-Language`
  - `Content-Type`
  - `ETag`
  - `Expiration`
  - `Expires`
  - `LastModified`
  - `VersionId`
- Expose S3 object tags via HTTP headers
- Provide pre-compiled binaries for Linux and macOS (and make it available as an `rpm` and via `brew`)

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
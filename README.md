![https://src.doom.fm/citruspi/froyg/pipelines](https://src.doom.fm/citruspi/froyg/badges/master/pipeline.svg)
![https://src.doom.fm/citruspi/froyg/tags](https://img.shields.io/badge/version-0.2.1-blue.svg)

<img src="https://src.doom.fm/citruspi/froyg/raw/master/header.png" width="100%"/>

**Do not run an instance of `froyg` that is open to the Internet unless you're absolutely sure that it is safe to do so. If you have any doubt, do not do it.**

## Overview

`froyg` automatically exposes **every object** in **every bucket** in **every region** that you have access to over HTTP without authentication.

For example, given an instance of `froyg` running on `localhost:1815` and an object at the key `foo/bar/hello-world` in the bucket `mybucket` in the region `us-east-1`, you can access this file at

```
http://localhost:1815/us-east-1/mybucket/foo/bar/hello-world
```

### Sample Use Cases

- Quickly access an object in S3 (e.g. a text file, a PDF, an image, etc) on your local workstation
- Backend for Nginx, Varnish, etc.
- Enable any program which understands HTTP to access objects in S3 without understanding S3

## Installation
From source:

```
$ go get -u src.doom.fm/citruspi/froyg
```

Binaries for the following operating systems and architectures are also built and released for tagged versions at [releases.beastnet.works](#).

```
https://releases.beastnet.works/froyg/froyg-{OS}.{ARCH}-{REF}.tar.xf

Darwin          386 / amd64
Linux           386 / amd64 / amd64_static / arm / arm64
Windows         386 / amd64
Dragonfly       amd64
FreeBSD         386 / amd64 / arm
NetBSD          386 / amd64 / arm
OpenBSD         386 / amd64 / arm
```

## Configuration

Configuration of froyg is done entirely via CLI flags

```shell script
$ froyg --help
Usage of froyg:
  -bind string
        bind address (default "127.0.0.1:1815")
  -index string
        index file (default "index.html")
  -log-json
        json log format
  -v int
        verbosity (1-7; panic, fatal, error, warn, info, debug, trace) (default 4)
  -version
        show version and exit
```

## Usage

### Routing

The simplest method of accessing an S3 object via froyg is specifying the location completely via the HTTP URI:

```http request
GET /{region}/{bucket}/{key}
```

e.g.

```http request
GET /us-east-1/my-bucket/some/asset/to/fetch.json
```

An alternate to path-based routing is using HTTP headers, specifically
- `X-S3-Region`
- `X-S3-Bucket`
- `X-S3-Key-Prefix`

> Note: When using header-based routing, `X-S3-Region` and `X-S3-Bucket` are both required, `X-S3-Key-Prefix` is optional. Header-based routing will automatically be used if the `X-S3-Region` header is defined in the HTTP request. 

Using these headers, the same request could be made as

```http request
GET /some/asset/to/fetch.json
X-S3-Region us-east-1
X-S3-Bucket my-bucket
```

or even

```http request
GET /asset/to/fetch.json
X-S3-Region us-east-1
X-S3-Bucket my-bucket
X-S3-Key-Prefix some
```

### Web Server Mode

froyg can (and will) act as a webserver - when a request is made for an object which is not found, a second attempt will be made to retrieve the original key suffixed with an index file.

e.g. if a request is made for `foobar` which returns a `404`, froyg will attempt to return `foobar/index.html`. 

### HTTP Headers

froyg passes a number of HTTP headers from each client request as metadata when requesting the object from S3 and includes metadata from the backend response as HTTP headers in the response to the client. Some of these are standard HTTP headers meant to facilitate caching and other functions required by browers and other clients. Others are specific to froyg and/or S3 and are meant to facilitate interoperability with different programs. 

Information about headers not detailed below can be found [here][httpHeaders].

**Request Headers**

| Header | Details |
|:-------|:------------|
| `If-Match` | n/a |
| `If-None-Match` | n/a |
| `Range` | n/a |
| `If-Modified-Since` | n/a |
| `If-Unmodified-Since` | n/a |
| `X-S3-Object-Version` | Specify a version of the object to request |
| `X-S3-Object-Part` | Specify a part of the object to request |
| `X-Request-Id` | n/a |

**Response Headers**

| Header | Details |
|:-------|:------------|
| `Cache-Control` | n/a |
| `Content-Disposition` | n/a |
| `Content-Encoding` | n/a |
| `Content-Language` | n/a |
| `Content-Length` | n/a |
| `Content-Range` | n/a |
| `Content-Type` | n/a |
| `ETag` | n/a |
| `Expires` | n/a |
| `Last-Modified` | n/a |
| `X-S3-Object-Version` | n/a |
| `X-Request-Id` | ID from request if set otherwise UUIDv4 |

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
$ curl -H 'X-S3-Region: sa-east-1' -H 'X-S3-Bucket: froyg-se1' -H 'X-S3-Key-Prefix: another/path' \
    http://localhost:1815/hello-world
Hello from Brazil!
```

[httpHeaders]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
## v0.2.1 (18 January 2020)

- Bumped GitLab CI Go build version from 1.12.5 to 1.13.6
- Return a `400` if a request is received for an unknown region instead of panicking
- Include the S3 object version ID in the response where applicable
- Log HTTP request methods
- Return a `405` if an HTTP request is received with a verb other than `GET`
- Return a `412` on HTTP precondition failures
- Pass through `Content-Length` header from S3 to client
- Enabled Linux static amd64 builds

## v0.2.0 (26 June 2019)

- Bumped Gitlab CI Go build version from 1.9.3 to 1.12.5
- Enable auto-publishing releases via [releases.beastnet.works](#) with Gitlab CI
- Enabled Go modules
- Enabled binding to a user defined address
- Support for HTTP header-based routing where the S3 region, bucket, and optionally a key-prefix to be joined with the HTTP path are provided via HTTP headers
- Attempt to fall back to an index page in a second pass if the original key is not found 
- Log information about each step of the request (w/ support for human-friendly and JSON)
    - reading the HTTP request into an S3 object request
    - retrieving the object from the S3 backend
    - writing the S3 object and metadata into an HTTP response
- Tag each request with an ID for tracing through logs which is returned to the client via the `X-Request-Id` header
- Return a `304` if the object has not been modified
- Return a `404` if the requested bucket is not found
- Pass through the following HTTP headers from the client to S3
    - `If-Match`
    - `If-None-Match`
    - `Range`
    - `If-Modified-Since`
    - `If-Unmodified-Since`
    - `X-S3-Object-Version`
    - `X-S3-Object-Part`
- Pass through the following response metadata from S3 to the client via HTTP headers
    - `Cache-Control`
    - `Content-Disposition`
    - `Content-Encoding`
    - `Content-Language`
    - `Content-Range`
    - `Content-Type`
    - `ETag`
    - `Expires`
    - `Last-Modified`
    
## v0.1.1 (10 May 2018)

- Automatically set the `Content-Type` HTTP header based on the MIME type of the file extension

## v0.1.0 (29 January 2018)

- Initial release
- Supports retrieving an object from AWS S3 using HTTP path-based routing to provide the S3 region, bucket, and key

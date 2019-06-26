package main

import (
	"errors"
	"html"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type objectRequest struct {
	httpRequest *http.Request
	log         *logrus.Entry

	s3Region        string
	s3Bucket        *string
	s3ObjectRequest *s3.GetObjectInput
	s3Key           *string
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	rawUUID := uuid.Must(uuid.NewV4(), nil)

	o := &objectRequest{
		log: logrus.WithField("request_id", rawUUID.String()),
	}

	status := o.readHttpRequest(r)

	if status != http.StatusOK {
		o.log.WithField("resp_status_code", status).Infoln("writing http response")

		w.WriteHeader(status)
		return
	}

	o.writeHttpResponse(w)
}

func (o *objectRequest) readHttpRequest(r *http.Request) int {
	var s3Region *string
	var s3Bucket *string
	var s3Key *string

	o.log.WithFields(logrus.Fields{
		"req_url":      r.URL.String(),
		"req_headers":  r.Header,
		"req_addr":     r.RemoteAddr,
		"req_referrer": r.Referer(),
	}).Infoln("reading http request")

	if s3BucketHeader, ok := r.Header["X-S3-Bucket"]; ok {
		s3Bucket = &s3BucketHeader[0]

		if s3RegionHeader, ok := r.Header["X-S3-Region"]; ok {
			s3Region = &s3RegionHeader[0]

			if s3KeyPrefixHeader, ok := r.Header["X-S3-Key-Prefix"]; ok {
				key := path.Join(s3KeyPrefixHeader[0], r.URL.Path)
				s3Key = &key
			} else {
				s3Key = &r.URL.Path
			}
		} else {
			o.log.Warnln("malformed header routing request")
		}
	} else {
		tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 4)

		if len(tokens) < 4 {
			o.log.Warnln("malformed path routing request")
		} else {
			s3Region = &tokens[1]
			s3Bucket = &tokens[2]
			s3Key = &tokens[3]
		}
	}

	if s3Region == nil || s3Bucket == nil || s3Key == nil {
		return http.StatusBadRequest
	}

	o.s3Region = *s3Region
	o.s3ObjectRequest = &s3.GetObjectInput{
		Bucket:                     s3Bucket,
		IfMatch:                    nil,
		IfModifiedSince:            nil,
		IfNoneMatch:                nil,
		IfUnmodifiedSince:          nil,
		Key:                        s3Key,
		PartNumber:                 nil,
		Range:                      nil,
		RequestPayer:               nil,
		ResponseCacheControl:       nil,
		ResponseContentDisposition: nil,
		ResponseContentEncoding:    nil,
		ResponseContentLanguage:    nil,
		ResponseContentType:        nil,
		ResponseExpires:            nil,
		SSECustomerAlgorithm:       nil,
		SSECustomerKey:             nil,
		SSECustomerKeyMD5:          nil,
		VersionId:                  nil,
	}

	o.log.WithFields(logrus.Fields{
		"s3_region":     o.s3Region,
		"s3_object_req": o.s3ObjectRequest,
	}).Debugln("http request loaded")

	return http.StatusOK
}

func (o *objectRequest) fetchObject() (io.Reader, map[string]string, int) {
	object, err := s3conn[o.s3Region].GetObject(o.s3ObjectRequest)

	if err != nil {
		var errStatus int

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket:
				errStatus = http.StatusNotFound
			default:
				errStatus = http.StatusInternalServerError
			}

			err = errors.New(aerr.Message())
		} else {
			errStatus = http.StatusInternalServerError
		}

		o.log.WithField("error", err).Errorln("error retrieving object from backend")

		return nil, nil, errStatus
	}

	headers := make(map[string]string)

	rawHeaders := map[string]*string{
		"Cache-Control":       object.CacheControl,
		"Content-Disposition": object.ContentDisposition,
		"Content-Encoding":    object.ContentEncoding,
		"Content-Language":    object.ContentLanguage,
		"Content-Range":       object.ContentRange,
		"Content-Type":        object.ContentType,
		"ETag":                object.ETag,
		"Expires":             object.Expires,
	}

	for header, val := range rawHeaders {
		if val != nil {
			headers[header] = *val
		}
	}

	if object.LastModified != nil {
		headers["Last-Modified"] = object.LastModified.Format(http.TimeFormat)
	}

	return object.Body, headers, http.StatusOK
}

func (o *objectRequest) writeHttpResponse(w http.ResponseWriter) {
	body, headers, status := o.fetchObject()

	o.log.WithFields(logrus.Fields{
		"resp_headers":     headers,
		"resp_status_code": status,
	}).Infoln("writing http response")

	if headers != nil {
		for header, val := range headers {
			w.Header().Set(header, val)
		}
	}

	w.WriteHeader(status)

	if body == nil {
		return
	}

	_, err := io.Copy(w, body)

	if err != nil {
		o.log.WithField("error", err).Warnln("error writing response body")
	}
}

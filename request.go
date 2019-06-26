package main

import (
	"errors"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
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
	o := &objectRequest{}
	o.readHttpRequest(r)
	o.writeHttpResponse(w)
}

func (o *objectRequest) readHttpRequest(r *http.Request) {
	var s3Region string
	var s3Bucket *string
	var s3Key *string

	tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 4)

	s3Region = tokens[1]
	s3Bucket = &tokens[2]
	s3Key = &tokens[3]

	o.s3Region = s3Region
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
}

func (o *objectRequest) fetchObject() (io.Reader, map[string]string, int, error) {
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

		return nil, nil, errStatus, err
	}

	headers := make(map[string]string)

	return object.Body, headers, http.StatusOK, nil
}

func (o *objectRequest) writeHttpResponse(w http.ResponseWriter) {
	body, headers, status, err := o.fetchObject()

	if headers != nil {
		for header, val := range headers {
			w.Header().Set(header, val)
		}
	}

	w.WriteHeader(status)

	if err != nil {
		return
	}

	_, err = io.Copy(w, body)
}

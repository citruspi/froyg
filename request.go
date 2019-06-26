package main

import (
	"errors"
	"html"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

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
	requestId := r.Header.Get("X-Request-Id")

	if requestId == "" {
		requestId = uuid.Must(uuid.NewV4(), nil).String()
	}

	o := &objectRequest{
		log: logrus.WithField("request_id", requestId),
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

	stringHeaders := make(map[string]*string)
	timeHeaders := make(map[string]*time.Time)

	rawStringHeaders := []string{
		"If-Match",
		"If-None-Match",
		"Range",
		"X-S3-Object-Version",
	}

	rawTimeHeaders := []string{
		"If-Modified-Since",
		"If-Unmodified-Since",
	}

	for _, header := range rawStringHeaders {
		if headerVal, ok := r.Header[header]; ok {
			stringHeaders[header] = &headerVal[0]
		} else {
			stringHeaders[header] = nil
		}
	}

	for _, header := range rawTimeHeaders {
		if headerVal, ok := r.Header[header]; ok {
			rawTime, err := time.Parse(http.TimeFormat, headerVal[0])

			if err != nil {
				o.log.WithField("time", headerVal[0]).Warnln("malformed time header")

				return http.StatusBadRequest
			}

			timeHeaders[header] = &rawTime
		} else {
			timeHeaders[header] = nil
		}
	}

	var partNo *int64

	if partNoHeader, ok := r.Header["X-S3-Object-Part"]; ok {
		partNoRaw, err := strconv.ParseInt(partNoHeader[0], 10, 64)

		if err != nil {
			o.log.WithField("part_no", partNoHeader[0]).Warnln("malformed object part no")

			return http.StatusBadRequest
		}

		partNo = &partNoRaw
	}

	o.s3Region = *s3Region
	o.s3ObjectRequest = &s3.GetObjectInput{
		Bucket:            s3Bucket,
		IfMatch:           stringHeaders["If-Match"],
		IfModifiedSince:   timeHeaders["If-Modified-Since"],
		IfNoneMatch:       stringHeaders["If-None-Match"],
		IfUnmodifiedSince: timeHeaders["If-Unmodified-Since"],
		Key:               s3Key,
		PartNumber:        partNo,
		Range:             stringHeaders["Range"],
		VersionId:         stringHeaders["X-S3-Object-Version"],
	}

	return http.StatusOK
}

func (o *objectRequest) upstreamRequest(secondPass bool) (*s3.GetObjectOutput, int) {
	var object *s3.GetObjectOutput
	var err error

	status := http.StatusOK

	if strings.TrimSpace(*o.s3ObjectRequest.Key) == "/" {
		o.s3ObjectRequest.Key = &conf.IndexFile
		secondPass = true
	}

	o.log.WithFields(logrus.Fields{
		"s3_region":     o.s3Region,
		"s3_object_req": o.s3ObjectRequest,
	}).Debugln("establishing upstream request")

	object, err = s3conn[o.s3Region].GetObject(o.s3ObjectRequest)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				if secondPass {
					status = http.StatusNotFound
				} else {
					gnuKey := path.Join(*o.s3ObjectRequest.Key, conf.IndexFile)
					o.s3ObjectRequest.Key = &gnuKey

					object, status = o.upstreamRequest(true)
				}
			case s3.ErrCodeNoSuchBucket:
				status = http.StatusNotFound
			default:
				switch aerr.Code() {
				case "NotModified":
					status = http.StatusNotModified
				default:
					status = http.StatusInternalServerError
				}
			}

			err = errors.New(aerr.Message())
		} else {
			status = http.StatusInternalServerError
		}

		if status >= http.StatusBadRequest {
			o.log.WithField("error", err).Errorln("error retrieving object from backend")
		}
	}

	return object, status
}

func (o *objectRequest) fetchObject() (io.Reader, map[string]string, int) {
	object, status := o.upstreamRequest(false)

	if status != http.StatusOK {
		return nil, nil, status
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

	w.Header().Set("X-Request-Id", o.log.Data["request_id"].(string))
	w.WriteHeader(status)

	if body == nil {
		return
	}

	_, err := io.Copy(w, body)

	if err != nil {
		o.log.WithField("error", err).Warnln("error writing response body")
	}
}

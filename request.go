package main

import (
	"bytes"
	"errors"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	s3KeyPrefix     *string
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
		"req_method":   r.Method,
		"req_url":      r.URL.String(),
		"req_headers":  r.Header,
		"req_addr":     r.RemoteAddr,
		"req_referrer": r.Referer(),
	}).Infoln("reading http request")

	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed
	}

	if s3BucketHeader, ok := r.Header["X-S3-Bucket"]; ok {
		s3Bucket = &s3BucketHeader[0]

		if s3RegionHeader, ok := r.Header["X-S3-Region"]; ok {
			s3Region = &s3RegionHeader[0]

			if s3KeyPrefixHeader, ok := r.Header["X-S3-Key-Prefix"]; ok {
				key := path.Join(s3KeyPrefixHeader[0], r.URL.Path)
				s3Key = &key

				o.s3KeyPrefix = &s3KeyPrefixHeader[0]
			} else {
				s3Key = &r.URL.Path
			}
		} else {
			o.log.Warnln("malformed header routing request")
		}
	} else {
		tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 4)

		if len(tokens) == 4 {
			s3Region = &tokens[1]
			s3Bucket = &tokens[2]
			s3Key = &tokens[3]
		} else if len(tokens) == 3 {
			s3Region = &tokens[1]
			s3Bucket = &tokens[2]
		} else {
			o.log.Warnln("malformed path routing request")
		}
	}

	if s3Region == nil || s3Bucket == nil {
		return http.StatusBadRequest
	}

	if s3Key == nil || len(*s3Key) == 0 {
		s := "/"
		s3Key = &s
	}

	if _, ok := s3conn[*s3Region]; !ok {
		o.log.WithField("region", *s3Region).Warnln("unregistered S3 region requested")
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

	o.httpRequest = r
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

func (o *objectRequest) indexCommonPrefix(prefix string) (*s3.GetObjectOutput, int, error) {
	t, err := template.New("directory_index").Parse(DIR_INDEX_TEMPLATE)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	type Link struct {
		Name         string
		Href         string
		Size         string
		LastModified string
	}

	var links []Link

	prefix = strings.TrimPrefix(prefix, "/")

	if (o.s3KeyPrefix == nil && prefix != "") || (o.s3KeyPrefix != nil && len(strings.TrimPrefix(prefix, *o.s3KeyPrefix)) > 0) {
		links = append(links, Link{
			Name:         ".. /",
			Href:         "../",
			Size:         "",
			LastModified: "",
		})
	}

	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:    o.s3ObjectRequest.Bucket,
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(1000),
		Prefix:    aws.String(prefix),
	}

	o.log.WithFields(logrus.Fields{
		"s3_region":           o.s3Region,
		"s3_list_objects_req": listObjectsInput,
	}).Debugln("indexing prefix")

	var n int

	err = s3conn[o.s3Region].ListObjectsV2Pages(listObjectsInput, func(output *s3.ListObjectsV2Output, b bool) bool {
		for _, p := range output.CommonPrefixes {
			n += 1

			name := strings.TrimPrefix(*p.Prefix, prefix)

			links = append(links, Link{
				Name:         name,
				Href:         path.Join(o.httpRequest.URL.Path, url.QueryEscape(name[:len(name)-1])) + "/",
				Size:         "",
				LastModified: "",
			})
		}

		for _, object := range output.Contents {
			if *object.Key == prefix && *object.Size == 0 {
				continue
			}

			n += 1

			name := strings.TrimPrefix(*object.Key, prefix)

			links = append(links, Link{
				Name:         name,
				Href:         path.Join(o.httpRequest.URL.Path, url.QueryEscape(name)),
				Size:         strconv.FormatInt(*object.Size, 10),
				LastModified: object.LastModified.Format(time.RFC1123),
			})
		}

		return true
	})

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if n == 0 {
		return nil, http.StatusNotFound, nil
	}

	buf := bytes.Buffer{}

	err = t.Execute(&buf, struct {
		Links []Link
	}{
		Links: links,
	})

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	length := int64(buf.Len())
	type_ := "text/html"

	return &s3.GetObjectOutput{
		Body:          io.NopCloser(&buf),
		ContentLength: &length,
		ContentType:   &type_,
	}, http.StatusOK, nil
}

func (o *objectRequest) upstreamRequest() (*s3.GetObjectOutput, int) {
	var object *s3.GetObjectOutput
	var err error
	var status int

	k := strings.TrimSpace(*o.s3ObjectRequest.Key)

	if k[len(k)-1:] == "/" {
		if conf.ServeWww {
			k_mod := path.Join(k, conf.IndexFile)
			o.s3ObjectRequest.Key = &k_mod

			object, status, err = o.getObject()
		}

		if conf.AutoIndex && (status == http.StatusNotFound || status == 0) {
			index, status, err := o.indexCommonPrefix(k)

			if err == nil {
				return index, status
			} else {
				logrus.WithError(err).Errorln("failed to index common prefix")
				return nil, status
			}
		}
	} else {
		object, status, err = o.getObject()

		if status == http.StatusNotFound && conf.ServeWww {
			k_mod := path.Join(k, conf.IndexFile)
			o.s3ObjectRequest.Key = &k_mod

			object, status, err = o.getObject()
		}

		if status == http.StatusNotFound && conf.AutoIndex {
			index, status, err := o.indexCommonPrefix(k + "/")

			if err == nil {
				return index, status
			} else {
				logrus.WithError(err).Errorln("failed to index common prefix")
				return nil, status
			}
		}
	}

	if err != nil {
		logrus.WithError(err).Errorln("err was not nil")
	}

	return object, status
}

func (o *objectRequest) getObject() (*s3.GetObjectOutput, int, error) {
	object, err := s3conn[o.s3Region].GetObject(o.s3ObjectRequest)
	status := http.StatusOK

	o.log.WithFields(logrus.Fields{
		"s3_region":     o.s3Region,
		"s3_object_req": o.s3ObjectRequest,
	}).Debugln("getting S3 object")

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket:
				status = http.StatusNotFound
				err = nil
			default:
				switch aerr.Code() {
				case "NotModified":
					status = http.StatusNotModified
					err = nil
				case "PreconditionFailed":
					status = http.StatusPreconditionFailed
					err = nil
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

	return object, status, err
}

func (o *objectRequest) fetchObject() (io.Reader, map[string]string, int) {
	object, status := o.upstreamRequest()

	if status != http.StatusOK {
		return nil, map[string]string{"Content-Length": "0"}, status
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
		"X-S3-Object-Version": object.VersionId,
	}

	for header, val := range rawHeaders {
		if val != nil {
			headers[header] = *val
		}
	}

	if object.LastModified != nil {
		headers["Last-Modified"] = object.LastModified.Format(http.TimeFormat)
	}

	if object.ContentLength == nil {
		headers["Content-Length"] = "0"
	} else {
		headers["Content-Length"] = strconv.FormatInt(*object.ContentLength, 10)
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

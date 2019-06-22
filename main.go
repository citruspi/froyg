package main

import (
	"flag"
	"html"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	BindAddress string
	LogJSON     bool
}

var (
	s3Regions = []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"ca-central-1",
		"ap-south-1",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-southeast-1",
		"ap-southeast-2",
		"cn-north-1",
		"cn-northwest-1",
		"eu-central-1",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"sa-east-1",
	}

	s3conn = make(map[string]*s3.S3)

	conf = &Configuration{}
)

func contentTypeForPath(p string) string {
	contentType := ""
	extension := path.Ext(p)

	if extension != "" {
		contentType = mime.TypeByExtension(extension)
	}

	return contentType
}

func parseFlags() {
	flag.StringVar(&conf.BindAddress, "bind", "127.0.0.1:1815", "bind address")
	flag.BoolVar(&conf.LogJSON, "log-json", false, "json log format")

	flag.Parse()
}

func main() {
	parseFlags()

	if conf.LogJSON {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:          true,
			DisableLevelTruncation: true,
		})
	}

	for _, region := range s3Regions {
		s3conn[region] = s3.New(session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		})))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqLog := log.WithFields(log.Fields{
			"req_url":      r.URL.String(),
			"req_headers":  r.Header,
			"req_addr":     r.RemoteAddr,
			"req_referrer": r.Referer(),
		})

		tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 4)

		object, err := s3conn[tokens[1]].GetObject(&s3.GetObjectInput{
			Bucket: aws.String(tokens[2]),
			Key:    aws.String(tokens[3]),
		})

		if err != nil {
			var status int

			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket:
					status = http.StatusNotFound
				default:
					status = http.StatusInternalServerError
				}

				reqLog.Data["error"] = aerr.Message()
			} else {
				status = http.StatusInternalServerError
				reqLog.Data["error"] = err.Error()
			}

			w.WriteHeader(status)
			reqLog.WithField("resp_code", status).Errorln("")

			return
		}

		contentType := contentTypeForPath(tokens[3])

		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		_, err = io.Copy(w, object.Body)

		if err != nil {
			reqLog.WithFields(log.Fields{
				"resp_code": http.StatusOK,
				"error":     err,
			}).Warnln("")
		} else {
			reqLog.WithFields(log.Fields{
				"resp_code": http.StatusOK,
			}).Infoln("")
		}
	})

	log.Fatalln(http.ListenAndServe(conf.BindAddress, nil))
}

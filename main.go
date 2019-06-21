package main

import (
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Configuration struct {
	BindAddress string
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

	flag.Parse()
}

func main() {
	parseFlags()

	for _, region := range s3Regions {
		s3conn[region] = s3.New(session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		})))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 4)

		object, err := s3conn[tokens[1]].GetObject(&s3.GetObjectInput{
			Bucket: aws.String(tokens[2]),
			Key:    aws.String(tokens[3]),
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket:
					w.WriteHeader(http.StatusNotFound)
				default:
					fmt.Println(aerr.Error())
					w.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				fmt.Println(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		contentType := contentTypeForPath(tokens[3])

		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		io.Copy(w, object.Body)
	})

	log.Fatal(http.ListenAndServe(conf.BindAddress, nil))
}

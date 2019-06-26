package main

import (
	"flag"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
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

func init() {
	flag.StringVar(&conf.BindAddress, "bind", "127.0.0.1:1815", "bind address")
	logJson := flag.Bool("log-json", false, "json log format")
	logLevel := flag.Int("v", 4, "verbosity (1-7; panic, fatal, error, warn, info, debug, trace)")

	flag.Parse()

	if *logJson {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:          true,
			DisableLevelTruncation: true,
		})
	}

	if (1 <= *logLevel) && (*logLevel <= 7) {
		log.SetLevel(log.Level(*logLevel))
	} else {
		log.Fatalln("log level must be between 1 and 7 (inclusive)")
	}

	for _, region := range s3Regions {
		s3conn[region] = s3.New(session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		})))
	}
}

func main() {
	http.HandleFunc("/", httpHandler)

	log.Fatalln(http.ListenAndServe(conf.BindAddress, nil))
}

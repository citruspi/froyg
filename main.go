package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	BindAddress string
	IndexFile   string
	ServeWww    bool
}

var (
	version string = "unset"

	s3Regions = []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"ca-central-1",
		"ap-east-1",
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
		"eu-north-1",
		"me-south-1",
		"sa-east-1",
	}

	s3conn = make(map[string]*s3.S3)

	conf = &Configuration{}
)

func init() {
	flag.StringVar(&conf.BindAddress, "bind", "127.0.0.1:1815", "bind address")
	flag.StringVar(&conf.IndexFile, "index", "index.html", "index file")
	flag.BoolVar(&conf.ServeWww, "www", false, "act as web server")
	versionFlag := flag.Bool("version", false, "show version and exit")
	logJson := flag.Bool("log-json", false, "json log format")
	logLevel := flag.Int("v", 4, "verbosity (1-7; panic, fatal, error, warn, info, debug, trace)")

	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

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

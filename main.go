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
	AutoIndex   bool
}

const (
	DIR_INDEX_TEMPLATE = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: monospace; font-size: 115%; }
			table { border-collapse: collapse; }
			strong#title { margin-bottom: 1rem; }
			th { text-align: left; padding: 0.5rem 15px; font-size: 120%; }
			thead > tr { border-bottom: 1.5px solid #131313; }
			thead > tr > th { padding-bottom: 0.5rem; }
			tbody > tr > td { padding: 0.25rem 15px; }
			tbody > tr:first-child td { padding-top: 0.5rem; }
			tr:nth-child(even) { background-color: #EDEDED; }
			th#name { min-width: 175px; padding-right: 1rem; }
			th#size { min-width: 75px; padding-right: 1rem; }
			th#lmod { min-width: 250px; }
			a { text-decoration: none; }

			@media (prefers-color-scheme: dark) {
				body { background-color: #131313; color: white; }
				thead > tr { border-bottom-color: white; }
				a { color: #FFCD00; }
				a:visited { color: #FF9800; }
				tr:nth-child(even) { background-color: #2A2A2A; }
			}
		</style>
	</head>
	<body>
		<table>
			<thead>
				<tr>
					<th id="name">Name</th>
					<th id="size">Size</th>
					<th id="lmod">Last Modified</th>
				</tr>
			</thead>
			<tbody>
				{{range .Links}}
				<tr>
					<td><a href="{{ .Href }}">{{ .Name }}</a></td>
					<td>{{ .Size }}</td>
					<td>{{ .LastModified }}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</body>
</html>`
)

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
	flag.BoolVar(&conf.AutoIndex, "auto-index", false, "auto index common prefixes")

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

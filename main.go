package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/influxdata/influxdb-client-go/v2"
	influxAPI "github.com/influxdata/influxdb-client-go/v2/api"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	BindAddress     string
	IndexFile       string
	ServeWww        bool
	AutoIndex       bool
	CPITemplate     *template.Template
	CPIMsg          string
	CPIFooter       string
	CPICacheControl string
}

const (
	DIR_INDEX_TEMPLATE = `<!DOCTYPE html>
<html>
	<head>
		<title>{{ .Title }} @ /{{ range .Prefix }}{{.Name}}{{end}}</title>
		<meta charset="UTF-8">
		<style>
			body { font-family: monospace; font-size: 110%; }
			table { width: 100%; border-collapse: collapse; }
			strong#title { margin-bottom: 1rem; }
			th { text-align: left; padding: 0.5rem 15px; font-size: 110%; }
			thead > tr { border-bottom: 1.5px solid #131313; }
			thead > tr > th { padding-bottom: 0.5rem; }
			tbody > tr > td { padding: 0.25rem 15px; overflow: scroll; }
			tbody > tr:first-child td { padding-top: 0.5rem; }
			tr:nth-child(even), div#message { background-color: #EDEDED; }
			th#name { min-width: 175px; padding-right: 1rem; }
			th#size { min-width: 75px; max-width: 100px; padding-right: 1rem; }
			td#size { text-align: right; }
			th#lmod { min-width: 250px; }
			a, span#title-separator { color: #FF0000; text-decoration: none; }
			a:visited { color: #8100FF; }
			h3 { font-size: 125%; margin-top: 0; margin-bottom: 5px; }
			span#title-separator { font-size: 140%; }
			div#message { padding: 10px; margin: 10px 0; overflow: wrap; }
			div#footer { margin-top: 1.5rem; margin-bottom: 1rem; }
			div.container { width: 100%; max-width: 800px; overflow: scroll; }
			code { color: white; }
			p > code { background-color: grey; padding: 2px 3px; }
			pre { background-color: grey; padding: 5px; }

			@media (prefers-color-scheme: dark) {
				body { background-color: #131313; color: white; }
				thead > tr { border-bottom-color: white; }
				a, span#title-separator { color: #FFCD00; }
				a:visited { color: #FF9800; }
				tr:nth-child(even), div#message { background-color: #2A2A2A; }
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h3>{{if gt (len .TitleLink) 0}}<a href="{{.TitleLink}}">{{ .Title }}</a>{{else}}{{ .Title }}{{end}} <span id="title-separator">&#10031;</span> /{{ range .Prefix }}<a href="{{.Href}}">{{.Name}}</a>/{{end}}</h3>
			{{if .Root }}{{if gt (len .Message) 0}}<div id="message">{{ .Message }}</div>{{end}}{{end}}
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
						<td id="size">{{ .Size }}</td>
						<td>{{ .LastModified }}</td>
					</tr>
					{{end}}
				</tbody>
			</table>
			{{if gt (len .Footer) 0}}<div id="footer">{{ .Footer }}</div>{{end}}
		</div>
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

	influxDB influxAPI.WriteAPI = nil
)

func init() {
	flag.StringVar(&conf.BindAddress, "bind", "127.0.0.1:1815", "bind address")
	flag.StringVar(&conf.IndexFile, "index", "index.html", "index file")
	flag.BoolVar(&conf.ServeWww, "www", false, "act as web server")
	flag.BoolVar(&conf.AutoIndex, "auto-index", false, "auto index common prefixes")
	flag.StringVar(&conf.CPIMsg, "auto-index-msg-html", "", "common prefixes index HTML message")
	flag.StringVar(&conf.CPIFooter, "auto-index-footer-html", "", "common prefixes index HTML footer")
	flag.StringVar(&conf.CPICacheControl, "auto-index-cache-control", "", "common prefixes index Cache-Control header")

	influxDBHost := flag.String("influxdb2-host", "http://localhost:8086", "InfluxDB 2 server address")
	influxDBToken := flag.String("influxdb2-token", "", "InfluxDB 2 write token")
	influxDBOrg := flag.String("influxdb2-org", "", "InfluxDB 2 organization")
	influxDBBucket := flag.String("influxdb2-bucket", "froyg", "InfluxDB 2 bucket")
	influxDBBatchSize := flag.Uint("influxdb2-batch-size", 100, "InfluxDB 2 write batch size")
	influxDBFlushInterval := flag.Uint("influxdb2-flush-interval", 1000, "InfluxDB 2 flush interval (ms)")
	influxDBPrecision := flag.String("influxdb2-precision", "ns", "InfluxDB 2 precision (ns, μs, ms, or s)")

	cpiTemplatePath := flag.String("auto-index-template", "", "path to custom template for common prefix index")
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

	var t *template.Template
	var raw []byte
	var err error

	if cpiTemplatePath == nil || len(strings.TrimSpace(*cpiTemplatePath)) == 0 {
		t, err = template.New("directory_index").Parse(DIR_INDEX_TEMPLATE)
	} else {
		raw, err = ioutil.ReadFile(*cpiTemplatePath)

		if err == nil {
			t, err = template.New("directory_index").Parse(string(raw))
		}
	}

	if err != nil || t == nil {
		log.WithError(err).Fatalln("failed to parse common prefix index template")
	}

	conf.CPITemplate = t

	if influxDBToken != nil && len(*influxDBToken) > 0 && influxDBOrg != nil && len(*influxDBOrg) > 0 {
		var precision time.Duration

		switch *influxDBPrecision {
		case "ns":
			precision = time.Nanosecond
		case "μs", "us":
			precision = time.Microsecond
		case "ms":
			precision = time.Millisecond
		case "s":
			precision = time.Second
		default:
			log.WithField("precision", *influxDBPrecision).Fatalln("unsupported InfluxDB 2 precision")
		}

		client := influxdb2.NewClientWithOptions(*influxDBHost, *influxDBToken,
			influxdb2.DefaultOptions().SetBatchSize(*influxDBBatchSize).SetFlushInterval(*influxDBFlushInterval).SetPrecision(precision))

		influxDB = client.WriteAPI(*influxDBOrg, *influxDBBucket)
	}
}

func main() {
	http.HandleFunc("/", httpHandler)

	log.Fatalln(http.ListenAndServe(conf.BindAddress, nil))
}

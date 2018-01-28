package main

import (
    "io"
    "fmt"
    "log"        
    "html"
    "net/http"
    "strings"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/awserr"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

var (
    s3_regions = []string{
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
)

func main() {
    for _, region := range s3_regions {
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
                case s3.ErrCodeNoSuchKey:
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

        io.Copy(w, object.Body)
    })

    log.Fatal(http.ListenAndServe(":1815", nil))
}
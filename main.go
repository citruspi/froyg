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
    aws_us_east_1 = session.Must(session.NewSession(&aws.Config{
        Region: aws.String("us-east-1"),
    }))

    s3conn = s3.New(aws_us_east_1)
)

func fetchObject(bucket string, key string) (*s3.GetObjectOutput, error) {
    requestInput := &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    }

    return s3conn.GetObject(requestInput)
}

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 3)

        object, err := fetchObject(tokens[1], tokens[2])

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
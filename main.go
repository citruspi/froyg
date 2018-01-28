package main

import (
    "fmt"
    "log"        
    "html"
    "net/http"
    "strings"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        tokens := strings.SplitN(html.EscapeString(r.URL.Path), "/", 3)
        
        bucket := tokens[1]
        key := tokens[2]
    })

    log.Fatal(http.ListenAndServe(":1815", nil))
}
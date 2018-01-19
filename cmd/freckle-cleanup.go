package main

import "os"
import "flag"
import "fmt"
import "log"
import "net/url"
import "strconv"
import "errors"

import (
    "github.com/parnurzeal/gorequest";
    "github.com/peterhellberg/link";
)

// http://developer.letsfreckle.com/v2/tags/
type tag struct {
    Id int `json:"id"`
    Name string `json:"name"`
    Entries int `json:"entries"`
}

// Generate a personal access token as per
// http://developer.letsfreckle.com/v2/authentication/#using-personal-access-tokens
var token = os.Getenv("FRECKLE_TOKEN")

const threads = 4

func main() {
    thresholdPtr := flag.Int("threshold", 0, "Tags with less than this number of entries will be deleted")
    doDeletePtr := flag.Bool("do-delete", false, "Actually delete tags")
    flag.Parse()
    to_delete := make([]tag, 0);

    fmt.Println("Fetching all tags. This may take some time.")
    tags := getTags()
    fmt.Println(len(tags), "tags have been fetched.")

    for i := 0; i < len(tags); i++ {
        if (tags[i].Entries < *thresholdPtr) {
            to_delete = append(to_delete, tags[i])
        }
    }

    fmt.Println(len(to_delete), "tags used less than", *thresholdPtr, "times.")

    type tids struct {
        tag_ids []int
    }
    tag_ids := tids{}
    for i := 0; i < len(to_delete); i++ {
        fmt.Println("Delete tag", to_delete[i].Name, "with", to_delete[i].Entries, "usages.")
        tag_ids.tag_ids = append(tag_ids.tag_ids, to_delete[i].Id)
    }

    if *doDeletePtr {
        fmt.Println("Doing delete...")
        request := gorequest.New()
        resp, body, err := request.Put("https://api.letsfreckle.com/v2/tags/dele").
          Send(tag_ids).
          Set("X-FreckleToken", token).
          End()

        if err != nil {
            if (resp.StatusCode >= 299) {
                log.Fatal("Freckle returned an error: " + resp.Status + " ")
            }

            log.Fatal(err)
        }

        fmt.Println(body)
    } else {
        fmt.Println("No tags have been deleted.")
    }
}

// Fetch all tags, walking next relations as required.
func getTags() ([]tag) {
    // The tags to return
    var tags []tag
    tagChannel := make(chan []tag, threads)

    pages, _ := getLastTagPage()
    semaphore := make(chan bool, threads)
    for page := 0; page <= pages; page++ {
        semaphore <- true
        fmt.Println(page)
        uri := "https://api.letsfreckle.com/v2/tags?per_page=100&page=" + strconv.Itoa(page)
        go requestTags(uri, tagChannel, semaphore)
        go func() {
          tags = append(tags, <-tagChannel...)
        }()
    }

    // Ensure all requests have finished.
    for i := 0; i < cap(semaphore); i++ {
        semaphore <- true
    }

    return tags
}

func getLastTagPage() (int, error) {
    uri := "https://api.letsfreckle.com/v2/tags?per_page=100"
    request := gorequest.New()
    // EndStruct() automatically parses the response using the struct format
    // in the header. Cool!
    resp, _, err := request.Get(uri).
      Set("X-FreckleToken", token).
      End()

    if err != nil {
        if (resp.StatusCode >= 299) {
            log.Fatal("Freckle returned an error: " + resp.Status + " ")
        }

        log.Fatal(err)
    }

    // Parse out the last URI to calculate how many pages there are.
    link := link.ParseResponse(resp)
    if link["last"] != nil {
        last := link["last"].URI
        parsed, err := url.ParseRequestURI(last)
        if err != nil {
            log.Fatal("Unable to parse", last)
        }
        query := parsed.Query()
        last_page := query.Get("page")
        return strconv.Atoi(last_page)
    }

    return -1, errors.New("There was no last relationship in the response")
}

// Make the actual request for tags
func requestTags(uri string, tags chan []tag, semaphore chan bool) {
    var tagBuffer []tag

    defer func() { <- semaphore }()

    request := gorequest.New()
    // EndStruct() automatically parses the response using the struct format
    // in the header. Cool!
    resp, _, err := request.Get(uri).
      Set("X-FreckleToken", token).
      EndStruct(&tagBuffer)

    if err != nil {
        if (resp.StatusCode >= 299) {
            log.Fatal("Freckle returned an error: " + resp.Status + " ")
        }

        log.Fatal(err)
    }

    tags <- tagBuffer
}

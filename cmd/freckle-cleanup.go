package main

import "os"
import "flag"
import "fmt"
import "log"
import "net/url"
import "strconv"
import "errors"
import "time"

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

// 7+ threads causes HTTP 429 Too Many Requests.
const threads = 6

// The number of tags to fetch at once. High numbers result is lower
// performance.
const page_count = "400"

func main() {
    thresholdPtr := flag.Int("threshold", 0, "Tags with less than this number of entries will be deleted")
    doDeletePtr := flag.Bool("do-delete", false, "Actually delete tags")
    flag.Parse()
    to_delete := make([]tag, 0);

    fmt.Println("Fetching all tags. This may take some time.")
    tagChannel := make(chan tag, threads)
    go getTags(tagChannel)

    tagCount := 0
    for tag := range tagChannel {
        tagCount++;
        if (tag.Entries < *thresholdPtr) {
            to_delete = append(to_delete, tag)
        }
    }
    fmt.Println(tagCount, "tags have been fetched.")

    fmt.Println(len(to_delete), "tags used less than", *thresholdPtr, "times eligible to be deleted.")

    // http://developer.letsfreckle.com/v2/tags/#delete-multiple-tags-at-once
    // requires a tag_ids key in the PUT request.
    type tids struct {
        tag_ids []int
    }
    tag_ids := tids{}
    for i := 0; i < len(to_delete); i++ {
        fmt.Println("Delete tag", to_delete[i].Name, "with", to_delete[i].Entries, "usages.")
        tag_ids.tag_ids = append(tag_ids.tag_ids, to_delete[i].Id)
    }

    // Actually do the delete!
    if *doDeletePtr {
        fmt.Println("Last chance to cancel. Will do delete in 3 seconds...")
        time.Sleep(3 * time.Second)
        request := gorequest.New()
        // @todo Correct this when I'm ready to test live!
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
func getTags(tagChannel chan tag) {
    pages, _ := getLastTagPage()
    semaphore := make(chan bool, threads)
    for page := 1; page <= pages; page++ {
        fmt.Print(".")
        uri := "https://api.letsfreckle.com/v2/tags?per_page=" + page_count + "&page=" + strconv.Itoa(page)
        semaphore <- true
        go requestTags(uri, tagChannel, semaphore)
    }
    fmt.Println()

    // Ensure all requests have finished.
    for i := 0; i < cap(semaphore); i++ {
        semaphore <- true
    }

    // This lets us range over the channel.
    close(tagChannel)
}

// Determine what the last page number is so we can thread each request.
func getLastTagPage() (int, error) {
    uri := "https://api.letsfreckle.com/v2/tags?per_page=" + page_count
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
func requestTags(uri string, tagChan chan tag, semaphore chan bool) {
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

    for i := 0; i < len(tagBuffer); i++ {
        tagChan <- tagBuffer[i]
    }
}

package main

import "os"
import "fmt"
import "log"

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

// Delete tags with this number of entries or less.
const threshold = 1

func main() {
    var to_delete = make([]tag, 0);

    fmt.Println("Fetching all tags. This may take some time.")
    tags := getTags()

    for i := 0; i < len(tags); i++ {
        if (tags[i].Entries <= threshold) {
            to_delete = append(to_delete, tags[i])
        }
    }

    fmt.Println(len(to_delete), "tags used less than", threshold, "times.")
    for i := 0; i < len(to_delete); i++ {
        fmt.Println("Delete tag", to_delete[i].Name, "with", to_delete[i].Entries, "usages.")
    }
}

// Fetch all tags, walking next relations as required.
func getTags() ([]tag) {
    // The tags to return
    var tags []tag
    // The tags returned from a single request
    var t []tag

    // 25 is the default page limit and is painfully slow.
    uri := "https://api.letsfreckle.com/v2/tags?per_page=100"
    for {
      if uri != "" {
          t, uri = requestTags(uri)
          tags = append(tags, t...)
          fmt.Println("Fetched", len(tags), "tags...")
      } else {
          // We reached the end of the list of tags
          break
      }
    }

    return tags
}

// Make the actual request for tags
func requestTags(uri string) ([]tag, string){
    var tags []tag
    var next string

    request := gorequest.New()
    // EndStruct() automatically parses the response using the struct format
    // in the header. Cool!
    resp, _, err := request.Get(uri).
      Set("X-FreckleToken", token).
      EndStruct(&tags)

    if err != nil {
        if (resp.StatusCode >= 299) {
            log.Fatal("Freckle returned an error: " + resp.Status + " ")
        }

        log.Fatal(err)
    }

    // Parse out the next URI to fetch tags from.
    link := link.ParseResponse(resp)
    if link["next"] != nil {
        next = link["next"].URI
    } else {
        next = ""
    }

    return tags, next
}

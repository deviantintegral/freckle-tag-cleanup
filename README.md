# Freckle Tag Cleanup Tool

[Freckle](https://letsfreckle.com) has a UI with reporting on least-used tags,
but no bulk UI for cleaning them up. Reducing tags can help with reporting and
with the performance of their mobile timer. This tool:

* Fetches all tags
* Filters them based on the threshold set
* Deletes them (which really just means editing the entry to remove the leading
  hash).

# To use

1. Clone this repository
1. Create a
   [Personal Access Token](http://developer.letsfreckle.com/v2/authentication/#using-personal-access-tokens)
   and export it to the `FRECKLE_TOKEN` environment variable.
1. `go run cmd/freckle-cleanup.go --help`
1. To actually delete some tags: `go run cmd/freckle-cleanup.go --threshold=0 --do-delete`

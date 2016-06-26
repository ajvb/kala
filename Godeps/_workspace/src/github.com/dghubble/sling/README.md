
# Sling [![Build Status](https://travis-ci.org/dghubble/sling.png)](https://travis-ci.org/dghubble/sling) [![Coverage](http://gocover.io/_badge/github.com/dghubble/sling)](http://gocover.io/github.com/dghubble/sling) [![GoDoc](http://godoc.org/github.com/dghubble/sling?status.png)](http://godoc.org/github.com/dghubble/sling)
<img align="right" src="https://s3.amazonaws.com/dghubble/small-gopher-with-sling.png">

Sling is a Go HTTP client library for creating and sending API requests.

Slings store HTTP Request properties to simplify sending requests and decoding responses. Check [usage](#usage) or the [examples](examples) to learn how to compose a Sling into your API client.

Note: Sling **v1.0** recently introduced some breaking changes. See [changes](CHANGES.md).

### Features

* Base/Path - path extend a Sling for different endpoints
* Method Setters: Get/Post/Put/Patch/Delete/Head
* Add and Set Request Headers
* Encode structs into URL query parameters
* Encode a form or JSON into the Request Body
* Receive JSON success or failure responses

## Install

    go get github.com/dghubble/sling

## Documentation

Read [GoDoc](https://godoc.org/github.com/dghubble/sling)

## Usage

Use a Sling to create an `http.Request` with a chained API for setting properties (path, method, queries, body, etc.).

```go
type Params struct {
    Count int `url:"count,omitempty"`
}
params := &Params{Count: 5}

req, err := sling.New().Get("https://example.com").QueryStruct(params).Request()
client.Do(req)
```

### Path

Use `Path` to set or extend the URL for created Requests. Extension means the path will be resolved relative to the existing URL.

```go
// sends a GET request to http://example.com/foo/bar
req, err := sling.New().Base("http://example.com/").Path("foo/").Path("bar").Request()
```

Use `Get`, `Post`, `Put`, `Patch`, `Delete`, or `Head` which are exactly the same as `Path` except they set the HTTP method too.

```go
req, err := sling.New().Post("http://upload.com/gophers")
```

### Headers

`Add` or `Set` headers which should be applied to the Requests created by a Sling.

```go
base := sling.New().Base(baseUrl).Set("User-Agent", "Gophergram API Client")
req, err := base.New().Get("gophergram/list").Request()
```

### QueryStruct

Define [url parameter structs](https://godoc.org/github.com/google/go-querystring/query) and use `QueryStruct` to encode query parameters.

```go
// Github Issue Parameters
type IssueParams struct {
    Filter    string `url:"filter,omitempty"`
    State     string `url:"state,omitempty"`
    Labels    string `url:"labels,omitempty"`
    Sort      string `url:"sort,omitempty"`
    Direction string `url:"direction,omitempty"`
    Since     string `url:"since,omitempty"`
}
```

```go
githubBase := sling.New().Base("https://api.github.com/").Client(httpClient)
path := fmt.Sprintf("repos/%s/%s/issues", owner, repo)

params := &IssueParams{Sort: "updated", State: "open"}
req, err := githubBase.New().Get(path).QueryStruct(params).Request()
```

### Body

#### Json Body

Make a Sling include JSON in the Body of its Requests using `BodyJSON`.

```go
type IssueRequest struct {
    Title     string   `json:"title,omitempty"`
    Body      string   `json:"body,omitempty"`
    Assignee  string   `json:"assignee,omitempty"`
    Milestone int      `json:"milestone,omitempty"`
    Labels    []string `json:"labels,omitempty"`
}
```

```go
githubBase := sling.New().Base("https://api.github.com/").Client(httpClient)
path := fmt.Sprintf("repos/%s/%s/issues", owner, repo)

body := &IssueRequest{
    Title: "Test title",
    Body:  "Some issue",
}
req, err := githubBase.New().Post(path).BodyJSON(body).Request()
```

Requests will include an `application/json` Content-Type header.

#### Form Body

Make a Sling include a url-tagged struct as a url-encoded form in the Body of its Requests using `BodyForm`.

```go
type StatusUpdateParams struct {
    Status             string   `url:"status,omitempty"`
    InReplyToStatusId  int64    `url:"in_reply_to_status_id,omitempty"`
    MediaIds           []int64  `url:"media_ids,omitempty,comma"`
}
```

```go
tweetParams := &StatusUpdateParams{Status: "writing some Go"}
req, err := twitterBase.New().Post(path).BodyForm(tweetParams).Request()
```

Requests will include an `application/x-www-form-urlencoded` Content-Type header.

### Extend a Sling

Each distinct Sling generates an `http.Request` (say with some path and query
params) each time `Request()` is called, based on its state. When creating
different kinds of requests using distinct Slings, you may wish to extend
an existing Sling to minimize duplication (e.g. a common client).

Each Sling instance provides a `New()` method which creates an independent copy, so setting properties on the child won't mutate the parent Sling. 

```go
const twitterApi = "https://api.twitter.com/1.1/"
base := sling.New().Base(twitterApi).Client(httpAuthClient)

// statuses/show.json Sling
tweetShowSling := base.New().Get("statuses/show.json").QueryStruct(params)
req, err := tweetShowSling.Request()

// statuses/update.json Sling
tweetPostSling := base.New().Post("statuses/update.json").BodyForm(params)
req, err := tweetPostSling.Request()
```

Without the calls to `base.New()`, tweetShowSling and tweetPostSling reference
the base Sling and POST to
"https://api.twitter.com/1.1/statuses/show.json/statuses/update.json", which
is undesired.

Recap: If you wish to extend a Sling, create a new child copy with `New()`.

### Receive

Define a JSON struct to decode a type from 2XX success responses. Use `ReceiveSuccess(successV interface{})` to send a new Request and decode the response body into `successV` if it succeeds.

```go
// Github Issue (abbreviated)
type Issue struct {
    Title  string `json:"title"`
    Body   string `json:"body"`
}
```

```go
issues := new([]Issue)
resp, err := githubBase.New().Get(path).QueryStruct(params).ReceiveSuccess(issues)
fmt.Println(issues, resp, err)
```

Most APIs return failure responses with JSON error details. To decode these, define success and failure JSON structs. Use `Receive(successV, failureV interface{})` to send a new Request that will automatically decode the response into the `successV` for 2XX responses or into `failureV` for non-2XX responses.

```go
type GithubError struct {
    Message string `json:"message"`
    Errors  []struct {
        Resource string `json:"resource"`
        Field    string `json:"field"`
        Code     string `json:"code"`
    } `json:"errors"`
    DocumentationURL string `json:"documentation_url"`
}
```

```go
issues := new([]Issue)
githubError := new(GithubError)
resp, err := githubBase.New().Get(path).QueryStruct(params).Receive(issues, githubError)
fmt.Println(issues, githubError, resp, err)
```

Pass a nil `successV` or `failureV` argument to skip JSON decoding into that value.

### Build an API

APIs typically define an endpoint (also called a service) for each type of resource. For example, here is a tiny Github IssueService which [lists](https://developer.github.com/v3/issues/#list-issues-for-a-repository) repository issues.

```go
type IssueService struct {
    sling *sling.Sling
}

func NewIssueService(httpClient *http.Client) *IssueService {
    return &IssueService{
        sling: sling.New().Client(httpClient).Base(baseURL),
    }
}

func (s *IssueService) ListByRepo(owner, repo string, params *IssueListParams) ([]Issue, *http.Response, error) {
    issues := new([]Issue)
    githubError := new(GithubError)
    path := fmt.Sprintf("repos/%s/%s/issues", owner, repo)
    resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(issues, githubError)
    if err == nil {
        err = githubError
    }
    return *issues, resp, err
}
```

## Projects using Sling

* [cescoferraro/kala](https://github.com/cescoferraro/kala)
* [dghubble/go-twitter](https://github.com/dghubble/go-twitter)
* [dghubble/go-digits](https://github.com/dghubble/go-digits)
* [drinkin/go-gosquared](https://github.com/drinkin/go-gosquared)

Create a Pull Request to add a link to your own API.

## Motivation

Many client libraries follow the lead of [google/go-github](https://github.com/google/go-github) (our inspiration!), but do so by reimplementing logic common to all clients.

This project borrows and abstracts those ideas into a Sling, an agnostic component any API client can use for creating and sending requests.

## License

[MIT License](LICENSE)


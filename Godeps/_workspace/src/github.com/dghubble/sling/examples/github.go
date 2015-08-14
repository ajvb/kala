package main

import (
	"fmt"
	"github.com/dghubble/sling"
	"golang.org/x/oauth2"
	"net/http"
	"os"
)

const baseURL = "https://api.github.com/"

// Define models

// Issue is a simplified Github issue
// https://developer.github.com/v3/issues/#response
type Issue struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// GithubError represents a Github API error response
// https://developer.github.com/v3/#client-errors
type GithubError struct {
	Message string `json:"message"`
	Errors  []struct {
		Resource string `json:"resource"`
		Field    string `json:"field"`
		Code     string `json:"code"`
	} `json:"errors"`
	DocumentationURL string `json:"documentation_url"`
}

func (e GithubError) Error() string {
	return fmt.Sprintf("github: %v %+v %v", e.Message, e.Errors, e.DocumentationURL)
}

// IssueRequest is a simplified issue request
// https://developer.github.com/v3/issues/#create-an-issue
type IssueRequest struct {
	Title     string   `json:"title,omitempty"`
	Body      string   `json:"body,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Milestone int      `json:"milestone,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

// IssueListParams are the params for IssueService.List
// https://developer.github.com/v3/issues/#parameters
type IssueListParams struct {
	Filter    string `url:"filter,omitempty"`
	State     string `url:"state,omitempty"`
	Labels    string `url:"labels,omitempty"`
	Sort      string `url:"sort,omitempty"`
	Direction string `url:"direction,omitempty"`
	Since     string `url:"since,omitempty"`
}

// Implement services

// IssueService provides methods for creating and reading issues.
type IssueService struct {
	sling *sling.Sling
}

// NewIssueService returns a new IssueService.
func NewIssueService(httpClient *http.Client) *IssueService {
	return &IssueService{
		sling: sling.New().Client(httpClient).Base(baseURL),
	}
}

// List returns the authenticated user's issues across repos and orgs.
func (s *IssueService) List(params *IssueListParams) ([]Issue, *http.Response, error) {
	issues := new([]Issue)
	githubError := new(GithubError)
	resp, err := s.sling.New().Path("issues").QueryStruct(params).Receive(issues, githubError)
	if err == nil {
		err = githubError
	}
	return *issues, resp, err
}

// ListByRepo returns a repository's issues.
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

// Create creates a new issue on the specified repository.
func (s *IssueService) Create(owner, repo string, issueBody *IssueRequest) (*Issue, *http.Response, error) {
	issue := new(Issue)
	githubError := new(GithubError)
	path := fmt.Sprintf("repos/%s/%s/issues", owner, repo)
	resp, err := s.sling.New().Post(path).BodyJSON(issueBody).Receive(issue, githubError)
	if err == nil {
		err = githubError
	}
	return issue, resp, err
}

// (optional) Create a client to wrap services

// Client is a tiny Github client
type Client struct {
	IssueService *IssueService
	// other service endpoints...
}

// NewClient returns a new Client
func NewClient(httpClient *http.Client) *Client {
	return &Client{
		IssueService: NewIssueService(httpClient),
	}
}

func main() {

	// Github Unauthenticated API
	client := NewClient(nil)
	params := &IssueListParams{Sort: "updated"}
	issues, _, _ := client.IssueService.ListByRepo("golang", "go", params)
	fmt.Printf("Public golang/go Issues:\n%v\n", issues)

	// Github OAuth2 Example - httpClient handles authorization
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	if accessToken == "" {
		fmt.Println("Run 'export GITHUB_ACCESS_TOKEN=mytoken' and retry to list your public/private issues")
		os.Exit(0)
	}

	config := &oauth2.Config{}
	token := &oauth2.Token{AccessToken: accessToken}
	httpClient := config.Client(oauth2.NoContext, token)

	client = NewClient(httpClient)
	issues, _, _ = client.IssueService.List(params)
	fmt.Printf("Your Github Issues:\n%v\n", issues)

	// body := &IssueRequest{
	// 	Title: "Test title",
	// 	Body:  "Some test issue",
	// }
	// issue, _, _ := client.IssueService.Create("dghubble", "temp", body)
	// fmt.Println(issue)
}

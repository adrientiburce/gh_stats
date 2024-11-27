package model

// PullRequest struct to represent a single PR
type PullRequest struct {
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

// Repository struct with PRs
type Repository struct {
	Name              string   `json:"name"`
	FullName          string   `json:"full_name"`
	Description       string   `json:"description"`
	Stargazers        int      `json:"stargazers_count"`
	Topics            []string `json:"topics"`
	PushedAt          string   `json:"pushed_at"`
	HTMLURL           string   `json:"html_url"`
	Team              string
	PullRequests      []PullRequest
	RecentCommits     []CommitRes
	RecentDeployments []Deployment
	CILink            string
}

// FullCommit is the object we get from /commits API
type FullCommit struct {
	Commit  Commit `json:"commit"`
	HTMLURL string `json:"html_url"`
}

// Author of a commit
type Author struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

// Sub object from the commit API
type Commit struct {
	Message string `json:"message"`
	Author  Author `json:"author"`
}

// CommitRes is returned to the front
type CommitRes struct {
	Message string
	Date    string
	Author  string
	Link    string
}

// Deployment from github API
type Deployment struct {
	CreatedAt   string `json:"created_at"`
	Environment string `json:"environment"`
	Stack       string
}

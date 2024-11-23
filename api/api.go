package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PullRequest struct to represent a single PR
type PullRequest struct {
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

// Repository struct with PRs
type Repository struct {
	Name              string        `json:"name"`
	FullName          string        `json:"full_name"`
	Description       string        `json:"description"`
	Stargazers        int           `json:"stargazers_count"`
	PullRequests      []PullRequest `json:"pull_requests"`
	Topics            []string      `json:"topics"`
	PushedAt          string        `json:"pushed_at"`
	Team              string
	RecentCommits     []CommitRes
	RecentDeployments []Deployment
}

type Author struct {
	Name string `json:"pushed_at"`
	Date string `json:"date"`
}

type Commit struct {
	Message string `json:"message"`
	Author  Author `json:"author"`
}

type CommitRes struct {
	Message string
	Date    string
	Author  string
}

type Deployment struct {
	CreatedAt   string `json:"created_at"`
	Environment string `json:"environment"`
}

const MAX_REPOS = 10

// GetTopRepositories fetches top repositories and their PRs
func GetTopRepositories() []Repository {
	reposEnv := os.Getenv("TOP_PROJECTS")
	if reposEnv == "" {
		return []Repository{}
	}

	var reposMutex sync.Mutex

	repoNames := strings.Split(reposEnv, ",")
	var repos []Repository
	var wg sync.WaitGroup

	for _, name := range repoNames {
		wg.Add(1)
		go fetchOneRepo(name, &repos, &reposMutex, &wg)

		// for now only top 10 repos
		if len(repos) >= MAX_REPOS {
			break
		}
	}

	wg.Wait()
	return repos
}

func fetchOneRepo(name string, repos *[]Repository, m *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	repo := fetchRepo(name, doGithubRequest)
	if repo != nil {
		repo.PullRequests = fetchPullRequests(name)
		repo.RecentCommits = fetchCommits(name)
		repo.RecentDeployments = fetchDeploy(name)
	}

	m.Lock()
	*repos = append(*repos, *repo)
	m.Unlock()
}

func doGithubRequest(repoName string, path string) (res *http.Response, err error) {
	apiURL := "https://api.github.com/repos/Glovo/" + repoName
	if path != "" {
		apiURL = apiURL + "/" + path
	}
	token := os.Getenv("GITHUB_TOKEN")

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if err == nil {
			err = errors.New(fmt.Sprintf("StatusCode not [200] but %d", resp.StatusCode))
		}
		return nil, err
	}

	return resp, err
}

func fetchRepo(name string, doGithubRequest func(repoName, path string) (*http.Response, error)) *Repository {
	resp, err := doGithubRequest(name, "")
	if err != nil {
		log.Printf("Github request failed for %s: %s", name, err)
		return nil
	}
	defer resp.Body.Close()

	var repo Repository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		log.Printf("Failed to parse repo %s: %v", name, err)
		return nil
	}

	var teamName string
	for _, topic := range repo.Topics {
		if strings.HasPrefix(topic, "t-") {
			teamName = strings.Replace(topic, "t-", "", 0)
		}
	}

	parsedPushed, _ := time.Parse(time.RFC3339, repo.PushedAt)
	repo.PushedAt = parsedPushed.Format("Jan 2")
	repo.Team = teamName
	return &repo
}

func fetchCommits(repoName string) []CommitRes {
	resp, err := doGithubRequest(repoName, "commits?per_page=15")
	if err != nil {
		log.Printf("Can't fetch commits for %s: %v", repoName, err)
		return []CommitRes{}
	}
	defer resp.Body.Close()

	var commits []CommitRes
	var recentCommits []Commit
	if err := json.NewDecoder(resp.Body).Decode(&recentCommits); err != nil {
		log.Printf("Failed to parse commits for %s: %v", repoName, err)
		return commits
	}

	// Convert to []Commit and limit to 5 commits from developers
	for _, c := range recentCommits {
		if c.Author.Name == "GlovoRobot" {
			break
		}

		parsedDate, _ := time.Parse(time.RFC3339, c.Author.Date)
		commits = append(commits, CommitRes{
			Message: c.Message,
			Author:  c.Author.Name,
			Date:    parsedDate.Format("Jan 2"),
		})

		if len(commits) >= 5 {
			break
		}
	}

	return commits
}

func fetchDeploy(repoName string) (deploy []Deployment) {
	resp, err := doGithubRequest(repoName, "deployments?per_page=10")
	if err != nil {
		log.Printf("Failed to fetch deploy for %s: %v", repoName, err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&deploy); err != nil {
		log.Printf("Failed to parse deploy for %s: %v", repoName, err)
		return
	}

	// Regex to extract the environment name within square brackets
	envRegex := regexp.MustCompile(`\[(.*?)\]`)

	for i, d := range deploy {
		parsedTime, _ := time.Parse(time.RFC3339, d.CreatedAt)
		deploy[i].CreatedAt = parsedTime.Format("Jan 2")

		matches := envRegex.FindStringSubmatch(d.Environment)
		if len(matches) > 1 {
			deploy[i].Environment = matches[1] // The captured group
		} else {
			deploy[i].Environment = d.Environment // Fallback to original
		}
	}

	return
}

func fetchPullRequests(repoName string) []PullRequest {
	resp, err := doGithubRequest(repoName, "pulls?per_page=10")
	if err != nil {
		log.Printf("Can't fetch PR for %s: %v", repoName, err)
		return []PullRequest{}
	}
	defer resp.Body.Close()

	var prs []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		log.Printf("Failed to parse PRs for %s: %v", repoName, err)
		return []PullRequest{}
	}

	// Format dates for display
	for i, pr := range prs {
		parsedTime, _ := time.Parse(time.RFC3339, pr.CreatedAt)
		prs[i].CreatedAt = parsedTime.Format("Jan 2")
	}

	return prs
}

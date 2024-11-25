package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Stargazers  int      `json:"stargazers_count"`
	Topics      []string `json:"topics"`
	PushedAt    string   `json:"pushed_at"`
	// HTMLURL           string        `json:"html_url"`
	Team              string
	PullRequests      []PullRequest
	RecentCommits     []CommitRes
	RecentDeployments []Deployment
	CILink            string
}

type Author struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type FullCommit struct {
	Commit  Commit `json:"commit"`
	HTMLURL string `json:"html_url"`
}

type Commit struct {
	Message string `json:"message"`
	Author  Author `json:"author"`
}

type CommitRes struct {
	Message string
	Date    string
	Author  string
	Link    string
}

type Deployment struct {
	CreatedAt   string `json:"created_at"`
	Environment string `json:"environment"`
	Stack       string
}

const MAX_REPOS = 10
const DEPLOYMENT_LIMITS = 3
const COMMITS_LIMITS = 5

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

func fetchOneRepo(name string, repos *[]Repository, m *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	repo := fetchRepo(name)
	if repo != nil {
		repo.PullRequests = fetchPullRequests(name)
		repo.RecentDeployments = fetchDeploy(name)
		repo.RecentCommits = fetchCommits(name)
	}

	repo.CILink = fmt.Sprintf("%s%s", os.Getenv("CI_BASE_TEMPLATE"), repo.Name)

	// doGithubRequest func(repoName, path string) (*http.Response, error)

	m.Lock()
	*repos = append(*repos, *repo)
	m.Unlock()
}

func fetchRepo(name string) *Repository {
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
	var recentCommits []FullCommit
	if err := json.NewDecoder(resp.Body).Decode(&recentCommits); err != nil {
		log.Printf("Failed to parse commits for %s: %v", repoName, err)
		return commits
	}

	commitFirstPart := regexp.MustCompile(`^(.*\s\(#\d+\))`)

	// Convert to []Commit and limit to 5 commits from developers
	for _, c := range recentCommits {
		if c.Commit.Author.Name == "GlovoRobot" {
			continue
		}

		message := c.Commit.Message
		matches := commitFirstPart.FindStringSubmatch(message)
		if len(matches) > 1 {
			message = matches[1] // Trim message to end with PR number
		}

		parsedDate, _ := time.Parse(time.RFC3339, c.Commit.Author.Date)
		commits = append(commits, CommitRes{
			Message: message,
			Author:  c.Commit.Author.Name,
			Date:    parsedDate.Format("Jan 2"),
			Link:    c.HTMLURL,
		})

		if len(commits) >= COMMITS_LIMITS {
			break
		}
	}

	return commits
}

func fetchDeploy(repoName string) (deploy []Deployment) {
	resp, err := doGithubRequest(repoName, "deployments?per_page="+strconv.Itoa(DEPLOYMENT_LIMITS))
	if err != nil {
		log.Printf("Failed to fetch deploy for %s: %v", repoName, err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&deploy); err != nil {
		log.Printf("Failed to parse deploy for %s: %v", repoName, err)
		return
	}

	// Regex to extract the environment name within square brackets and the stack name
	envRegex := regexp.MustCompile(`\[(.*?)\]\s+(.*)`)

	for i, d := range deploy {
		parsedTime, _ := time.Parse(time.RFC3339, d.CreatedAt)
		deploy[i].CreatedAt = parsedTime.Format("Jan 2 15:04:05")

		matches := envRegex.FindStringSubmatch(d.Environment)
		if len(matches) > 2 {
			deploy[i].Environment = matches[1] // Extracted environment (e.g., "stage")
			deploy[i].Stack = matches[2]       // Extracted stack (e.g., "customer-profile-events")
		} else {
			// Fallback for cases where regex doesn't match
			deploy[i].Environment = d.Environment
			deploy[i].Stack = "Unknown Stack"
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

	fmt.Printf("fetched %d PR from: %s \n", len(prs), repoName)

	return prs
}

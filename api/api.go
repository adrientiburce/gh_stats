package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"gh_stats/model"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MAX_REPOS = 10
const DEPLOYMENT_LIMITS = 3
const COMMITS_LIMITS = 5

type GithubClient interface {
	DoGithubRequest(repoName, path string) (res *http.Response, err error)
}

// GetTopRepositories fetches top repositories and their PRs
func GetTopRepositories() []model.Repository {
	reposEnv := os.Getenv("TOP_PROJECTS")
	if reposEnv == "" {
		return []model.Repository{}
	}

	var reposMutex sync.Mutex

	repoNames := strings.Split(reposEnv, ",")
	var repos []model.Repository
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

	return reorderRepos(repos, repoNames)
}

func fetchOneRepo(name string, repos *[]model.Repository, m *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &RealGithubClient{}
	repo := client.FetchRepo(name)
	if repo != nil {
		repo.PullRequests = client.FetchPullRequests(name)
		repo.RecentDeployments = client.FetchDeploy(name)
		repo.RecentCommits = client.FetchCommits(name)
	}
	repo.CILink = fmt.Sprintf("%s%s", os.Getenv("CI_BASE_TEMPLATE"), repo.Name)

	m.Lock()
	*repos = append(*repos, *repo)
	m.Unlock()
}

type RealGithubClient struct {
}

func (c *RealGithubClient) DoGithubRequest(repoName, path string) (res *http.Response, err error) {
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

func (c *RealGithubClient) FetchRepo(name string) *model.Repository {
	resp, err := c.DoGithubRequest(name, "")
	if err != nil {
		log.Printf("Github request failed for %s: %s", name, err)
		return nil
	}
	defer resp.Body.Close()

	var repo model.Repository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		log.Printf("Failed to parse repo %s: %v", name, err)
		return nil
	}

	var teamName string
	for _, topic := range repo.Topics {
		if strings.HasPrefix(topic, "t-") {
			teamName = strings.Replace(topic, "t-", "", 1)
		}
	}

	parsedPushed, _ := time.Parse(time.RFC3339, repo.PushedAt)
	repo.PushedAt = parsedPushed.Format("Jan 2")
	repo.Team = teamName
	return &repo
}

func (c *RealGithubClient) FetchCommits(repoName string) []model.CommitRes {
	resp, err := c.DoGithubRequest(repoName, "commits?per_page=15")
	if err != nil {
		log.Printf("Can't fetch commits for %s: %v", repoName, err)
		return []model.CommitRes{}
	}
	defer resp.Body.Close()

	var commits []model.CommitRes
	var recentCommits []model.FullCommit
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
		commits = append(commits, model.CommitRes{
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

func (c *RealGithubClient) FetchDeploy(repoName string) (deploy []model.Deployment) {
	resp, err := c.DoGithubRequest(repoName, "deployments?per_page="+strconv.Itoa(DEPLOYMENT_LIMITS))
	if err != nil {
		log.Printf("Failed to fetch deploy for %s: %v", repoName, err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&deploy); err != nil {
		log.Printf("Failed to parse deploy for %s: %v", repoName, err)
		return
	}

	for i, d := range deploy {
		parsedTime, _ := time.Parse(time.RFC3339, d.CreatedAt)
		deploy[i].CreatedAt = parsedTime.Format("Jan 2 15:04:05")

		env, stack := extractStack(d.Environment)
		deploy[i].Environment = env
		deploy[i].Stack = stack
	}

	return
}

func (c *RealGithubClient) FetchPullRequests(repoName string) []model.PullRequest {
	resp, err := c.DoGithubRequest(repoName, "pulls?per_page=10")
	if err != nil {
		log.Printf("Can't fetch PR for %s: %v", repoName, err)
		return []model.PullRequest{}
	}
	defer resp.Body.Close()

	var prs []model.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		log.Printf("Failed to parse PRs for %s: %v", repoName, err)
		return []model.PullRequest{}
	}

	// Format dates for display
	for i, pr := range prs {
		parsedTime, _ := time.Parse(time.RFC3339, pr.CreatedAt)
		prs[i].CreatedAt = parsedTime.Format("Jan 2")
	}

	fmt.Printf("fetched %d PR from: %s \n", len(prs), repoName)

	return prs
}

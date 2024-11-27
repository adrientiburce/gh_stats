package api

import (
	"gh_stats/model"
	"regexp"
)

func reorderRepos(repos []model.Repository, names []string) []model.Repository {
	// Create a map for quick lookup
	repoMap := make(map[string]model.Repository, len(repos))
	for _, repo := range repos {
		repoMap[repo.Name] = repo
	}

	// Reorder repos based on repoNames
	var orderedRepos []model.Repository
	for _, name := range names {
		if repo, exists := repoMap[name]; exists {
			orderedRepos = append(orderedRepos, repo)
		}
	}
	return orderedRepos
}

func extractStack(envstack string) (env, stack string) {
	// Regex to extract the environment name within square brackets and the stack name
	envRegex := regexp.MustCompile(`\[(.*?)\]\s+(.*)`)
	matches := envRegex.FindStringSubmatch(envstack)
	if len(matches) > 2 {
		return matches[1], matches[2]
	} else {
		return envstack, "unknown"
	}
}

func extractCommitMessage(message string) (res string) {
	commitFirstPart := regexp.MustCompile(`^(.*\s\(#\d+\))`)
	res = message
	matches := commitFirstPart.FindStringSubmatch(message)

	if len(matches) > 1 {
		res = matches[1] // Trim message to end with PR number
	}

	return
}

package api

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"gh_stats/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGithubClient is a mock implementation of GithubClient
type MockGithubClient struct {
	mock.Mock
}

func (m *MockGithubClient) DoGithubRequest(repoName, path string) (res *http.Response, err error) {
	args := m.Called(repoName, path)
	return args.Get(0).(*http.Response), args.Error(1)
}

// LoadMockResponse loads mock JSON from a file
func LoadMockResponse(t *testing.T, filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read mock response file: %v", err)
	}
	return string(data)
}

func TestFetchRepo(t *testing.T) {
	// Prepare test cases
	testCases := []struct {
		name          string
		mockJSON      string
		expectedRepo  *model.Repository
		expectedError bool
	}{
		{
			name: "Successful repository fetch",
			mockJSON: `{
				"name": "test-repo",
				"full_name": "Glovo/test-repo",
				"description": "A test repository",
				"pushed_at": "2023-06-15T10:30:00Z",
				"topics": ["t-backend", "golang"]
			}`,
			expectedRepo: &model.Repository{
				Name:        "test-repo",
				FullName:    "Glovo/test-repo",
				Description: "A test repository",
				PushedAt:    "Jun 15",
				Team:        "backend",
				Topics:      []string{"t-backend", "golang"},
			},
			expectedError: false,
		},
		{
			name: "Repository with no team topic",
			mockJSON: `{
				"name": "no-team-repo",
				"full_name": "Glovo/no-team-repo",
				"description": "A repo without team topic",
				"pushed_at": "2023-07-20T15:45:00Z",
				"topics": ["golang", "library"]
			}`,
			expectedRepo: &model.Repository{
				Name:        "no-team-repo",
				FullName:    "Glovo/no-team-repo",
				Description: "A repo without team topic",
				PushedAt:    "Jul 20",
				Team:        "",
				Topics:      []string{"golang", "library"},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock HTTP response
			mockResponse := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(tc.mockJSON)),
			}

			// Create mock GitHub client
			mockClient := new(MockGithubClient)
			mockClient.On("DoGithubRequest", mock.Anything, "").Return(mockResponse, nil)

			repo := FetchRepo("test-repo", mockClient)

			// Assert expectations
			if tc.expectedError {
				assert.Nil(t, repo)
			} else {
				assert.NotNil(t, repo)
				assert.Equal(t, tc.expectedRepo.Name, repo.Name)
				assert.Equal(t, tc.expectedRepo.FullName, repo.FullName)
				assert.Equal(t, tc.expectedRepo.Description, repo.Description)
				assert.Equal(t, tc.expectedRepo.PushedAt, repo.PushedAt)
				assert.Equal(t, tc.expectedRepo.Team, repo.Team)
				assert.ElementsMatch(t, tc.expectedRepo.Topics, repo.Topics)
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}

func TestFetchCommits(t *testing.T) {
	testCases := []struct {
		name            string
		mockPath        string
		repoName        string
		expectedCommits int
	}{
		{
			name:            "Successful commits fetch with multiple commits",
			mockPath:        "../testdata/mock_commits.json",
			repoName:        "test-repo",
			expectedCommits: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			commitsJson := LoadMockResponse(t, tc.mockPath)
			// Create mock HTTP response
			mockResponse := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(commitsJson)),
			}

			// Create mock GitHub client
			mockClient := new(MockGithubClient)
			mockClient.On("DoGithubRequest", tc.repoName, "commits?per_page=15").Return(mockResponse, nil)

			// Call FetchCommits
			commits := FetchCommits(tc.repoName, mockClient)

			// Assert expectations
			assert.Len(t, commits, tc.expectedCommits)

			// Verify specific commit details
			if len(commits) > 0 {
				assert.Equal(t, "Feature: Add new functionality (#123)", commits[0].Message)
				assert.Equal(t, "John Doe", commits[0].Author)
				assert.Equal(t, "Jan 15", commits[0].Date)
				assert.Equal(t, "https://github.com/Glovo/repo/commit/sha1", commits[0].Link)
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}

	t.Run("Failed HTTP request", func(t *testing.T) {
		// Create mock GitHub client that returns an error
		mockClient := new(MockGithubClient)
		mockClient.On("DoGithubRequest", "error-repo", "commits?per_page=15").Return(
			&http.Response{},
			assert.AnError,
		)

		// Call FetchCommits
		commits := FetchCommits("error-repo", mockClient)

		// Assert empty slice for failed request
		assert.Empty(t, commits)

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		// Create mock HTTP response with invalid JSON
		mockResponse := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
		}

		// Create mock GitHub client
		mockClient := new(MockGithubClient)
		mockClient.On("DoGithubRequest", "invalid-repo", "commits?per_page=15").Return(mockResponse, nil)

		// Call FetchCommits
		commits := FetchCommits("invalid-repo", mockClient)

		// Assert empty slice for parsing error
		assert.Empty(t, commits)

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})
}

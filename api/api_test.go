package api

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock fetchRepo implementation for test
func mockFetchRepoServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

// LoadMockResponse loads mock JSON from a file
func LoadMockResponse(t *testing.T, filePath string) string {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read mock response file: %v", err)
	}
	return string(data)
}

func TestFetchRepo(t *testing.T) {
	mockResponse := LoadMockResponse(t, "../testdata/mock_repo.json")

	// Create a mock server
	server := mockFetchRepoServer(mockResponse)
	defer server.Close()

	// Update doGithubRequest to point to the mock server
	// originalDoGithubRequest := doGithubRequest

	mockFunc := func(repoName, path string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	}

	// Call the method to test
	repo := fetchRepo("mock-repo", mockFunc)

	// Assertions
	if repo == nil {
		t.Fatalf("Expected repo, got nil")
	}

	if repo.Name != "customer-profile" {
		t.Errorf("Expected Name to be 'customer-profile', got '%s'", repo.Name)
	}

	if repo.Description != "Mock description" {
		t.Errorf("Expected Description to be 'Mock description', got '%s'", repo.Description)
	}

	if repo.PushedAt != "Nov 22" { // Assuming date formatting happens in fetchRepo
		t.Errorf("Expected PushedAt to be 'Nov 22', got '%s'", repo.PushedAt)
	}

	// if repo.Language != "Kotlin" {
	// 	t.Errorf("Expected Language to be 'Kotlin', got '%s'", repo.Language)
	// }
}

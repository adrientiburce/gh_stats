package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	// Replace with your actual module path
)

// Mock fetchRepo server implementation
func mockFetchRepoServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

// LoadMockResponse loads mock JSON from a file
func LoadMockResponse(t *testing.T, filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read mock response file: %v", err)
	}
	return string(data)
}

// MockGithubClient is a mock implementation of the GithubClient interface
type MockGithubClient struct {
	MockResponse string
}

func (m *MockGithubClient) DoGithubRequest(repoName, path string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(m.MockResponse)),
	}, nil
}

func TestFetchRepo(t *testing.T) {
	mockResponse := LoadMockResponse(t, "../testdata/mock_repo.json")

	// Use the mock client with the prepared response
	mockClient := &MockGithubClient{MockResponse: mockResponse}
	// mock DoGithubRequest method to use the mockClient

	repo := mockClient.DoGithubRequest()
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

}

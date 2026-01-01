package fetchers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/farhapartex/search-proxy/internal/models"
)

// GitHubFetcher fetches search results from GitHub
type GitHubFetcher struct {
	apiToken string
	baseURL  string
	client   *http.Client
}

// NewGitHubFetcher creates a new GitHub fetcher
func NewGitHubFetcher(apiToken, baseURL string) *GitHubFetcher {
	return &GitHubFetcher{
		apiToken: apiToken,
		baseURL:  baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the platform name
func (g *GitHubFetcher) Name() string {
	return "github"
}

// Fetch retrieves search results from GitHub
func (g *GitHubFetcher) Fetch(ctx context.Context, query string, maxResults int) ([]*models.SearchResult, error) {
	// Build search URL
	searchURL := fmt.Sprintf("%s/search/repositories?q=%s&per_page=%d&sort=stars&order=desc",
		g.baseURL,
		url.QueryEscape(query),
		maxResults,
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	//req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiToken))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Parse response
	var githubResp GitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&githubResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to internal format
	results := make([]*models.SearchResult, 0, len(githubResp.Items))
	for _, item := range githubResp.Items {
		result := models.NewSearchResult(
			"github",
			item.FullName,
			item.Description,
			item.HTMLURL,
		)
		result.Timestamp = item.CreatedAt.Unix()
		result.Metadata = map[string]string{
			"stars":       fmt.Sprintf("%d", item.StargazersCount),
			"forks":       fmt.Sprintf("%d", item.ForksCount),
			"language":    item.Language,
			"open_issues": fmt.Sprintf("%d", item.OpenIssuesCount),
		}
		results = append(results, result)
	}

	return results, nil
}

// GitHubSearchResponse represents the GitHub API search response
type GitHubSearchResponse struct {
	TotalCount int                `json:"total_count"`
	Items      []GitHubRepository `json:"items"`
}

// GitHubRepository represents a GitHub repository in search results
type GitHubRepository struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	HTMLURL         string    `json:"html_url"`
	StargazersCount int       `json:"stargazers_count"`
	ForksCount      int       `json:"forks_count"`
	Language        string    `json:"language"`
	OpenIssuesCount int       `json:"open_issues_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TruncateString truncates a string to maxLength and adds "..." if truncated
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return strings.TrimSpace(s[:maxLength-3]) + "..."
}

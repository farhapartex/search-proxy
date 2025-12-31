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

// StackOverflowFetcher fetches search results from StackOverflow
type StackOverflowFetcher struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewStackOverflowFetcher creates a new StackOverflow fetcher
func NewStackOverflowFetcher(apiKey, baseURL string) *StackOverflowFetcher {
	return &StackOverflowFetcher{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the platform name
func (s *StackOverflowFetcher) Name() string {
	return "stackoverflow"
}

// Fetch retrieves search results from StackOverflow
func (s *StackOverflowFetcher) Fetch(ctx context.Context, query string, maxResults int) ([]*models.SearchResult, error) {
	// Build search URL
	searchURL := fmt.Sprintf("%s/search/advanced?q=%s&pagesize=%d&order=desc&sort=relevance&site=stackoverflow",
		s.baseURL,
		url.QueryEscape(query),
		maxResults,
	)

	// Add API key if available
	if s.apiKey != "" {
		searchURL += fmt.Sprintf("&key=%s", s.apiKey)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("StackOverflow API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Parse response
	var soResp StackOverflowSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&soResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to internal format
	results := make([]*models.SearchResult, 0, len(soResp.Items))
	for _, item := range soResp.Items {
		// Create snippet from title and tags
		snippet := item.Title
		if len(item.Tags) > 0 {
			snippet += " | Tags: " + strings.Join(item.Tags, ", ")
		}

		result := models.NewSearchResult(
			"stackoverflow",
			item.Title,
			TruncateString(snippet, 500),
			item.Link,
		)
		result.Timestamp = item.CreationDate
		result.Metadata = map[string]string{
			"score":        fmt.Sprintf("%d", item.Score),
			"answer_count": fmt.Sprintf("%d", item.AnswerCount),
			"view_count":   fmt.Sprintf("%d", item.ViewCount),
			"is_answered":  fmt.Sprintf("%t", item.IsAnswered),
			"tags":         strings.Join(item.Tags, ","),
		}
		results = append(results, result)
	}

	return results, nil
}

// StackOverflowSearchResponse represents the StackOverflow API search response
type StackOverflowSearchResponse struct {
	Items          []StackOverflowQuestion `json:"items"`
	HasMore        bool                    `json:"has_more"`
	QuotaMax       int                     `json:"quota_max"`
	QuotaRemaining int                     `json:"quota_remaining"`
}

// StackOverflowQuestion represents a StackOverflow question in search results
type StackOverflowQuestion struct {
	QuestionID   int      `json:"question_id"`
	Title        string   `json:"title"`
	Link         string   `json:"link"`
	Score        int      `json:"score"`
	AnswerCount  int      `json:"answer_count"`
	ViewCount    int      `json:"view_count"`
	IsAnswered   bool     `json:"is_answered"`
	Tags         []string `json:"tags"`
	CreationDate int64    `json:"creation_date"`
}

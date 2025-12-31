package fetchers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/farhapartex/search-proxy/internal/models"
)

// RedditFetcher fetches search results from Reddit
type RedditFetcher struct {
	clientID     string
	clientSecret string
	userAgent    string
	baseURL      string
	client       *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// NewRedditFetcher creates a new Reddit fetcher
func NewRedditFetcher(clientID, clientSecret, userAgent, baseURL string) *RedditFetcher {
	return &RedditFetcher{
		clientID:     clientID,
		clientSecret: clientSecret,
		userAgent:    userAgent,
		baseURL:      baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the platform name
func (r *RedditFetcher) Name() string {
	return "reddit"
}

// Fetch retrieves search results from Reddit
func (r *RedditFetcher) Fetch(ctx context.Context, query string, maxResults int) ([]*models.SearchResult, error) {
	// For simplicity, use the public JSON endpoint (no OAuth required)
	// This works without authentication but has lower rate limits
	searchURL := fmt.Sprintf("https://www.reddit.com/search.json?q=%s&limit=%d&sort=relevance",
		url.QueryEscape(query),
		maxResults,
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Reddit API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Parse response
	var redditResp RedditSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to internal format
	results := make([]*models.SearchResult, 0, len(redditResp.Data.Children))
	for _, child := range redditResp.Data.Children {
		post := child.Data

		// Build Reddit URL
		postURL := fmt.Sprintf("https://www.reddit.com%s", post.Permalink)

		// Create snippet from selftext or title
		snippet := post.Selftext
		if snippet == "" {
			snippet = post.Title
		}

		result := models.NewSearchResult(
			"reddit",
			post.Title,
			TruncateString(snippet, 500),
			postURL,
		)
		result.Timestamp = int64(post.CreatedUTC)
		result.Metadata = map[string]string{
			"score":         fmt.Sprintf("%d", post.Score),
			"num_comments":  fmt.Sprintf("%d", post.NumComments),
			"subreddit":     post.Subreddit,
			"author":        post.Author,
			"upvote_ratio":  fmt.Sprintf("%.2f", post.UpvoteRatio),
		}
		results = append(results, result)
	}

	return results, nil
}

// RedditSearchResponse represents the Reddit API search response
type RedditSearchResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After    string         `json:"after"`
		Children []RedditChild  `json:"children"`
	} `json:"data"`
}

// RedditChild represents a child item in Reddit response
type RedditChild struct {
	Kind string     `json:"kind"`
	Data RedditPost `json:"data"`
}

// RedditPost represents a Reddit post in search results
type RedditPost struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Selftext     string  `json:"selftext"`
	Author       string  `json:"author"`
	Subreddit    string  `json:"subreddit"`
	Score        int     `json:"score"`
	NumComments  int     `json:"num_comments"`
	CreatedUTC   float64 `json:"created_utc"`
	Permalink    string  `json:"permalink"`
	URL          string  `json:"url"`
	UpvoteRatio  float64 `json:"upvote_ratio"`
}

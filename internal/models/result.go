package models

import (
	"time"

	pb "github.com/farhapartex/search-proxy/proto"
)

// SearchResult represents an internal search result
// This is the unified structure used internally before converting to protobuf
type SearchResult struct {
	Platform  string
	Title     string
	Snippet   string
	URL       string
	Timestamp int64
	Metadata  map[string]string
}

// NewSearchResult creates a new SearchResult
func NewSearchResult(platform, title, snippet, url string) *SearchResult {
	return &SearchResult{
		Platform:  platform,
		Title:     title,
		Snippet:   snippet,
		URL:       url,
		Timestamp: time.Now().Unix(),
		Metadata:  make(map[string]string),
	}
}

// ToProto converts SearchResult to protobuf Result
func (r *SearchResult) ToProto() *pb.Result {
	return &pb.Result{
		Platform:  r.Platform,
		Title:     r.Title,
		Snippet:   r.Snippet,
		Url:       r.URL,
		Timestamp: r.Timestamp,
		Metadata:  r.Metadata,
	}
}

// FetchResult represents the result of a platform fetch operation
type FetchResult struct {
	Platform string
	Results  []*SearchResult
	Error    error
	Duration time.Duration
	TimedOut bool
}

// NewFetchResult creates a new FetchResult
func NewFetchResult(platform string) *FetchResult {
	return &FetchResult{
		Platform: platform,
		Results:  make([]*SearchResult, 0),
	}
}

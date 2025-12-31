package fetchers

import (
	"context"

	"github.com/farhapartex/search-proxy/internal/models"
)

// Fetcher is the interface that all platform fetchers must implement
type Fetcher interface {
	// Fetch retrieves search results from the platform
	// ctx: context with timeout
	// query: search query string
	// maxResults: maximum number of results to return
	Fetch(ctx context.Context, query string, maxResults int) ([]*models.SearchResult, error)

	// Name returns the platform name
	Name() string
}

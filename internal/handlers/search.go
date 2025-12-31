package handlers

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/farhapartex/search-proxy/internal/config"
	"github.com/farhapartex/search-proxy/internal/fetchers"
	"github.com/farhapartex/search-proxy/internal/models"
	pb "github.com/farhapartex/search-proxy/proto"
)

// SearchHandler orchestrates concurrent searches across multiple platforms
type SearchHandler struct {
	fetchers map[string]fetchers.Fetcher
	config   *config.Config
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(cfg *config.Config) *SearchHandler {
	handler := &SearchHandler{
		fetchers: make(map[string]fetchers.Fetcher),
		config:   cfg,
	}

	// Initialize fetchers
	handler.fetchers["github"] = fetchers.NewGitHubFetcher(
		cfg.GitHub.APIToken,
		cfg.GitHub.BaseURL,
	)
	handler.fetchers["stackoverflow"] = fetchers.NewStackOverflowFetcher(
		cfg.StackOverflow.APIKey,
		cfg.StackOverflow.BaseURL,
	)
	handler.fetchers["reddit"] = fetchers.NewRedditFetcher(
		cfg.Reddit.ClientID,
		cfg.Reddit.ClientSecret,
		cfg.Reddit.UserAgent,
		cfg.Reddit.BaseURL,
	)

	return handler
}

// Search performs a federated search using the Fan-out/Fan-in pattern
func (h *SearchHandler) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	startTime := time.Now()

	// Determine which platforms to query
	platforms := req.Platforms
	if len(platforms) == 0 {
		// Default to all platforms
		platforms = []string{"github", "stackoverflow", "reddit"}
	}

	// Determine max results
	maxResults := int(req.MaxResults)
	if maxResults <= 0 || maxResults > 100 {
		maxResults = h.config.Performance.MaxResultsPerPlatform
	}

	// Create a channel to collect results from all goroutines
	resultsChan := make(chan *models.FetchResult, len(platforms))

	// Create a WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Fan-out: Launch a goroutine for each platform
	for _, platform := range platforms {
		fetcher, exists := h.fetchers[platform]
		if !exists {
			log.Printf("WARNING: Unknown platform: %s", platform)
			continue
		}

		wg.Add(1)
		go h.fetchFromPlatform(ctx, fetcher, req.Query, maxResults, resultsChan, &wg)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Fan-in: Collect results from all platforms
	var allResults []*pb.Result
	var platformsSuccess []string
	var platformsTimeout []string
	var platformsError []string

	for fetchResult := range resultsChan {
		if fetchResult.Error != nil {
			if fetchResult.TimedOut {
				platformsTimeout = append(platformsTimeout, fetchResult.Platform)
				log.Printf("Platform %s timed out: %v", fetchResult.Platform, fetchResult.Error)
			} else {
				platformsError = append(platformsError, fetchResult.Platform)
				log.Printf("Platform %s error: %v", fetchResult.Platform, fetchResult.Error)
			}
			continue
		}

		// Success
		platformsSuccess = append(platformsSuccess, fetchResult.Platform)
		log.Printf("Platform %s returned %d results in %v",
			fetchResult.Platform, len(fetchResult.Results), fetchResult.Duration)

		// Convert results to protobuf
		for _, result := range fetchResult.Results {
			allResults = append(allResults, result.ToProto())
		}
	}

	// Calculate response time
	responseTime := time.Since(startTime)

	// Build response
	response := &pb.SearchResponse{
		Results:           allResults,
		TotalCount:        int32(len(allResults)),
		PlatformsSuccess:  platformsSuccess,
		PlatformsTimeout:  platformsTimeout,
		PlatformsError:    platformsError,
		Metadata: &pb.ResponseMetadata{
			ResponseTimeMs:  int32(responseTime.Milliseconds()),
			PlatformsQueried: int32(len(platforms)),
		},
	}

	log.Printf("Search completed in %v. Total results: %d (Success: %d, Timeout: %d, Error: %d)",
		responseTime, len(allResults), len(platformsSuccess), len(platformsTimeout), len(platformsError))

	return response, nil
}

// fetchFromPlatform fetches results from a single platform (runs in goroutine)
func (h *SearchHandler) fetchFromPlatform(
	parentCtx context.Context,
	fetcher fetchers.Fetcher,
	query string,
	maxResults int,
	resultsChan chan<- *models.FetchResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	startTime := time.Now()
	result := models.NewFetchResult(fetcher.Name())

	// Create a context with timeout for this specific platform
	ctx, cancel := context.WithTimeout(parentCtx, h.config.Server.PerAPITimeout)
	defer cancel()

	// Fetch results
	results, err := fetcher.Fetch(ctx, query, maxResults)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = err
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
		}
	} else {
		result.Results = results
	}

	// Send result to channel
	resultsChan <- result
}

package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig
	GitHub    GitHubConfig
	StackOverflow StackOverflowConfig
	Reddit    RedditConfig
	Performance PerformanceConfig
	Logging   LoggingConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	GRPCPort       string
	ServerTimeout  time.Duration
	PerAPITimeout  time.Duration
}

// GitHubConfig holds GitHub API configuration
type GitHubConfig struct {
	APIToken  string
	BaseURL   string
}

// StackOverflowConfig holds StackOverflow API configuration
type StackOverflowConfig struct {
	APIKey  string
	BaseURL string
}

// RedditConfig holds Reddit API configuration
type RedditConfig struct {
	ClientID     string
	ClientSecret string
	UserAgent    string
	BaseURL      string
}

// PerformanceConfig holds performance tuning configuration
type PerformanceConfig struct {
	MaxResultsPerPlatform int
	EnableCircuitBreaker  bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			GRPCPort:      getEnv("GRPC_SERVER_PORT", "50051"),
			ServerTimeout: getDurationEnv("SERVER_TIMEOUT_MS", 500) * time.Millisecond,
			PerAPITimeout: getDurationEnv("PER_API_TIMEOUT_MS", 400) * time.Millisecond,
		},
		GitHub: GitHubConfig{
			APIToken: getEnv("GITHUB_API_TOKEN", ""),
			BaseURL:  getEnv("GITHUB_API_BASE_URL", "https://api.github.com"),
		},
		StackOverflow: StackOverflowConfig{
			APIKey:  getEnv("STACKOVERFLOW_API_KEY", ""),
			BaseURL: getEnv("STACKOVERFLOW_API_BASE_URL", "https://api.stackexchange.com/2.3"),
		},
		Reddit: RedditConfig{
			ClientID:     getEnv("REDDIT_CLIENT_ID", ""),
			ClientSecret: getEnv("REDDIT_CLIENT_SECRET", ""),
			UserAgent:    getEnv("REDDIT_USER_AGENT", "FederatedSearchEngine/1.0"),
			BaseURL:      getEnv("REDDIT_API_BASE_URL", "https://oauth.reddit.com"),
		},
		Performance: PerformanceConfig{
			MaxResultsPerPlatform:   getIntEnv("MAX_RESULTS_PER_PLATFORM", 20),
			EnableCircuitBreaker:    getBoolEnv("ENABLE_CIRCUIT_BREAKER", true),
			CircuitBreakerThreshold: getIntEnv("CIRCUIT_BREAKER_THRESHOLD", 5),
			CircuitBreakerTimeout:   getDurationEnv("CIRCUIT_BREAKER_TIMEOUT_SEC", 30) * time.Second,
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if required configuration fields are set
func (c *Config) Validate() error {
	// GitHub token is optional (but recommended for higher rate limits)
	if c.GitHub.APIToken == "" {
		log.Println("WARNING: GITHUB_API_TOKEN not set. Rate limit: 60 requests/hour")
	}

	// StackOverflow key is optional
	if c.StackOverflow.APIKey == "" {
		log.Println("WARNING: STACKOVERFLOW_API_KEY not set. Rate limit: 300 requests/day")
	}

	// Reddit credentials are optional
	if c.Reddit.ClientID == "" || c.Reddit.ClientSecret == "" {
		log.Println("WARNING: REDDIT_CLIENT_ID or REDDIT_CLIENT_SECRET not set. Using unauthenticated access")
	}

	return nil
}

// Helper functions to get environment variables with defaults

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getIntEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("WARNING: Invalid integer value for %s: %s. Using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}

	return value
}

func getBoolEnv(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("WARNING: Invalid boolean value for %s: %s. Using default: %t", key, valueStr, defaultValue)
		return defaultValue
	}

	return value
}

func getDurationEnv(key string, defaultValue int) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return time.Duration(defaultValue)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("WARNING: Invalid duration value for %s: %s. Using default: %d", key, valueStr, defaultValue)
		return time.Duration(defaultValue)
	}

	return time.Duration(value)
}

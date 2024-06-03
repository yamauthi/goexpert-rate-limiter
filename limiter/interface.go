package limiter

import (
	"errors"
	"time"
)

const (
	CHECK_IP_ONLY       = iota // 0
	CHECK_API_KEY_ONLY  = iota // 1
	CHECK_IP_OR_API_KEY = iota // 2
)

var ErrApiKeyNotFound = errors.New("the provided api key was not found")
var ErrInvalidClient = errors.New("the provided client is invalid")
var ErrMaxNumberRequestsReached = errors.New("you have reached the maximum number of requests or actions allowed within a certain time frame")

type LimiterConfig struct {
	ClientCheckType       int
	ClientBlockTime       time.Duration
	MaxIPRequests         int
	RequestsLimitInterval time.Duration
}

type APIKey struct {
	ID          string
	MaxRequests int
}

// Client represents a client request information
type Client struct {
	// ID is the client IP or API Key
	ID string

	// CurrentRequests is the requests amount made
	CurrentRequests int

	// TTL is the interval time that this entry will be considered
	TTL time.Duration

	// Reports whether the client is blocked to make requests
	Blocked bool
}

type LimiterRepositoryInterface interface {
	ApiKey(id string) *APIKey
	Client(id string) *Client
	SaveApiKey(apiKey APIKey)
	SaveClient(client Client)
}

type RateLimiterInterface interface {
	AllowRequest(clientID, apiKeyID string) (bool, error)
}

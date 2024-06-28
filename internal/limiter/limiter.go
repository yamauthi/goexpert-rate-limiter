package limiter

import (
	"log"
	"time"
)

const REQUESTS_PER_SECOND = time.Second

// Limiter is a implementation of `RateLimiter interface`
type Limiter struct {
	Config     LimiterConfig
	Repository LimiterRepositoryInterface
}

func NewLimiter(
	conf LimiterConfig,
	repository LimiterRepositoryInterface,
) *Limiter {
	return &Limiter{
		Config:     conf,
		Repository: repository,
	}
}

func (l *Limiter) AllowRequest(clientID, apiKeyID string) (bool, error) {
	switch l.Config.ClientCheckType {
	case CHECK_IP_ONLY:
		return l.checkClientRequests(clientID, l.Config.MaxIPRequests)
	case CHECK_API_KEY_ONLY:
		return l.checkAPIKeyOnly(apiKeyID)
	default: // CHECK_IP_OR_API_KEY
		return l.checkIPOrAPIKey(clientID, apiKeyID)
	}
}

func (l *Limiter) checkClientRequests(clientID string, maxRequests int) (bool, error) {
	if clientID == "" {
		return false, ErrInvalidClient
	}

	client := l.Repository.Client(clientID)

	if client != nil {
		if !client.Blocked {
			if client.CurrentRequests < maxRequests {
				client.CurrentRequests++
				client.TTL = l.Config.RequestsLimitInterval
				l.Repository.SaveClient(*client)

				log.Printf("---------Client: %s | Requests Current/Max: %v/%v", clientID, client.CurrentRequests, maxRequests)
				return true, nil
			}

			//apply block if client tries to access after limit is reached out
			client.Blocked = true
			client.TTL = l.Config.ClientBlockTime
			l.Repository.SaveClient(*client)
		}
		log.Printf("---------Client: %s blocked for %v seconds", clientID, l.Config.ClientBlockTime)
		return false, ErrMaxNumberRequestsReached
	} else {
		client = &Client{
			ID:              clientID,
			CurrentRequests: 1,
			TTL:             l.Config.RequestsLimitInterval,
			Blocked:         false,
		}
		l.Repository.SaveClient(*client)
		log.Printf("---------Client: %s | Requests Current/Max: %v/%v", clientID, client.CurrentRequests, maxRequests)
		return true, nil
	}
}

func (l *Limiter) checkAPIKeyOnly(apiKeyID string) (bool, error) {
	var apiKey *APIKey

	if apiKeyID != "" {
		apiKey = l.Repository.ApiKey(apiKeyID)

		if apiKey != nil {
			return l.checkClientRequests(apiKeyID, apiKey.MaxRequests)
		}
	}

	return false, ErrApiKeyNotFound
}

func (l *Limiter) checkIPOrAPIKey(clientID, apiKeyID string) (bool, error) {
	var apiKey *APIKey

	if apiKeyID != "" {
		apiKey = l.Repository.ApiKey(apiKeyID)

		if apiKey != nil {
			return l.checkClientRequests(apiKeyID, apiKey.MaxRequests)
		}
	}

	return l.checkClientRequests(clientID, l.Config.MaxIPRequests)
}

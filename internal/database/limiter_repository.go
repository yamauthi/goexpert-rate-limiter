package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/yamauthi/goexpert-rate-limiter/internal/limiter"
)

const KEYSPACE_API_KEY = "apiKey"
const KEYSPACE_CLIENT = "client"

type RedisLimiterRepository struct {
	ctx   context.Context
	redis *redis.Client
}

func NewRedisLimiterRepository(ctx context.Context, redisClient *redis.Client) *RedisLimiterRepository {
	return &RedisLimiterRepository{
		ctx:   ctx,
		redis: redisClient,
	}
}

func (r *RedisLimiterRepository) ApiKey(id string) *limiter.APIKey {
	res := r.getMap(KEYSPACE_API_KEY, id)
	if len(res) > 0 {
		apiKey := mapToApiKey(res)
		if (apiKey != limiter.APIKey{}) {
			return &apiKey
		}
	}
	return nil
}

func (r *RedisLimiterRepository) Client(id string) *limiter.Client {
	res := r.getMap(KEYSPACE_CLIENT, id)
	if len(res) > 0 {
		client := mapToClient(res)
		if (client != limiter.Client{}) {
			return &client
		}
	}
	return nil
}

func (r *RedisLimiterRepository) SaveApiKey(apiKey limiter.APIKey) {
	if apiKey.ID != "" {
		r.saveMap(KEYSPACE_API_KEY, apiKey.ID, map[string]string{
			"id":          apiKey.ID,
			"maxRequests": strconv.Itoa(apiKey.MaxRequests),
		})
	}
}

func (r *RedisLimiterRepository) SaveClient(client limiter.Client) {
	if client.ID != "" {
		r.saveMap(KEYSPACE_CLIENT, client.ID, map[string]string{
			"id":              client.ID,
			"currentRequests": strconv.Itoa(client.CurrentRequests),
			"blocked":         strconv.FormatBool(client.Blocked),
		})

		r.redis.Expire(r.ctx, generateKey(KEYSPACE_CLIENT, client.ID), client.TTL)
	}
}

func (r *RedisLimiterRepository) getMap(keyspace, key string) map[string]string {
	res, err := r.redis.HGetAll(r.ctx, generateKey(keyspace, key)).Result()
	if err != nil {
		return nil
	}

	return res
}

func (r *RedisLimiterRepository) saveMap(keyspace, key string, valueMap map[string]string) {
	err := r.redis.HSet(r.ctx, generateKey(keyspace, key), valueMap).Err()
	if err != nil {
		panic(err)
	}
}

func generateKey(keyspace, key string) string {
	return fmt.Sprintf("%s:%s", keyspace, key)
}

func mapToApiKey(res map[string]string) limiter.APIKey {
	maxRequests, err := strconv.Atoi(res["maxRequests"])
	if err != nil {
		return limiter.APIKey{}
	}

	return limiter.APIKey{
		ID:          res["id"],
		MaxRequests: maxRequests,
	}
}

func mapToClient(res map[string]string) limiter.Client {
	currentRequests, err := strconv.Atoi(res["currentRequests"])
	if err != nil {
		return limiter.Client{}
	}

	blocked, err := strconv.ParseBool(res["blocked"])
	if err != nil {
		return limiter.Client{}
	}

	return limiter.Client{
		ID:              res["id"],
		CurrentRequests: currentRequests,
		Blocked:         blocked,
	}
}

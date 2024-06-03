package database_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/yamauthi/goexpert-rate-limiter/database"
	"github.com/yamauthi/goexpert-rate-limiter/limiter"
)

type RedisLimiterRepositoryTestSuite struct {
	suite.Suite
	RedisClient *redis.Client
	Repository  *database.RedisLimiterRepository
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(RedisLimiterRepositoryTestSuite))
}

func (suite *RedisLimiterRepositoryTestSuite) SetupTest() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "redis-passw0rd",
	})
	client.FlushAll(context.Background())
	suite.RedisClient = client
	suite.Repository = database.NewRedisLimiterRepository(context.Background(), client)
}

func (suite *RedisLimiterRepositoryTestSuite) TearDownTest() {
	suite.RedisClient.FlushAll(context.Background())
}

func (suite *RedisLimiterRepositoryTestSuite) TestRedisLimiterRepository_ApiKey() {
	type TestCase struct {
		Name     string
		Input    string
		Expected limiter.APIKey
	}

	testCases := []TestCase{
		{
			Name:     "Should return nil if apiKeyID is empty",
			Input:    "",
			Expected: limiter.APIKey{},
		},
		{
			Name:     "Should return nil if apiKeyID does not exist",
			Input:    "Inexistent key",
			Expected: limiter.APIKey{},
		},
	}

	testApiKeys := []limiter.APIKey{
		{
			ID:          "secretKey1",
			MaxRequests: 10,
		},
		{
			ID:          "secretKey2",
			MaxRequests: 15,
		},
		{
			ID:          "secretKey3",
			MaxRequests: 2,
		},
	}

	for _, k := range testApiKeys {
		suite.RedisClient.HSet(
			context.Background(),
			fmt.Sprintf("%s:%s", database.KEYSPACE_API_KEY, k.ID),
			map[string]string{
				"id":          k.ID,
				"maxRequests": strconv.Itoa(k.MaxRequests),
			})

		testCases = append(testCases, TestCase{
			Name:     fmt.Sprint("Should return apiKey register with ID ", k.ID),
			Input:    k.ID,
			Expected: k,
		})
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			apiKey := suite.Repository.ApiKey(t.Input)
			if (t.Expected != limiter.APIKey{}) {
				suite.Equal(t.Expected, *apiKey)
			} else {
				suite.Nil(apiKey)
			}

		})
	}
}

func (suite *RedisLimiterRepositoryTestSuite) TestRedisLimiterRepository_Client() {
	type TestCase struct {
		Name     string
		Input    string
		Expected limiter.Client
	}

	testCases := []TestCase{
		{
			Name:     "Should return nil if clientID is empty",
			Input:    "",
			Expected: limiter.Client{},
		},
		{
			Name:     "Should return nil if clientID does not exist",
			Input:    "Inexistent clientID",
			Expected: limiter.Client{},
		},
	}

	testClients := []limiter.Client{
		{
			ID:              "192.168.0.1",
			CurrentRequests: 10,
			TTL:             time.Second,
			Blocked:         false,
		},
		{
			ID:              "192.168.10.2",
			CurrentRequests: 2,
			TTL:             time.Second * 2,
			Blocked:         false,
		},
		{
			ID:              "secretKey1",
			CurrentRequests: 20,
			TTL:             time.Second * 6,
			Blocked:         true,
		},
	}

	for _, c := range testClients {
		suite.RedisClient.HSet(
			context.Background(),
			fmt.Sprintf("%s:%s", database.KEYSPACE_CLIENT, c.ID),
			map[string]string{
				"id":              c.ID,
				"currentRequests": strconv.Itoa(c.CurrentRequests),
				"blocked":         strconv.FormatBool(c.Blocked),
			})
		suite.RedisClient.Expire(
			context.Background(),
			fmt.Sprintf("%s:%s", database.KEYSPACE_CLIENT, c.ID),
			c.TTL,
		)
		testCases = append(testCases, TestCase{
			Name:     fmt.Sprint("Should return client register with ID ", c.ID),
			Input:    c.ID,
			Expected: c,
		})
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			client := suite.Repository.Client(t.Input)
			if (t.Expected != limiter.Client{}) {
				suite.NotNil(client)
				suite.Equal(t.Expected.ID, client.ID)
				suite.Equal(t.Expected.CurrentRequests, client.CurrentRequests)
				suite.Equal(t.Expected.Blocked, client.Blocked)
			} else {
				suite.Nil(client)
			}
		})
	}

	suite.Run("Should return nil after TTL expired", func() {
		const sleepFor = time.Second * 2
		time.Sleep(sleepFor)
		client0 := suite.Repository.Client(testClients[0].ID)
		if testClients[0].TTL <= sleepFor {
			suite.Nil(client0)
		} else {
			suite.NotNil(client0)
			suite.Equal(testClients[0].ID, client0.ID)
			suite.Equal(testClients[0].CurrentRequests, client0.CurrentRequests)
			suite.Equal(testClients[0].Blocked, client0.Blocked)
		}

		client2 := suite.Repository.Client(testClients[2].ID)
		if testClients[2].TTL <= sleepFor {
			suite.Nil(client2)
		} else {
			suite.NotNil(client2)
			suite.Equal(testClients[2].ID, client2.ID)
			suite.Equal(testClients[2].CurrentRequests, client2.CurrentRequests)
			suite.Equal(testClients[2].Blocked, client2.Blocked)
		}
	})

}

func (suite *RedisLimiterRepositoryTestSuite) TestRedisLimiterRepository_SaveApiKey() {
	type TestCase struct {
		Name     string
		Input    limiter.APIKey
		Expected map[string]string
	}

	testCases := []TestCase{
		{
			Name: "Should create key if not exists id SecretApiKey1",
			Input: limiter.APIKey{
				ID:          "SecretApiKey1",
				MaxRequests: 10,
			},
			Expected: map[string]string{
				"id":          "SecretApiKey1",
				"maxRequests": "10",
			},
		},
		{
			Name: "Should create key if not exists id DifferentApiKey2",
			Input: limiter.APIKey{
				ID:          "DifferentApiKey2",
				MaxRequests: 100,
			},
			Expected: map[string]string{
				"id":          "DifferentApiKey2",
				"maxRequests": "100",
			},
		},
		{
			Name: "Should ovewrite key if it already exists",
			Input: limiter.APIKey{
				ID:          "SecretApiKey1",
				MaxRequests: 200,
			},
			Expected: map[string]string{
				"id":          "SecretApiKey1",
				"maxRequests": "200",
			},
		},
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			suite.Repository.SaveApiKey(t.Input)
			result := suite.RedisClient.HGetAll(
				context.Background(),
				fmt.Sprintf("%s:%s", database.KEYSPACE_API_KEY, t.Input.ID),
			)

			suite.NotNil(result)
			suite.Equal(t.Expected, result.Val())
		})
	}
}

func (suite *RedisLimiterRepositoryTestSuite) TestRedisLimiterRepository_SaveClient() {
	type TestCase struct {
		Name     string
		Input    limiter.Client
		Expected map[string]string
	}

	testCases := []TestCase{
		{
			Name: "Should create client if not exists id 192.168.0.1",
			Input: limiter.Client{
				ID:              "192.168.0.1",
				CurrentRequests: 5,
				TTL:             time.Second,
				Blocked:         false,
			},
			Expected: map[string]string{
				"id":              "192.168.0.1",
				"currentRequests": "5",
				"blocked":         "false",
			},
		},
		{
			Name: "Should create client if not exists id ApiKey2",
			Input: limiter.Client{
				ID:              "ApiKey2",
				CurrentRequests: 100,
				TTL:             time.Second,
				Blocked:         true,
			},
			Expected: map[string]string{
				"id":              "ApiKey2",
				"currentRequests": "100",
				"blocked":         "true",
			},
		},
		{
			Name: "Should ovewrite client if it already exists",
			Input: limiter.Client{
				ID:              "192.168.0.1",
				CurrentRequests: 50,
				TTL:             time.Second * 10,
				Blocked:         true,
			},
			Expected: map[string]string{
				"id":              "192.168.0.1",
				"currentRequests": "50",
				"blocked":         "true",
			},
		},
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			suite.Repository.SaveClient(t.Input)
			result := suite.RedisClient.HGetAll(
				context.Background(),
				fmt.Sprintf("%s:%s", database.KEYSPACE_CLIENT, t.Input.ID),
			)

			suite.NotNil(result)
			suite.Equal(t.Expected, result.Val())
		})
	}
}

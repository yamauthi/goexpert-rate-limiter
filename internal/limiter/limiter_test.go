package limiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yamauthi/goexpert-rate-limiter/internal/limiter"
)

type MockLimiterRepository struct {
	mock.Mock
}

func (r *MockLimiterRepository) ApiKey(id string) *limiter.APIKey {
	args := r.Called(id)
	return args.Get(0).(*limiter.APIKey)
}

func (r *MockLimiterRepository) Client(id string) *limiter.Client {
	args := r.Called(id)
	return args.Get(0).(*limiter.Client)
}

func (r *MockLimiterRepository) SaveClient(client limiter.Client) {
	r.Called(client)
}

func (r *MockLimiterRepository) SaveApiKey(apiKey limiter.APIKey) {
	r.Called(apiKey)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(LimiterTestSuite))
}

const ClientBlockTime = 3
const MaxRequests = 3

type LimiterTestSuite struct {
	suite.Suite
	MockLimiterRepository *MockLimiterRepository
	Config                limiter.LimiterConfig
	Limiter               *limiter.Limiter
}

func (suite *LimiterTestSuite) SetupTest() {
	suite.MockLimiterRepository = &MockLimiterRepository{}
	suite.Config = limiter.LimiterConfig{
		ClientBlockTime:       time.Second * ClientBlockTime,
		RequestsLimitInterval: limiter.REQUESTS_PER_SECOND,
		MaxIPRequests:         MaxRequests,
	}
}

type TestCaseInput struct {
	ClientID string
	ApiKeyID string
}

type TestCaseExpected struct {
	IsAllowed   bool
	Error       error
	SavedClient limiter.Client
}

type TestCaseReturn struct {
	Client *limiter.Client
	ApiKey *limiter.APIKey
}

type TestCase struct {
	Name     string
	Input    TestCaseInput
	Return   TestCaseReturn
	Expected TestCaseExpected
}

func (suite *LimiterTestSuite) TestLimiter_AllowRequest_CheckIpOnly() {
	suite.Config.ClientCheckType = limiter.CHECK_IP_ONLY
	suite.Limiter = limiter.NewLimiter(suite.Config, suite.MockLimiterRepository)

	testCases := []TestCase{
		{
			Name: "Should not allow and return Invalid Client error if clientID is empty",
			Input: TestCaseInput{
				ClientID: "",
			},
			Return: TestCaseReturn{},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrInvalidClient,
			},
		},
		{
			Name: "Should allow with no error if clientId is not empty and does not exist in limiter repository",
			Input: TestCaseInput{
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should allow with no error if an existing client has CurrentRequests within the limit",
			Input: TestCaseInput{
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: &limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 2,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should allow with no error if an existing client is reaching the limit",
			Input: TestCaseInput{
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: &limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 2,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 3,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false, // Should not block if it is the last request within limit
				},
			},
		},
		{
			Name: "Should not allow with MaxNumberRequestsReached error if an existing client requests after limit is reached out, applying block time",
			Input: TestCaseInput{
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: &limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 3,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrMaxNumberRequestsReached,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 3,
					TTL:             suite.Config.ClientBlockTime,
					Blocked:         true,
				},
			},
		},
		{
			Name: "Should not allow with MaxNumberRequestsReached error a blocked client",
			Input: TestCaseInput{
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: &limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 3,
					TTL:             suite.Config.ClientBlockTime,
					Blocked:         true,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrMaxNumberRequestsReached,
			},
		},
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			suite.MockLimiterRepository = &MockLimiterRepository{}
			suite.Limiter.Repository = suite.MockLimiterRepository

			suite.MockLimiterRepository.Mock.On("Client", t.Input.ClientID).Return(t.Return.Client)
			suite.MockLimiterRepository.Mock.On("SaveClient", t.Expected.SavedClient)

			allowed, err := suite.Limiter.AllowRequest(t.Input.ClientID, t.Input.ApiKeyID)
			suite.Equal(t.Expected.IsAllowed, allowed)
			suite.Equal(t.Expected.Error, err)

			if t.Input.ClientID != "" {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "Client", 1)
			}

			if (t.Expected.SavedClient != limiter.Client{}) {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "SaveClient", 1)
			}
		})
	}
}

func (suite *LimiterTestSuite) TestLimiter_AllowRequest_CheckAPIKeyOnly() {
	suite.Config.ClientCheckType = limiter.CHECK_API_KEY_ONLY
	suite.Limiter = limiter.NewLimiter(suite.Config, suite.MockLimiterRepository)
	testApiKey := limiter.APIKey{
		ID:          "SecretKey123",
		MaxRequests: 5,
	}
	testCases := []TestCase{
		{
			Name: "Should not allow and return ApiKeyNotFound error if apiKey is empty",
			Input: TestCaseInput{
				ApiKeyID: "",
			},
			Return: TestCaseReturn{},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrApiKeyNotFound,
			},
		},
		{
			Name: "Should not allow and return ApiKeyNotFound error if apiKey does not exist in limiter repository",
			Input: TestCaseInput{
				ApiKeyID: "Inexistent API Key",
			},
			Return: TestCaseReturn{
				ApiKey: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrApiKeyNotFound,
			},
		},
		{
			Name: "Should allow and return no error if an existing APIKey has no entry as client",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should allow and return no error if an existing APIKey has CurrentRequests within the limit",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: &limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests - 2,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests - 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should allow with no error if an existing APIKey is reaching the limit",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: &limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests - 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false, // Should not block if it is the last request within limit
				},
			},
		},
		{
			Name: "Should not allow with MaxNumberRequestsReached error if an existing APIKey requests after limit is reached out, applying block time",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: &limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrMaxNumberRequestsReached,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests,
					TTL:             suite.Config.ClientBlockTime,
					Blocked:         true,
				},
			},
		},
		{
			Name: "Should not allow with MaxNumberRequestsReached error a blocked client",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: &limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests,
					TTL:             suite.Config.ClientBlockTime,
					Blocked:         true,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrMaxNumberRequestsReached,
			},
		},
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			suite.MockLimiterRepository = &MockLimiterRepository{}
			suite.Limiter.Repository = suite.MockLimiterRepository

			suite.MockLimiterRepository.Mock.On("ApiKey", t.Input.ApiKeyID).Return(t.Return.ApiKey)
			suite.MockLimiterRepository.Mock.On("Client", t.Input.ApiKeyID).Return(t.Return.Client)
			suite.MockLimiterRepository.Mock.On("SaveClient", t.Expected.SavedClient)

			allowed, err := suite.Limiter.AllowRequest(t.Input.ClientID, t.Input.ApiKeyID)
			suite.Equal(t.Expected.IsAllowed, allowed)
			suite.Equal(t.Expected.Error, err)

			if t.Input.ApiKeyID != "" {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "ApiKey", 1)
			}

			if t.Input.ClientID != "" {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "Client", 1)
			}

			if (t.Expected.SavedClient != limiter.Client{}) {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "SaveClient", 1)
			}
		})
	}
}

func (suite *LimiterTestSuite) TestLimiter_AllowRequest_CheckIpOrAPIKey() {
	suite.Config.ClientCheckType = limiter.CHECK_IP_OR_API_KEY
	suite.Limiter = limiter.NewLimiter(suite.Config, suite.MockLimiterRepository)
	testApiKey := limiter.APIKey{
		ID:          "SecretKey123",
		MaxRequests: 5,
	}
	testCases := []TestCase{
		{
			Name: "Should not allow and return InvalidClient error if client and apiKey is empty",
			Input: TestCaseInput{
				ApiKeyID: "",
				ClientID: "",
			},
			Return: TestCaseReturn{},
			Expected: TestCaseExpected{
				IsAllowed: false,
				Error:     limiter.ErrInvalidClient,
			},
		},
		{
			Name: "Should allow with no error if apiKey is empty but clientId is not empty and does not exist in limiter repository",
			Input: TestCaseInput{
				ApiKeyID: "",
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				Client: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should use apiKey as client if ClientID IP is empty and there is an existing APIKey with passed ID",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
				ClientID: "",
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should use Client IP validation if APIKey does not exists",
			Input: TestCaseInput{
				ApiKeyID: "Inexisting Key",
				ClientID: "192.168.0.1",
			},
			Return: TestCaseReturn{
				ApiKey: nil,
				Client: nil,
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              "192.168.0.1",
					CurrentRequests: 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
		{
			Name: "Should use apiKey as client even if there is already a registered IP for ClientID",
			Input: TestCaseInput{
				ApiKeyID: "SecretKey123",
				ClientID: "192.168.0.1", //API Key should be considered even if there is already a register for IP
			},
			Return: TestCaseReturn{
				ApiKey: &testApiKey,
				Client: &limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests - 2,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
			Expected: TestCaseExpected{
				IsAllowed: true,
				Error:     nil,
				SavedClient: limiter.Client{
					ID:              testApiKey.ID,
					CurrentRequests: testApiKey.MaxRequests - 1,
					TTL:             suite.Config.RequestsLimitInterval,
					Blocked:         false,
				},
			},
		},
	}

	for _, t := range testCases {
		suite.Run(t.Name, func() {
			suite.MockLimiterRepository = &MockLimiterRepository{}
			suite.Limiter.Repository = suite.MockLimiterRepository

			suite.MockLimiterRepository.Mock.On("ApiKey", t.Input.ApiKeyID).Return(t.Return.ApiKey)
			suite.MockLimiterRepository.Mock.On("Client", t.Input.ApiKeyID).Return(t.Return.Client)
			suite.MockLimiterRepository.Mock.On("Client", t.Input.ClientID).Return(t.Return.Client)
			suite.MockLimiterRepository.Mock.On("SaveClient", t.Expected.SavedClient)

			allowed, err := suite.Limiter.AllowRequest(t.Input.ClientID, t.Input.ApiKeyID)
			suite.Equal(t.Expected.IsAllowed, allowed)
			suite.Equal(t.Expected.Error, err)

			if t.Input.ApiKeyID != "" {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "ApiKey", 1)
			}

			if t.Input.ClientID != "" {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "Client", 1)
			}

			if (t.Expected.SavedClient != limiter.Client{}) {
				suite.MockLimiterRepository.AssertNumberOfCalls(suite.T(), "SaveClient", 1)
			}
		})
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yamauthi/goexpert-rate-limiter/internal/configs"
	"github.com/yamauthi/goexpert-rate-limiter/internal/database"
	"github.com/yamauthi/goexpert-rate-limiter/internal/limiter"
	"github.com/yamauthi/goexpert-rate-limiter/internal/web/middleware"
)

func main() {
	conf, err := configs.LoadConfig(".")
	if err != nil {
		log.Fatalf("error on config file loading: %s", err.Error())
	}

	redis := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.DBHost, conf.DBPort),
		Password: conf.DBPassword,
	})

	repository := database.NewRedisLimiterRepository(context.Background(), redis)

	ratelimiter := limiter.NewLimiter(
		limiter.LimiterConfig{
			ClientCheckType:       conf.DefaultLimitType,
			ClientBlockTime:       time.Second * time.Duration(conf.DefaultClientBlockTime),
			MaxIPRequests:         conf.DefaultRequestsLimit,
			RequestsLimitInterval: limiter.REQUESTS_PER_SECOND,
		},
		repository,
	)

	mux := http.NewServeMux()
	mux.Handle("/", middleware.NewLimiterMiddleware(ratelimiter).Limit(http.HandlerFunc(handler)))

	ratelimiter.Config.ClientCheckType = limiter.CHECK_IP_ONLY
	mux.Handle("/ip", middleware.NewLimiterMiddleware(ratelimiter).Limit(http.HandlerFunc(handler)))

	ratelimiter.Config.ClientCheckType = limiter.CHECK_IP_ONLY
	mux.Handle("/apikey", middleware.NewLimiterMiddleware(ratelimiter).Limit(http.HandlerFunc(handler)))

	ratelimiter.Config.ClientCheckType = limiter.CHECK_IP_OR_API_KEY
	mux.Handle("/ip-apikey", middleware.NewLimiterMiddleware(ratelimiter).Limit(http.HandlerFunc(handler)))

	log.Println("server running on port 8080")
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("ping %s", r.URL.Path)))
}

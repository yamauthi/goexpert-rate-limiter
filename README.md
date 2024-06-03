# GoExpert Postgraduate program - Rate Limiter challenge <img src="https://www.svgrepo.com/show/353830/gopher.svg" width="40" height="40">

## Challenge description
### Objective
Develop a rate limiter in Go that can be configured to limit the maximum number of requests per second based on a specific IP address or an access token.

### Description
The goal of this challenge is to create a rate limiter in Go that can be used to control the traffic of requests to a web service. The rate limiter should be able to limit the number of requests based on two criteria:

- **IP Address:** The rate limiter should restrict the number of requests received from a single IP address within a defined time interval.
- **Access Token:** The rate limiter should also be able to limit requests based on a unique access token, allowing different expiration time limits for different tokens. The Token should be provided in the header in the following format:
  `API_KEY: <TOKEN>`
  The access token limit settings should override those of the IP. For example, if the limit per IP is 10 req/s and that of a specific token is 100 req/s, the rate limiter should use the token information.

### Requirements
- The rate limiter should be able to work as a middleware that is injected into the web server.
- The rate limiter should allow the configuration of the maximum number of requests allowed per second.
- The rate limiter should have the option to choose the blocking time for the IP or Token if the number of requests has been exceeded.
- Limit settings should be done via environment variables or in a ".env" file in the root folder.
- It should be possible to configure the rate limiter for both IP and access token limitation.
- The system should respond appropriately when the limit is exceeded:
    - HTTP Code: 429
    - Message: you have reached the maximum number of requests or actions allowed within a certain time frame
- All limiter information should be stored and queried from a Redis database. You can use docker-compose to spin up Redis.
- Create a "strategy" that allows easily swapping Redis for another persistence mechanism.
- The limiter logic should be separated from the middleware.

### Examples
- **IP Limitation:** Suppose the rate limiter is configured to allow a maximum of 5 requests per second per IP. If IP 192.168.1.1 sends 6 requests in one second, the sixth request should be blocked.
- **Token Limitation:** If a token abc123 has a configured limit of 10 requests per second and sends 11 requests within that interval, the eleventh should be blocked.
  In both cases above, subsequent requests can only be made after the total expiration time has passed. For example, if the expiration time is 5 minutes, a specific IP can only make new requests after the 5 minutes have passed.

### Tips
- Test your rate limiter under different load conditions to ensure it functions as expected under high traffic situations.

### Delivery
- The complete source code of the implementation.
- Documentation explaining how the rate limiter works and how it can be configured.
- Automated tests demonstrating the effectiveness and robustness of the rate limiter.
- Use docker/docker-compose so we can test your application.
- The web server should respond on port 8080.

### Improvement for this code
- Benchmark tests
- Dependency injection
- Load tests
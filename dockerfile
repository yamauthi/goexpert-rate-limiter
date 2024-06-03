FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN cd cmd && CGO_ENABLED=0 go build -o rate-limiter .
RUN chmod +x /app/cmd/rate-limiter

FROM scratch

WORKDIR /app
COPY --from=builder /app/cmd/rate-limiter .
COPY --from=builder /app/cmd/.env .

ENTRYPOINT ["./rate-limiter"]
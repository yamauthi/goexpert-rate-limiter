FROM golang:1.22.3-alpine as builder
WORKDIR /go-app
COPY . .
RUN go build -o rate-limiter ./cmd
RUN chmod +x rate-limiter

FROM scratch
WORKDIR /go-app
COPY --from=builder /go-app/cmd/.env .
COPY --from=builder /go-app/rate-limiter .

ENTRYPOINT ["/go-app/rate-limiter"]
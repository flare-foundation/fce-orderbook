# Build stage — context must be tee/ so the replace directive resolves.
FROM golang:1.25.1-alpine AS builder

RUN apk add --no-cache git
WORKDIR /build

COPY tee-node/ ./tee-node/
COPY extension-examples/orderbook/ ./extension-examples/orderbook/

WORKDIR /build/extension-examples/orderbook
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/extension-tee ./cmd/docker
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/types-server ./cmd/types-server

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/extension-tee ./
COPY --from=builder /app/types-server ./
COPY extension-examples/orderbook/config/pairs.json ./config/pairs.json

LABEL "tee.launch_policy.allow_env_override"="LOG_LEVEL,PROXY_URL,INITIAL_OWNER,EXTENSION_ID"

ENV MODE=0 CONFIG_PORT=5501 SIGN_PORT=7701 EXTENSION_PORT=7702
EXPOSE 5501 7701 7702
CMD ["./extension-tee"]

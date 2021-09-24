FROM golang:1.17.1-alpine AS builder

WORKDIR /app
ENV GOCACHE=/tmp/go/cache

COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache CGO_ENABLED=0 go build -o /app/switchbot_exporter -ldflags "-s -w"

FROM scratch

WORKDIR /app

COPY --from=builder /app/switchbot_exporter .

ENTRYPOINT ["/app/switchbot_exporter"]

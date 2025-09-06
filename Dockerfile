ARG APP_VERSION=dev
ARG APP_REVISION=snapshot

FROM --platform=$BUILDPLATFORM golang:1-alpine AS builder

WORKDIR /work
ENV CGO_ENABLED=0

RUN apk add --update --no-cache git

COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG APP_VERSION
ARG APP_REVISION

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -o /app/app -ldflags "-s -w -X main.version=$APP_VERSION -X main.revision=$APP_REVISION" .

FROM alpine:3 AS base
WORKDIR /app

RUN apk add --no-cache --update tzdata ca-certificates && \
    update-ca-certificates

COPY --from=builder /app/app ./

ENTRYPOINT ["/app/app"]

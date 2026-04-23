FROM golang:1.26-alpine AS builder

ENV GOOS=linux

WORKDIR /build

RUN apk add --no-cache make git build-base

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG GITHASH=docker
ARG BUILD_DATE

RUN --mount=type=cache,target=/root/.cache/go-build \
    make build \
        GITHASH="${GITHASH}" \
        BUILD_DATE="${BUILD_DATE:-$(date +'%Y-%m-%d %H:%M:%S')}"

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/bot .

ENTRYPOINT ["./bot"]

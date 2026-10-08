# Go 1.26.8; pinned multi-platform image for reproducible local development.
FROM golang:1.26-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628
RUN apk add --no-cache build-base
WORKDIR /app
ENV GOCACHE=/go/build-cache
COPY go.mod go.sum ./
RUN go mod download
CMD ["go", "run", "./cmd/multi-chain-wallet"]

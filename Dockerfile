# Go 1.26.10; pinned multi-platform image for reproducible local development.
FROM golang:1.26.10-alpine@sha256:5cb67f19c5e7994dcb80c074f7cfca8aeeada45bbdca035f73df269cefdcce61
RUN apk add --no-cache build-base
WORKDIR /app
ENV GOCACHE=/go/build-cache
COPY go.mod go.sum ./
RUN go mod download
CMD ["go", "run", "./cmd/multi-chain-wallet"]

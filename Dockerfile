# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/grader-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed-uas ./cmd/seed_uas

FROM alpine:3.21

# The API calls the host Docker daemon to run each Python submission.
RUN apk add --no-cache ca-certificates docker-cli

WORKDIR /app
COPY --from=builder /out/grader-api ./grader-api
COPY --from=builder /out/seed ./seed
COPY --from=builder /out/seed-uas ./seed-uas

EXPOSE 8080

CMD ["./grader-api"]

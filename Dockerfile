# syntax=docker/dockerfile:1

# ---- builder ----
FROM golang:latest AS builder

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Fully static binary (no CGO), stripped, reproducible path prefix.
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/go2internetarchive .

# ---- runtime ----
# Minimal image; ca-certificates are required for HTTPS calls to s3.us.archive.org.
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/go2internetarchive /app/go2internetarchive

# The server reads ./router.json and writes to ./uploads (relative to WORKDIR).
# Both are provided at runtime via volumes (see compose.yml.example).
RUN mkdir -p /app/uploads

EXPOSE 8080

ENTRYPOINT ["/app/go2internetarchive"]

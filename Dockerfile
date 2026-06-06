# Build stage
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm install --legacy-peer-deps
COPY web/ ./
RUN npm run build

# Go build stage
FROM golang:1.26-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-builder /app/web/dist /app/web/dist
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/prasenjeet-symon/ogcode/internal/cli.version=$(git describe --tags --always) -X github.com/prasenjeet-symon/ogcode/internal/cli.commit=$(git rev-parse --short HEAD) -X github.com/prasenjeet-symon/ogcode/internal/cli.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o ogcode .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=go-builder /app/ogcode /usr/local/bin/ogcode
EXPOSE 9595
ENTRYPOINT ["ogcode"]
CMD ["serve"]

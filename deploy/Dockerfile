# syntax=docker/dockerfile:1.7

FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist/ ./backend/internal/http/dist/
ARG VERSION=dev
RUN cd backend \
    && CGO_ENABLED=0 go build -trimpath \
       -ldflags="-s -w -X github.com/bublya/baby-tracker/backend/internal/http.version=${VERSION}" \
       -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app \
    && mkdir -p /data && chown app:app /data
USER app
WORKDIR /app
COPY --from=backend /out/server /app/server
ENV DATA_DIR=/data HTTP_ADDR=:8080 TZ=UTC
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/server"]

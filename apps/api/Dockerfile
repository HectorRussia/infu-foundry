FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /migrate ./cmd/migrate

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /server /usr/local/bin/server
COPY --from=build /migrate /usr/local/bin/migrate
WORKDIR /app
COPY migrations ./migrations
USER app
ENV PORT=8080 APP_ENV=production
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]

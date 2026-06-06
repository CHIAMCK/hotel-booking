# Hotel Booking API

A small Go API built with [Gin](https://gin-gonic.com/).

## Requirements

- Go 1.21+

## Run

```sh
go mod tidy
go run .
```

The API starts on `http://localhost:8080` by default. Set `PORT` to use another port.

## Routes

- `GET /` - welcome response
- `GET /health` - health check
- `GET /api/v1/rooms` - sample room listing

## Test

```sh
go test ./...
```

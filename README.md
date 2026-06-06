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
- `GET /api/v1/room-categories` - search available room categories by date range and guest count

### Search room categories

```
GET /api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2
```

Query parameters:

- `hotel_id` (required) - hotel to search
- `check_in` (required) - check-in date (`YYYY-MM-DD`)
- `check_out` (required) - check-out date (`YYYY-MM-DD`, must be after check-in)
- `guests` (required) - number of guests
- `limit` (optional) - max results, default `10`, max `10`

Example:

```sh
curl "http://localhost:8080/api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2"
```

## Database

Start Postgres with seed data:

```sh
docker compose up -d
```

If the database volume already exists, recreate it to load seed data:

```sh
docker compose down -v && docker compose up -d
```

## Test

```sh
go test ./...
```

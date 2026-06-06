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
- `POST /api/v1/bookings` - create a room booking (Redis lock + DB constraints + idempotent)

### Create booking

```
POST /api/v1/bookings
Content-Type: application/json

{
  "room_id": 2,
  "customer_id": 1,
  "check_in": "2026-07-01",
  "check_out": "2026-07-06"
}
```

The server derives an idempotency key from `room_id`, `customer_id`, `check_in`, and `check_out` (after parsing dates). Retrying the same JSON returns the original booking without creating a duplicate. Successful creates are recorded in **Redis** for 7 days (not in Postgres); if Redis is unavailable the API may return `503 Service Unavailable`.

Body fields:

- `room_id` (required) - room to book
- `customer_id` (required) - customer making the booking
- `check_in` (required) - check-in date (`YYYY-MM-DD`)
- `check_out` (required) - check-out date (`YYYY-MM-DD`, must be after check-in)

Responses:

- `201 Created` - new booking created
- `200 OK` - idempotent replay of an existing booking (same room, customer, and dates as a prior successful create)
- `503 Service Unavailable` - Redis idempotency cache error (safe to retry the same request)
- `409 Conflict` - room unavailable, overlapping booking, or lock not acquired

Double-booking protection:

1. **Redis lock** - serializes concurrent booking attempts per room
2. **Postgres exclusion constraint** - rejects overlapping `pending`/`confirmed` bookings for the same room
3. **Idempotency cache (Redis)** - replay detection for safe HTTP retries without duplicate bookings

Example:

```sh
curl -X POST "http://localhost:8080/api/v1/bookings" \
  -H "Content-Type: application/json" \
  -d '{"room_id":2,"customer_id":1,"check_in":"2026-07-01","check_out":"2026-07-06"}'
```

### Search room categories

```
GET /api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2
```

Query parameters:

- `hotel_id` (required) - hotel to search
- `check_in` (required) - check-in date (`YYYY-MM-DD`)
- `check_out` (required) - check-out date (`YYYY-MM-DD`, must be after check-in)
- `guests` (required) - number of guests
- `page` (optional) - page number, default `1`
- `limit` (optional) - results per page, default `10`, max `10`

Example:

```sh
curl "http://localhost:8080/api/v1/room-categories?hotel_id=1&check_in=2026-06-10&check_out=2026-06-15&guests=2&page=2&limit=10"
```

Response includes pagination metadata:

```json
{
  "categories": [...],
  "pagination": {
    "page": 2,
    "limit": 10,
    "total": 15,
    "total_pages": 2
  }
}
```

## Database

Start Postgres and Redis with seed data:

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
